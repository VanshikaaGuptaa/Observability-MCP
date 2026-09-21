package investigation

import (
	"context"
	"fmt"
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
		RequestRate float64 `json:"request_rate"`
		ErrorRate   float64 `json:"error_rate"`
		P95Latency  float64 `json:"p95_latency_seconds"`
		P99Latency  float64 `json:"p99_latency_seconds"`
	} `json:"metrics"`

	RecentErrors      []Error      `json:"recent_errors"`
	RecentDeployments []Deployment `json:"recent_deployments"`

	Diagnosis *Diagnosis `json:"diagnosis,omitempty"`
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
		Service: service,
	}

	window := formatDuration(duration)

	// --------------------------------------------------
	// 1. Service health
	// --------------------------------------------------

	healthResult, err := s.prometheus.Query(
		ctx,
		`up{job="orderflow"}`,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"health query failed: %w",
			err,
		)
	}

	if len(healthResult.Data.Result) > 0 &&
		len(healthResult.Data.Result[0].Value) >= 2 {

		value := fmt.Sprint(
			healthResult.Data.Result[0].Value[1],
		)

		if value == "1" {
			result.Health.Status = "healthy"
		} else {
			result.Health.Status = "down"
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

	result.Metrics.RequestRate = extractScalar(
		requestRateResult,
	)

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

	result.Metrics.ErrorRate = extractScalar(
		errorRateResult,
	)

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

	result.Metrics.P95Latency = extractScalar(
		p95Result,
	)

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

	result.Metrics.P99Latency = extractScalar(
		p99Result,
	)

	// --------------------------------------------------
	// 6. Recent errors from Loki
	// --------------------------------------------------

	end := time.Now()
	start := end.Add(-duration)

	logQuery := `{service_name="orderflow"} | json | level="ERROR"`

	logResult, err := s.loki.QueryRange(
		ctx,
		logQuery,
		start,
		end,
		50,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"recent errors query failed: %w",
			err,
		)
	}

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

	// --------------------------------------------------
	// 7. Recent deployments
	// --------------------------------------------------

	deploymentList, err := s.deployments.GetDeployments(
		ctx,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"deployment query failed: %w",
			err,
		)
	}

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
	result.Diagnosis = Diagnose(
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

func extractScalar(resp *prom.QueryResponse) float64 {
	if resp == nil || len(resp.Data.Result) == 0 {
		return 0
	}
	val := resp.Data.Result[0].Value
	if len(val) < 2 {
		return 0
	}
	valStr, ok := val[1].(string)
	if !ok {
		return 0
	}
	var scalar float64
	if _, err := fmt.Sscanf(valStr, "%f", &scalar); err != nil {
		return 0
	}
	return scalar
}
