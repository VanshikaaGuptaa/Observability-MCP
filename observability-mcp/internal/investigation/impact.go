package investigation

import (
	"context"
	"fmt"

	prom "observability-mcp/internal/prometheus"
)

type EndpointImpact struct {
	Endpoint    string  `json:"endpoint"`
	RequestRate float64 `json:"request_rate"`
	ErrorRate   float64 `json:"error_rate"`
	P95Latency  float64 `json:"p95_latency_seconds"`
	P99Latency  float64 `json:"p99_latency_seconds"`
	Status      string  `json:"status"`
}

type ImpactSummary struct {
	Detected      bool             `json:"detected"`
	AffectedPaths []string         `json:"affected_paths"`
	Endpoints     []EndpointImpact `json:"endpoints"`
	Description   string           `json:"description"`
}

var monitoredEndpoints = []string{
	"/api/orders",
	"/api/products",
	"/api/products/1",
	"/api/payment",
}

func analyzeEndpoint(
	ctx context.Context,
	client *prom.Client,
	endpoint string,
	window string,
) (EndpointImpact, error) {

	result := EndpointImpact{
		Endpoint: endpoint,
		Status:   "healthy",
	}

	// Request rate
	requestRateQuery := fmt.Sprintf(
		`sum(rate(orderflow_http_requests_total{route="%s"}[%s]))`,
		endpoint,
		window,
	)

	requestRateResult, err := client.Query(
		ctx,
		requestRateQuery,
	)

	if err != nil {
		return result, fmt.Errorf(
			"request rate query for %s failed: %w",
			endpoint,
			err,
		)
	}

	result.RequestRate = extractScalar(
		requestRateResult,
	)

	// Error rate
	errorRateQuery := fmt.Sprintf(
		`sum(rate(orderflow_http_requests_total{route="%s",status=~"5.."}[%s])) / sum(rate(orderflow_http_requests_total{route="%s"}[%s]))`,
		endpoint,
		window,
		endpoint,
		window,
	)

	errorRateResult, err := client.Query(
		ctx,
		errorRateQuery,
	)

	if err != nil {
		return result, fmt.Errorf(
			"error rate query for %s failed: %w",
			endpoint,
			err,
		)
	}

	result.ErrorRate = extractScalar(
		errorRateResult,
	)

	// P95
	p95Query := fmt.Sprintf(
		`histogram_quantile(0.95, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
		endpoint,
		window,
	)

	p95Result, err := client.Query(
		ctx,
		p95Query,
	)

	if err != nil {
		return result, fmt.Errorf(
			"P95 query for %s failed: %w",
			endpoint,
			err,
		)
	}

	result.P95Latency = extractScalar(
		p95Result,
	)

	// P99
	p99Query := fmt.Sprintf(
		`histogram_quantile(0.99, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
		endpoint,
		window,
	)

	p99Result, err := client.Query(
		ctx,
		p99Query,
	)

	if err != nil {
		return result, fmt.Errorf(
			"P99 query for %s failed: %w",
			endpoint,
			err,
		)
	}

	result.P99Latency = extractScalar(
		p99Result,
	)

	// Determine endpoint status.
	if result.ErrorRate >= 0.05 {
		result.Status = "degraded"
	} else if result.P99Latency >= 2 {
		result.Status = "degraded"
	}

	return result, nil
}
