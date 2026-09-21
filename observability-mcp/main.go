package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	deployments "observability-mcp/internal/deployments"
	investigation "observability-mcp/internal/investigation"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
	"observability-mcp/internal/validation"
)

type HealthInput struct {
	Service string `json:"service" jsonschema:"the name of the service to check"`
}

type HealthOutput struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}
type RequestRateInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 5m or 15m"`
}

type RequestRateOutput struct {
	Service           string  `json:"service"`
	Duration          string  `json:"duration"`
	RequestsPerSecond float64 `json:"requests_per_second"`
}
type Server struct {
	prometheus    *prom.Client
	loki          *loki.Client
	deployments   *deployments.Client
	investigation *investigation.Service
}
type ErrorRateInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 5m or 15m"`
}

type ErrorRateOutput struct {
	Service   string  `json:"service"`
	Duration  string  `json:"duration"`
	ErrorRate float64 `json:"error_rate"`
}
type LatencyInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Endpoint string `json:"endpoint" jsonschema:"the HTTP endpoint such as /api/orders"`
	Duration string `json:"duration" jsonschema:"the time window such as 5m or 15m"`
}

type LatencyOutput struct {
	Service    string  `json:"service"`
	Endpoint   string  `json:"endpoint"`
	Duration   string  `json:"duration"`
	P50Seconds float64 `json:"p50_seconds"`
	P95Seconds float64 `json:"p95_seconds"`
	P99Seconds float64 `json:"p99_seconds"`
}
type SearchLogsInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Query    string `json:"query" jsonschema:"text to search for in the logs"`
	Duration string `json:"duration" jsonschema:"the time window such as 10m or 1h"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type SearchLogsOutput struct {
	Service  string     `json:"service"`
	Query    string     `json:"query"`
	Duration string     `json:"duration"`
	Logs     []LogEntry `json:"logs"`
}
type RecentErrorsInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 10m or 1h"`
}

type RecentErrorsOutput struct {
	Service  string     `json:"service"`
	Duration string     `json:"duration"`
	Errors   []LogEntry `json:"errors"`
}
type RecentDeploymentsInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 1h or 24h"`
}

type DeploymentOutput struct {
	ID          string    `json:"id"`
	Service     string    `json:"service"`
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	DeployedBy  string    `json:"deployed_by"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"`
}

type RecentDeploymentsOutput struct {
	Service     string             `json:"service"`
	Duration    string             `json:"duration"`
	Deployments []DeploymentOutput `json:"deployments"`
}

// add new type here
func NewServer(
	prometheusClient *prom.Client,
	lokiClient *loki.Client,
	deploymentClient *deployments.Client,
) *Server {

	investigationService := investigation.NewService(
		prometheusClient,
		lokiClient,
		deploymentClient,
	)

	return &Server{
		prometheus:    prometheusClient,
		loki:          lokiClient,
		deployments:   deploymentClient,
		investigation: investigationService,
	}
}

func (s *Server) GetServiceHealth(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input HealthInput,
) (*mcp.CallToolResult, HealthOutput, error) {

	if input.Service == "" {
		return nil, HealthOutput{}, fmt.Errorf("service is required")
	}

	query := fmt.Sprintf(
		`up{job="%s"}`,
		input.Service,
	)

	result, err := s.prometheus.Query(ctx, query)
	if err != nil {
		return nil, HealthOutput{}, err
	}

	if len(result.Data.Result) == 0 {
		return nil, HealthOutput{
			Service: input.Service,
			Status:  "unknown",
		}, nil
	}

	value := result.Data.Result[0].Value

	if len(value) < 2 {
		return nil, HealthOutput{
			Service: input.Service,
			Status:  "unknown",
		}, nil
	}

	statusValue := fmt.Sprint(value[1])

	if statusValue == "1" {
		return nil, HealthOutput{
			Service: input.Service,
			Status:  "healthy",
		}, nil
	}

	return nil, HealthOutput{
		Service: input.Service,
		Status:  "down",
	}, nil
}
func (s *Server) GetRequestRate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RequestRateInput,
) (*mcp.CallToolResult, RequestRateOutput, error) {

	if input.Service == "" {
		return nil, RequestRateOutput{}, fmt.Errorf("service is required")
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}

	var query string

	switch input.Service {
	case "orderflow":
		query = fmt.Sprintf(
			`sum(rate(orderflow_http_requests_total[%s]))`,
			input.Duration,
		)
	default:
		return nil, RequestRateOutput{}, fmt.Errorf(
			"unsupported service: %s",
			input.Service,
		)
	}

	result, err := s.prometheus.Query(ctx, query)
	if err != nil {
		return nil, RequestRateOutput{}, err
	}

	if len(result.Data.Result) == 0 {
		return nil, RequestRateOutput{
			Service:           input.Service,
			Duration:          input.Duration,
			RequestsPerSecond: 0,
		}, nil
	}

	value := result.Data.Result[0].Value

	if len(value) < 2 {
		return nil, RequestRateOutput{}, fmt.Errorf(
			"unexpected Prometheus response",
		)
	}

	requestRate, ok := value[1].(string)
	if !ok {
		return nil, RequestRateOutput{}, fmt.Errorf(
			"unexpected Prometheus value type",
		)
	}

	var rate float64

	if _, err := fmt.Sscanf(requestRate, "%f", &rate); err != nil {
		return nil, RequestRateOutput{}, fmt.Errorf(
			"failed to parse request rate: %w",
			err,
		)
	}

	return nil, RequestRateOutput{
		Service:           input.Service,
		Duration:          input.Duration,
		RequestsPerSecond: rate,
	}, nil
}
func (s *Server) GetErrorRate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ErrorRateInput,
) (*mcp.CallToolResult, ErrorRateOutput, error) {

	if input.Service == "" {
		return nil, ErrorRateOutput{}, fmt.Errorf("service is required")
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}

	var query string

	switch input.Service {
	case "orderflow":
		query = fmt.Sprintf(
			`sum(rate(orderflow_http_requests_total{status=~"5.."}[%s])) / sum(rate(orderflow_http_requests_total[%s]))`,
			input.Duration,
			input.Duration,
		)
	default:
		return nil, ErrorRateOutput{}, fmt.Errorf(
			"unsupported service: %s",
			input.Service,
		)
	}

	result, err := s.prometheus.Query(ctx, query)
	if err != nil {
		return nil, ErrorRateOutput{}, err
	}

	if len(result.Data.Result) == 0 {
		return nil, ErrorRateOutput{
			Service:   input.Service,
			Duration:  input.Duration,
			ErrorRate: 0,
		}, nil
	}

	value := result.Data.Result[0].Value

	if len(value) < 2 {
		return nil, ErrorRateOutput{}, fmt.Errorf(
			"unexpected Prometheus response",
		)
	}

	errorRateString, ok := value[1].(string)
	if !ok {
		return nil, ErrorRateOutput{}, fmt.Errorf(
			"unexpected Prometheus value type",
		)
	}

	var errorRate float64

	if _, err := fmt.Sscanf(errorRateString, "%f", &errorRate); err != nil {
		return nil, ErrorRateOutput{}, fmt.Errorf(
			"failed to parse error rate: %w",
			err,
		)
	}

	return nil, ErrorRateOutput{
		Service:   input.Service,
		Duration:  input.Duration,
		ErrorRate: errorRate,
	}, nil
}
func (s *Server) GetLatency(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input LatencyInput,
) (*mcp.CallToolResult, LatencyOutput, error) {

	if input.Service == "" {
		return nil, LatencyOutput{}, fmt.Errorf("service is required")
	}

	if err := validation.ValidateEndpoint(
		input.Endpoint,
	); err != nil {
		return nil, LatencyOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}

	if input.Service != "orderflow" {
		return nil, LatencyOutput{}, fmt.Errorf(
			"unsupported service: %s",
			input.Service,
		)
	}

	queries := map[string]string{
		"p50": fmt.Sprintf(
			`histogram_quantile(0.50, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
			input.Endpoint,
			input.Duration,
		),
		"p95": fmt.Sprintf(
			`histogram_quantile(0.95, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
			input.Endpoint,
			input.Duration,
		),
		"p99": fmt.Sprintf(
			`histogram_quantile(0.99, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
			input.Endpoint,
			input.Duration,
		),
	}

	values := make(map[string]float64)

	for percentile, query := range queries {
		result, err := s.prometheus.Query(ctx, query)
		if err != nil {
			return nil, LatencyOutput{}, fmt.Errorf(
				"%s latency query failed: %w",
				percentile,
				err,
			)
		}

		if len(result.Data.Result) == 0 {
			continue
		}

		value := result.Data.Result[0].Value

		if len(value) < 2 {
			continue
		}

		valueString, ok := value[1].(string)
		if !ok {
			return nil, LatencyOutput{}, fmt.Errorf(
				"unexpected Prometheus value type for %s",
				percentile,
			)
		}

		var latency float64

		if _, err := fmt.Sscanf(valueString, "%f", &latency); err != nil {
			return nil, LatencyOutput{}, fmt.Errorf(
				"failed to parse %s latency: %w",
				percentile,
				err,
			)
		}

		values[percentile] = latency
	}

	return nil, LatencyOutput{
		Service:    input.Service,
		Endpoint:   input.Endpoint,
		Duration:   input.Duration,
		P50Seconds: values["p50"],
		P95Seconds: values["p95"],
		P99Seconds: values["p99"],
	}, nil
}
func (s *Server) SearchLogs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchLogsInput,
) (*mcp.CallToolResult, SearchLogsOutput, error) {

	if input.Service == "" {
		return nil, SearchLogsOutput{}, fmt.Errorf(
			"service is required",
		)
	}

	if input.Query == "" {
		return nil, SearchLogsOutput{}, fmt.Errorf(
			"query is required",
		)
	}

	if input.Duration == "" {
		input.Duration = "10m"
	}

	if input.Service != "orderflow" {
		return nil, SearchLogsOutput{}, fmt.Errorf(
			"unsupported service: %s",
			input.Service,
		)
	}

	duration, err := time.ParseDuration(input.Duration)
	if err != nil {
		return nil, SearchLogsOutput{}, fmt.Errorf(
			"invalid duration: %w",
			err,
		)
	}

	end := time.Now()
	start := end.Add(-duration)

	logQL := fmt.Sprintf(
		`{service_name="%s"} |= %q`,
		input.Service,
		input.Query,
	)

	result, err := s.loki.QueryRange(
		ctx,
		logQL,
		start,
		end,
		50,
	)

	if err != nil {
		return nil, SearchLogsOutput{}, err
	}

	output := SearchLogsOutput{
		Service:  input.Service,
		Query:    input.Query,
		Duration: input.Duration,
		Logs:     []LogEntry{},
	}

	for _, stream := range result.Data.Result {

		for _, value := range stream.Values {

			if len(value) < 2 {
				continue
			}

			timestamp := value[0]
			rawLog := value[1]

			var parsed struct {
				Time  string `json:"time"`
				Level string `json:"level"`
				Msg   string `json:"msg"`
			}

			if err := json.Unmarshal(
				[]byte(rawLog),
				&parsed,
			); err != nil {
				continue
			}

			output.Logs = append(
				output.Logs,
				LogEntry{
					Timestamp: timestamp,
					Level:     parsed.Level,
					Message:   parsed.Msg,
				},
			)
		}
	}

	return nil, output, nil
}
func (s *Server) GetRecentErrors(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RecentErrorsInput,
) (*mcp.CallToolResult, RecentErrorsOutput, error) {

	if input.Service == "" {
		return nil, RecentErrorsOutput{}, fmt.Errorf(
			"service is required",
		)
	}

	if input.Duration == "" {
		input.Duration = "10m"
	}

	if input.Service != "orderflow" {
		return nil, RecentErrorsOutput{}, fmt.Errorf(
			"unsupported service: %s",
			input.Service,
		)
	}

	duration, err := time.ParseDuration(input.Duration)
	if err != nil {
		return nil, RecentErrorsOutput{}, fmt.Errorf(
			"invalid duration: %w",
			err,
		)
	}

	end := time.Now()
	start := end.Add(-duration)

	logQL := fmt.Sprintf(
		`{service_name="%s"} | json | level="ERROR"`,
		input.Service,
	)

	result, err := s.loki.QueryRange(
		ctx,
		logQL,
		start,
		end,
		50,
	)

	if err != nil {
		return nil, RecentErrorsOutput{}, err
	}

	output := RecentErrorsOutput{
		Service:  input.Service,
		Duration: input.Duration,
		Errors:   []LogEntry{},
	}

	for _, stream := range result.Data.Result {
		for _, value := range stream.Values {

			if len(value) < 2 {
				continue
			}

			timestamp := value[0]
			rawLog := value[1]

			var parsed struct {
				Time  string `json:"time"`
				Level string `json:"level"`
				Msg   string `json:"msg"`
			}

			if err := json.Unmarshal(
				[]byte(rawLog),
				&parsed,
			); err != nil {
				continue
			}

			output.Errors = append(
				output.Errors,
				LogEntry{
					Timestamp: timestamp,
					Level:     parsed.Level,
					Message:   parsed.Msg,
				},
			)
		}
	}

	return nil, output, nil
}
func (s *Server) GetRecentDeployments(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RecentDeploymentsInput,
) (*mcp.CallToolResult, RecentDeploymentsOutput, error) {

	if input.Service == "" {
		return nil, RecentDeploymentsOutput{}, fmt.Errorf(
			"service is required",
		)
	}

	if input.Duration == "" {
		input.Duration = "24h"
	}

	duration, err := time.ParseDuration(input.Duration)
	if err != nil {
		return nil, RecentDeploymentsOutput{}, fmt.Errorf(
			"invalid duration: %w",
			err,
		)
	}

	deployments, err := s.deployments.GetDeployments(ctx)
	if err != nil {
		return nil, RecentDeploymentsOutput{}, err
	}

	cutoff := time.Now().Add(-duration)

	output := RecentDeploymentsOutput{
		Service:     input.Service,
		Duration:    input.Duration,
		Deployments: []DeploymentOutput{},
	}

	for _, deployment := range deployments {

		if deployment.Service != input.Service {
			continue
		}

		if deployment.Timestamp.Before(cutoff) {
			continue
		}

		output.Deployments = append(
			output.Deployments,
			DeploymentOutput{
				ID:          deployment.ID,
				Service:     deployment.Service,
				Version:     deployment.Version,
				Environment: deployment.Environment,
				DeployedBy:  deployment.DeployedBy,
				Timestamp:   deployment.Timestamp,
				Status:      deployment.Status,
			},
		)
	}

	return nil, output, nil
}

// add new handler here
func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	prometheusClient := prom.NewClient(
		config.PrometheusURL,
	)

	lokiClient := loki.NewClient(
		config.LokiURL,
	)

	deploymentClient := deployments.NewClient(
		config.OrderFlowURL,
	)

	app := NewServer(
		prometheusClient,
		lokiClient,
		deploymentClient,
	)

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "observability-mcp",
			Version: "0.1.0",
		},
		nil,
	)

	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_service_health",
			Description: "Get the current health status of an observability service",
		},
		app.GetServiceHealth,
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_request_rate",
			Description: "Get the average HTTP request rate of an observability service over a specified time window",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input RequestRateInput,
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			_ = duration
			return app.GetRequestRate(ctx, req, input)
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_error_rate",
			Description: "Get the HTTP 5xx error rate of an observability service over a specified time window",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input ErrorRateInput,
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			_ = duration
			return app.GetErrorRate(ctx, req, input)
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_latency",
			Description: "Get P50, P95, and P99 HTTP latency for an observability service endpoint",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input LatencyInput,
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			if err := validation.ValidateEndpoint(
				input.Endpoint,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			_ = duration
			return app.GetLatency(ctx, req, input)
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "search_logs",
			Description: "Search logs for an observability service over a specified time window",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input SearchLogsInput,
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			_ = duration
			return app.SearchLogs(ctx, req, input)
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_recent_errors",
			Description: "Get recent error logs for an observability service",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input RecentErrorsInput,
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			_ = duration
			return app.GetRecentErrors(ctx, req, input)
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_recent_deployments",
			Description: "Get recent deployments for an observability service",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input RecentDeploymentsInput,
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			_ = duration
			return app.GetRecentDeployments(ctx, req, input)
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "investigate_service",
			Description: "Investigate a service by correlating health, request rate, error rate, latency, recent errors, and recent deployments.",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input struct {
				Service  string `json:"service"`
				Duration string `json:"duration"`
			},
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			result, err := app.investigation.Investigate(
				ctx,
				input.Service,
				duration,
			)

			if err != nil {
				return nil, nil, err
			}

			return nil, result, nil
		},
	)
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "investigate_incident",
			Description: "Analyze observability evidence and produce a structured incident containing symptoms, evidence, and possible causes.",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input struct {
				Service  string `json:"service"`
				Duration string `json:"duration"`
			},
		) (*mcp.CallToolResult, any, error) {

			if err := validation.ValidateService(
				input.Service,
			); err != nil {
				return nil, nil, err
			}

			duration, err := validation.ValidateDuration(
				input.Duration,
			)

			if err != nil {
				return nil, nil, err
			}

			investigationResult, err :=
				app.investigation.Investigate(
					ctx,
					input.Service,
					duration,
				)

			if err != nil {
				return nil, nil, err
			}

			incident := investigation.BuildIncident(
				investigationResult,
			)

			if incident == nil {
				return nil, nil, fmt.Errorf(
					"unable to build incident",
				)
			}

			return nil, incident, nil
		},
	)
	// Register the new tool here
	if err := mcpServer.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		log.Fatal(err)
	}
}
