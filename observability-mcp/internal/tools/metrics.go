package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-mcp/internal/errors"
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
	Service           string   `json:"service"`
	Duration          string   `json:"duration"`
	RequestsPerSecond *float64 `json:"requests_per_second"`
}

type ErrorRateInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 5m or 15m"`
}

type ErrorRateOutput struct {
	Service   string   `json:"service"`
	Duration  string   `json:"duration"`
	ErrorRate *float64 `json:"error_rate"`
}

type LatencyInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Endpoint string `json:"endpoint" jsonschema:"the HTTP endpoint such as /api/orders"`
	Duration string `json:"duration" jsonschema:"the time window such as 5m or 15m"`
}

type LatencyOutput struct {
	Service    string   `json:"service"`
	Endpoint   string   `json:"endpoint"`
	Duration   string   `json:"duration"`
	P50Seconds *float64 `json:"p50_seconds"`
	P95Seconds *float64 `json:"p95_seconds"`
	P99Seconds *float64 `json:"p99_seconds"`
}

func (s *Server) GetServiceHealth(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input HealthInput,
) (*mcp.CallToolResult, HealthOutput, error) {

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, HealthOutput{}, err
	}

	query := fmt.Sprintf(
		`up{job="%s"}`,
		input.Service,
	)

	result, err := s.prometheus.Query(ctx, query)
	if err != nil {
		return nil, HealthOutput{}, errors.NewBackendUnavailableError("prometheus", err)
	}

	if len(result.Data.Result) == 0 {
		return nil, HealthOutput{
			Service: input.Service,
			Status:  "unknown",
		}, nil
	}

	val, err := prom.ExtractScalar(result)
	if err != nil || val == nil {
		return nil, HealthOutput{
			Service: input.Service,
			Status:  "unknown",
		}, nil
	}

	if *val == 1.0 {
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

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, RequestRateOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}
	if _, err := validation.ValidateDuration(input.Duration); err != nil {
		return nil, RequestRateOutput{}, err
	}

	var query string
	switch input.Service {
	case "orderflow":
		query = fmt.Sprintf(
			`sum(rate(orderflow_http_requests_total[%s]))`,
			input.Duration,
		)
	default:
		return nil, RequestRateOutput{}, errors.NewValidationError(
			fmt.Sprintf("unsupported service: %s", input.Service),
		)
	}

	result, err := s.prometheus.Query(ctx, query)
	if err != nil {
		return nil, RequestRateOutput{}, errors.NewBackendUnavailableError("prometheus", err)
	}

	rate, err := prom.ExtractScalar(result)
	if err != nil {
		return nil, RequestRateOutput{}, errors.NewQueryError(query, err)
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

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, ErrorRateOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}
	if _, err := validation.ValidateDuration(input.Duration); err != nil {
		return nil, ErrorRateOutput{}, err
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
		return nil, ErrorRateOutput{}, errors.NewValidationError(
			fmt.Sprintf("unsupported service: %s", input.Service),
		)
	}

	result, err := s.prometheus.Query(ctx, query)
	if err != nil {
		return nil, ErrorRateOutput{}, errors.NewBackendUnavailableError("prometheus", err)
	}

	errorRate, err := prom.ExtractScalar(result)
	if err != nil {
		return nil, ErrorRateOutput{}, errors.NewQueryError(query, err)
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

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, LatencyOutput{}, err
	}

	if err := validation.ValidateEndpoint(input.Endpoint); err != nil {
		return nil, LatencyOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "5m"
	}
	if _, err := validation.ValidateDuration(input.Duration); err != nil {
		return nil, LatencyOutput{}, err
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

	values := make(map[string]*float64)

	for percentile, query := range queries {
		result, err := s.prometheus.Query(ctx, query)
		if err != nil {
			return nil, LatencyOutput{}, errors.NewBackendUnavailableError("prometheus", err)
		}

		latency, err := prom.ExtractScalar(result)
		if err != nil {
			return nil, LatencyOutput{}, errors.NewQueryError(query, err)
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
