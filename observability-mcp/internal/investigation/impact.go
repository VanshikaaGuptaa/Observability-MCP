package investigation

import (
	"context"
	"fmt"

	prom "observability-mcp/internal/prometheus"
)

type EndpointImpact struct {
	Endpoint    string   `json:"endpoint"`
	RequestRate *float64 `json:"request_rate"`
	ErrorRate   *float64 `json:"error_rate"`
	P95Latency  *float64 `json:"p95_latency_seconds"`
	P99Latency  *float64 `json:"p99_latency_seconds"`
	Status      string   `json:"status"`
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
		Status:   "unknown",
	}

	// 1. Request rate
	requestRateQuery := fmt.Sprintf(
		`sum(rate(orderflow_http_requests_total{route="%s"}[%s]))`,
		endpoint,
		window,
	)

	requestRateResult, err := client.Query(ctx, requestRateQuery)
	if err != nil {
		return result, fmt.Errorf("request rate query for %s failed: %w", endpoint, err)
	}

	reqRate, err := prom.ExtractScalar(requestRateResult)
	if err != nil {
		return result, fmt.Errorf("parse request rate for %s: %w", endpoint, err)
	}
	result.RequestRate = reqRate

	// 2. Error rate
	errorRateQuery := fmt.Sprintf(
		`sum(rate(orderflow_http_requests_total{route="%s",status=~"5.."}[%s])) / sum(rate(orderflow_http_requests_total{route="%s"}[%s]))`,
		endpoint,
		window,
		endpoint,
		window,
	)

	errorRateResult, err := client.Query(ctx, errorRateQuery)
	if err != nil {
		return result, fmt.Errorf("error rate query for %s failed: %w", endpoint, err)
	}

	errRate, err := prom.ExtractScalar(errorRateResult)
	if err != nil {
		return result, fmt.Errorf("parse error rate for %s: %w", endpoint, err)
	}
	result.ErrorRate = errRate

	// 3. P95 latency
	p95Query := fmt.Sprintf(
		`histogram_quantile(0.95, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
		endpoint,
		window,
	)

	p95Result, err := client.Query(ctx, p95Query)
	if err != nil {
		return result, fmt.Errorf("P95 query for %s failed: %w", endpoint, err)
	}

	p95, err := prom.ExtractScalar(p95Result)
	if err != nil {
		return result, fmt.Errorf("parse P95 for %s: %w", endpoint, err)
	}
	result.P95Latency = p95

	// 4. P99 latency
	p99Query := fmt.Sprintf(
		`histogram_quantile(0.99, sum by (le) (rate(orderflow_http_request_duration_seconds_bucket{route="%s"}[%s])))`,
		endpoint,
		window,
	)

	p99Result, err := client.Query(ctx, p99Query)
	if err != nil {
		return result, fmt.Errorf("P99 query for %s failed: %w", endpoint, err)
	}

	p99, err := prom.ExtractScalar(p99Result)
	if err != nil {
		return result, fmt.Errorf("parse P99 for %s: %w", endpoint, err)
	}
	result.P99Latency = p99

	// 5. Determine endpoint status.
	// Never mark healthy if no requests were observed.
	// Degraded if error rate >= 5% or P99 latency >= 2s.
	if result.ErrorRate != nil && *result.ErrorRate >= 0.05 {
		result.Status = "degraded"
	} else if result.P99Latency != nil && *result.P99Latency >= 2.0 {
		result.Status = "degraded"
	} else if result.RequestRate != nil && *result.RequestRate > 0 {
		result.Status = "healthy"
	} else {
		result.Status = "unknown"
	}

	return result, nil
}
