package investigation

import (
	"context"
	"fmt"
	"strconv"
	"time"

	deployments "observability-mcp/internal/deployments"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
)

type Service struct {
	prometheus  *prom.Client
	loki        *loki.Client
	deployments *deployments.Client
}

func NewService(
	prometheusClient *prom.Client,
	lokiClient *loki.Client,
	deploymentClient *deployments.Client,
) *Service {
	return &Service{
		prometheus:  prometheusClient,
		loki:        lokiClient,
		deployments: deploymentClient,
	}
}

type Investigation struct {
	Service string `json:"service"`

	Health struct {
		Status string `json:"status"`
	} `json:"health"`

	Metrics struct {
		RequestRate *float64 `json:"request_rate"`
		ErrorRate   *float64 `json:"error_rate"`
		P95Latency  *float64 `json:"p95_latency_seconds"`
		P99Latency  *float64 `json:"p99_latency_seconds"`
	} `json:"metrics"`

	RecentErrors      []Error            `json:"recent_errors"`
	RecentDeployments []Deployment       `json:"recent_deployments"`
	Dependencies      []DependencyStatus `json:"dependencies"`
	Impact            ImpactSummary      `json:"impact"`
	Diagnosis         *Diagnosis         `json:"diagnosis,omitempty"`
	Timeline          []IncidentEvent    `json:"timeline"`
}

type Error struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type Deployment struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

func (s *Service) Investigate(
	ctx context.Context,
	service string,
	duration time.Duration,
) (*Investigation, error) {

	if service != "orderflow" {
		return nil, fmt.Errorf(
			"unsupported service: %s",
			service,
		)
	}

	if duration <= 0 {
		return nil, fmt.Errorf(
			"duration must be greater than zero",
		)
	}

	result := &Investigation{
		Service:           service,
		RecentErrors:      []Error{},
		RecentDeployments: []Deployment{},
		Dependencies:      []DependencyStatus{},
		Timeline:          []IncidentEvent{},
		Impact: ImpactSummary{
			AffectedPaths: []string{},
			Endpoints:     []EndpointImpact{},
		},
	}

	window := formatDuration(duration)
	end := time.Now()
	start := end.Add(-duration)

	recentWindowDuration := 2 * time.Minute
	if duration < recentWindowDuration {
		recentWindowDuration = duration / 2
	}
	recentCutoff := end.Add(-recentWindowDuration)

	// --------------------------------------------------
	// 1. Service health
	// --------------------------------------------------

	healthResult, err := s.prometheus.Query(
		ctx,
		`up{job="orderflow"}`,
	)

	if err != nil {
		result.Health.Status = "unknown"
	} else if len(healthResult.Data.Result) > 0 && len(healthResult.Data.Result[0].Value) >= 2 {
		val, parseErr := prom.ExtractScalar(healthResult)
		if parseErr == nil && val != nil {
			if *val == 1.0 {
				result.Health.Status = "healthy"
			} else {
				result.Health.Status = "down"
			}
		} else {
			result.Health.Status = "unknown"
		}
	} else {
		result.Health.Status = "unknown"
	}

	// --------------------------------------------------
	// 2. Request rate
	// --------------------------------------------------

	requestRateQuery := fmt.Sprintf(
		`sum(rate(orderflow_http_requests_total[%s]))`,
		window,
	)

	requestRateResult, err := s.prometheus.Query(
		ctx,
		requestRateQuery,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"request rate query failed: %w",
			err,
		)
	}

	reqRate, err := prom.ExtractScalar(requestRateResult)
	if err != nil {
		return nil, fmt.Errorf("parse request rate: %w", err)
	}
	result.Metrics.RequestRate = reqRate

	// --------------------------------------------------
	// 3. Error rate
	// --------------------------------------------------

	errorRateQuery := fmt.Sprintf(
		`sum(rate(orderflow_http_requests_total{status=~"5.."}[%s])) / sum(rate(orderflow_http_requests_total[%s]))`,
		window,
		window,
	)

	errorRateResult, err := s.prometheus.Query(
		ctx,
		errorRateQuery,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"error rate query failed: %w",
			err,
		)
	}

	errRate, err := prom.ExtractScalar(errorRateResult)
	if err != nil {
		return nil, fmt.Errorf("parse error rate: %w", err)
	}
	result.Metrics.ErrorRate = errRate

	// --------------------------------------------------
	// 4. P95 latency
	// --------------------------------------------------

	p95Query := fmt.Sprintf(
		`histogram_quantile(0.95, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket[%s])))`,
		window,
	)

	p95Result, err := s.prometheus.Query(
		ctx,
		p95Query,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"P95 latency query failed: %w",
			err,
		)
	}

	p95, err := prom.ExtractScalar(p95Result)
	if err != nil {
		return nil, fmt.Errorf("parse P95 latency: %w", err)
	}
	result.Metrics.P95Latency = p95

	// --------------------------------------------------
	// 5. P99 latency
	// --------------------------------------------------

	p99Query := fmt.Sprintf(
		`histogram_quantile(0.99, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket[%s])))`,
		window,
	)

	p99Result, err := s.prometheus.Query(
		ctx,
		p99Query,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"P99 latency query failed: %w",
			err,
		)
	}

	p99, err := prom.ExtractScalar(p99Result)
	if err != nil {
		return nil, fmt.Errorf("parse P99 latency: %w", err)
	}
	result.Metrics.P99Latency = p99

	// --------------------------------------------------
	// 6. Recent errors from Loki
	// --------------------------------------------------

	logQuery := `{service_name="orderflow"} | json | level="ERROR"`

	logResult, err := s.loki.QueryRange(
		ctx,
		logQuery,
		start,
		end,
		50,
	)

	if err == nil && logResult != nil {
		for _, stream := range logResult.Data.Result {
			for _, value := range stream.Values {
				if len(value) < 2 {
					continue
				}

				result.RecentErrors = append(
					result.RecentErrors,
					Error{
						Timestamp: value[0],
						Level:     "ERROR",
						Message:   value[1],
					},
				)
			}
		}
	}

	// --------------------------------------------------
	// 7. Impact on monitored endpoints
	// --------------------------------------------------

	for _, endpoint := range monitoredEndpoints {
		endpointImpact, err := analyzeEndpoint(
			ctx,
			s.prometheus,
			endpoint,
			window,
		)
		if err != nil {
			return nil, err
		}

		result.Impact.Endpoints = append(
			result.Impact.Endpoints,
			endpointImpact,
		)

		if endpointImpact.Status == "degraded" {
			result.Impact.Detected = true
			result.Impact.AffectedPaths = append(
				result.Impact.AffectedPaths,
				endpoint,
			)
		}
	}

	if result.Impact.Detected {
		result.Impact.Description = "One or more monitored endpoints are degraded."
	} else {
		result.Impact.Description = "No endpoint-level degradation detected."
	}

	// --------------------------------------------------
	// 8. Recent deployments
	// --------------------------------------------------

	deploymentList, err := s.deployments.GetDeployments(ctx)
	if err == nil {
		for _, deployment := range deploymentList {
			if deployment.Service != service {
				continue
			}

			if deployment.Timestamp.Before(start) {
				continue
			}

			result.RecentDeployments = append(
				result.RecentDeployments,
				Deployment{
					ID:          deployment.ID,
					Version:     deployment.Version,
					Environment: deployment.Environment,
					Timestamp:   deployment.Timestamp,
					Status:      deployment.Status,
				},
			)
		}
	}

	// --------------------------------------------------
	// 9. Dependency: Redis
	// --------------------------------------------------

	redisErrorQuery := `{service_name="orderflow"} | json | dependency="redis"`
	redisResult, err := s.loki.QueryRange(
		ctx,
		redisErrorQuery,
		start,
		end,
		100,
	)

	if err != nil {
		result.Dependencies = append(
			result.Dependencies,
			DependencyStatus{
				Name:          "redis",
				Status:        "unknown",
				CurrentStatus: "unknown",
				WindowStatus:  "unknown",
			},
		)
	} else {
		var redisHistoricalCount float64
		var redisRecentCount float64

		for _, stream := range redisResult.Data.Result {
			for _, val := range stream.Values {
				if len(val) < 2 {
					continue
				}
				redisHistoricalCount++
				if tsNano, parseErr := strconv.ParseInt(val[0], 10, 64); parseErr == nil {
					t := time.Unix(0, tsNano)
					if t.After(recentCutoff) {
						redisRecentCount++
					}
				}
			}
		}

		redisCurrentStatus := "healthy"
		redisWindowStatus := "healthy"
		if redisHistoricalCount > 0 {
			redisWindowStatus = "degraded"
		}
		if redisRecentCount > 0 {
			redisCurrentStatus = "degraded"
		}

		result.Dependencies = append(
			result.Dependencies,
			DependencyStatus{
				Name:                 "redis",
				Status:               redisCurrentStatus,
				CurrentStatus:        redisCurrentStatus,
				WindowStatus:         redisWindowStatus,
				HistoricalErrorCount: redisHistoricalCount,
				ErrorCount:           redisHistoricalCount,
				RecentErrorCount:     redisRecentCount,
			},
		)
	}

	// --------------------------------------------------
	// 10. Dependency: PostgreSQL (database)
	// --------------------------------------------------

	databaseErrorQuery := `{service_name="orderflow"} | json | dependency="database"`
	databaseResult, err := s.loki.QueryRange(
		ctx,
		databaseErrorQuery,
		start,
		end,
		100,
	)

	if err != nil {
		result.Dependencies = append(
			result.Dependencies,
			DependencyStatus{
				Name:          "postgresql",
				Status:        "unknown",
				CurrentStatus: "unknown",
				WindowStatus:  "unknown",
			},
		)
	} else {
		var dbHistoricalCount float64
		var dbRecentCount float64

		for _, stream := range databaseResult.Data.Result {
			for _, val := range stream.Values {
				if len(val) < 2 {
					continue
				}
				dbHistoricalCount++
				if tsNano, parseErr := strconv.ParseInt(val[0], 10, 64); parseErr == nil {
					t := time.Unix(0, tsNano)
					if t.After(recentCutoff) {
						dbRecentCount++
					}
				}
			}
		}

		dbCurrentStatus := "healthy"
		dbWindowStatus := "healthy"
		if dbHistoricalCount > 0 {
			dbWindowStatus = "degraded"
		}
		if dbRecentCount > 0 {
			dbCurrentStatus = "degraded"
		}

		result.Dependencies = append(
			result.Dependencies,
			DependencyStatus{
				Name:                 "postgresql",
				Status:               dbCurrentStatus,
				CurrentStatus:        dbCurrentStatus,
				WindowStatus:         dbWindowStatus,
				HistoricalErrorCount: dbHistoricalCount,
				ErrorCount:           dbHistoricalCount,
				RecentErrorCount:     dbRecentCount,
			},
		)
	}

	// --------------------------------------------------
	// 11. Dependency: Payment Provider
	// --------------------------------------------------

	paymentQuery := fmt.Sprintf(
		`increase(orderflow_payment_failure_total[%s])`,
		window,
	)
	paymentRecentQuery := fmt.Sprintf(
		`increase(orderflow_payment_failure_total[%s])`,
		formatDuration(recentWindowDuration),
	)

	paymentResult, pErr1 := s.prometheus.Query(ctx, paymentQuery)
	paymentRecentResult, pErr2 := s.prometheus.Query(ctx, paymentRecentQuery)

	if pErr1 != nil && pErr2 != nil {
		result.Dependencies = append(
			result.Dependencies,
			DependencyStatus{
				Name:          "payment-provider",
				Status:        "unknown",
				CurrentStatus: "unknown",
				WindowStatus:  "unknown",
			},
		)
	} else {
		windowFailures, _ := prom.ExtractScalar(paymentResult)
		recentFailures, _ := prom.ExtractScalar(paymentRecentResult)

		var histCount float64
		var recCount float64
		if windowFailures != nil {
			histCount = *windowFailures
		}
		if recentFailures != nil {
			recCount = *recentFailures
		}

		// Also check Loki for payment errors
		paymentLokiQuery := `{service_name="orderflow"} | json | dependency="payment"`
		if payLokiRes, payLokiErr := s.loki.QueryRange(ctx, paymentLokiQuery, start, end, 100); payLokiErr == nil && payLokiRes != nil {
			var lokiHist float64
			var lokiRec float64
			for _, stream := range payLokiRes.Data.Result {
				for _, val := range stream.Values {
					if len(val) < 2 {
						continue
					}
					lokiHist++
					if tsNano, parseErr := strconv.ParseInt(val[0], 10, 64); parseErr == nil {
						if time.Unix(0, tsNano).After(recentCutoff) {
							lokiRec++
						}
					}
				}
			}
			if lokiHist > histCount {
				histCount = lokiHist
			}
			if lokiRec > recCount {
				recCount = lokiRec
			}
		}

		paymentCurrentStatus := "healthy"
		paymentWindowStatus := "healthy"
		if histCount > 0 {
			paymentWindowStatus = "degraded"
		}
		if recCount > 0 {
			paymentCurrentStatus = "degraded"
		}

		result.Dependencies = append(
			result.Dependencies,
			DependencyStatus{
				Name:                 "payment-provider",
				Status:               paymentCurrentStatus,
				CurrentStatus:        paymentCurrentStatus,
				WindowStatus:         paymentWindowStatus,
				HistoricalErrorCount: histCount,
				ErrorCount:           histCount,
				RecentErrorCount:     recCount,
			},
		)
	}

	// --------------------------------------------------
	// 12. Diagnosis & Timeline
	// --------------------------------------------------

	result.Diagnosis = Diagnose(
		result,
		duration,
	)

	result.Timeline = BuildTimeline(
		result,
		duration,
	)

	return result, nil
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "5m"
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", d/time.Hour)
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", d/time.Minute)
	}
	if d%time.Second == 0 {
		return fmt.Sprintf("%ds", d/time.Second)
	}
	return d.String()
}
