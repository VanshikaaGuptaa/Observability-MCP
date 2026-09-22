package resources

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Embedded runbook contents as reliable fallback
//
//go:embed content/orderflow.md
var embeddedOrderflowRunbook string

//go:embed content/redis.md
var embeddedRedisRunbook string

//go:embed content/postgresql.md
var embeddedPostgresqlRunbook string

//go:embed content/payment.md
var embeddedPaymentRunbook string

// Helper to read runbook from disk or fallback to embedded
func loadRunbook(filename, embeddedFallback string) string {
	// Try possible relative paths
	candidates := []string{
		filepath.Join("runbooks", filename),
		filepath.Join("..", "runbooks", filename),
		filepath.Join("observability-mcp", "runbooks", filename),
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return string(data)
		}
	}
	return embeddedFallback
}

// RegisterResources registers all service topology and runbook MCP Resources.
func RegisterResources(server *mcp.Server) {
	// --------------------------------------------------
	// 1. service://orderflow
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "service://orderflow",
			Name:        "orderflow-overview",
			Title:       "OrderFlow Service Overview",
			Description: "Architecture, dependencies, endpoints, and observability sources for the OrderFlow service.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			overview := map[string]any{
				"service":     "orderflow",
				"description": "OrderFlow e-commerce backend service responsible for catalog, order processing, and payment orchestration.",
				"environment": "production",
				"architecture": map[string]any{
					"language":  "Go",
					"port":      8080,
					"framework": "standard net/http with slog structured logging and Prometheus metrics",
				},
				"dependencies": []string{
					"redis",
					"postgresql",
					"payment-provider",
				},
				"endpoints": []string{
					"/api/health",
					"/api/products",
					"/api/products/1",
					"/api/orders",
					"/api/payment",
					"/api/deployments",
					"/metrics",
				},
				"observability": map[string]any{
					"prometheus": map[string]any{
						"job": "orderflow",
						"metrics": []string{
							"up",
							"orderflow_http_requests_total",
							"orderflow_http_request_duration_seconds_bucket",
							"orderflow_payment_failure_total",
						},
					},
					"loki": map[string]any{
						"service_name": "orderflow",
						"format":       "JSON slog",
					},
				},
				"simulated_scenarios": []string{
					"normal",
					"slow",
					"error_spike",
					"payment_failure",
					"redis_failure",
					"db_failure",
				},
			}
			bytes, err := json.MarshalIndent(overview, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal resource: %w", err)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(bytes),
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 2. service://orderflow/dependencies
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "service://orderflow/dependencies",
			Name:        "orderflow-dependencies",
			Title:       "OrderFlow Dependencies Topology",
			Description: "Detailed topology and failure behaviors for Redis, PostgreSQL, and payment provider.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			deps := map[string]any{
				"service": "orderflow",
				"dependencies": []map[string]any{
					{
						"name":         "redis",
						"type":         "cache",
						"port":         6379,
						"purpose":      "Product catalog caching for /api/products and /api/products/{id}",
						"fallback":     "Queries fall back transparently to PostgreSQL. HTTP errors remain 0%, slight latency increase possible.",
						"failure_mode": "Cache misses increase database query load; does not cause user-facing outage.",
					},
					{
						"name":         "postgresql",
						"type":         "relational_database",
						"port":         5432,
						"database":     "orderflow_db",
						"purpose":      "Primary data store for orders, order items, and products.",
						"fallback":     "None. Failures directly cause HTTP 500 errors on /api/orders and uncached product queries.",
						"failure_mode": "Immediate critical user-facing degradation for order placement.",
					},
					{
						"name":         "payment-provider",
						"type":         "external_api",
						"purpose":      "Payment authorization and settlement on /api/payment",
						"fallback":     "None. Transactions fail and are rejected with HTTP 500.",
						"failure_mode": "Checkout settlement fails; product browsing and order drafting remain functional.",
					},
				},
			}
			bytes, err := json.MarshalIndent(deps, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal resource: %w", err)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(bytes),
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 3. service://orderflow/endpoints
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "service://orderflow/endpoints",
			Name:        "orderflow-endpoints",
			Title:       "OrderFlow Monitored Endpoints",
			Description: "Monitored endpoints, HTTP methods, route patterns, and criticality levels.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			endpoints := map[string]any{
				"service": "orderflow",
				"endpoints": []map[string]any{
					{
						"path":        "/api/health",
						"method":      "GET",
						"description": "Liveness and readiness check verifying database pool and redis connectivity.",
						"criticality": "high",
					},
					{
						"path":        "/api/products",
						"method":      "GET",
						"description": "Catalog listing of available products. Reads from Redis cache, falls back to PostgreSQL.",
						"criticality": "high",
					},
					{
						"path":        "/api/products/1",
						"method":      "GET",
						"description": "Single product lookup by ID. Reads from Redis cache, falls back to PostgreSQL.",
						"criticality": "high",
					},
					{
						"path":        "/api/orders",
						"method":      "POST",
						"description": "Order creation and inventory validation against PostgreSQL.",
						"criticality": "critical",
					},
					{
						"path":        "/api/payment",
						"method":      "POST",
						"description": "Payment authorization and processing via payment provider.",
						"criticality": "critical",
					},
					{
						"path":        "/api/deployments",
						"method":      "GET",
						"description": "Query recent deployment history and versions.",
						"criticality": "medium",
					},
					{
						"path":        "/metrics",
						"method":      "GET",
						"description": "Prometheus scrape endpoint exposing latency histograms and request counters.",
						"criticality": "medium",
					},
				},
			}
			bytes, err := json.MarshalIndent(endpoints, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal resource: %w", err)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(bytes),
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 4. service://orderflow/deployments
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "service://orderflow/deployments",
			Name:        "orderflow-deployments",
			Title:       "OrderFlow Deployments Metadata",
			Description: "Deployment endpoint details and environment metadata for tracking changes.",
			MIMEType:    "application/json",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			deployMeta := map[string]any{
				"service":     "orderflow",
				"endpoint":    "/api/deployments",
				"environment": "production",
				"description": "Provides deployment IDs, versions, deployment timestamps, and status for correlation.",
			}
			bytes, err := json.MarshalIndent(deployMeta, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal resource: %w", err)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(bytes),
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 5. Runbook: runbook://orderflow
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "runbook://orderflow",
			Name:        "runbook-orderflow",
			Title:       "OrderFlow Service Runbook",
			Description: "Operational runbook for OrderFlow triage, symptoms, impact, and verification.",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			content := loadRunbook("orderflow.md", embeddedOrderflowRunbook)
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "text/markdown",
						Text:     content,
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 6. Runbook: runbook://redis
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "runbook://redis",
			Name:        "runbook-redis",
			Title:       "Redis Cache Runbook",
			Description: "Operational runbook for Redis cache degradation and database fallback triage.",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			content := loadRunbook("redis.md", embeddedRedisRunbook)
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "text/markdown",
						Text:     content,
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 7. Runbook: runbook://postgresql
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "runbook://postgresql",
			Name:        "runbook-postgresql",
			Title:       "PostgreSQL Database Runbook",
			Description: "Operational runbook for PostgreSQL database connection failure and pool exhaustion.",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			content := loadRunbook("postgresql.md", embeddedPostgresqlRunbook)
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "text/markdown",
						Text:     content,
					},
				},
			}, nil
		},
	)

	// --------------------------------------------------
	// 8. Runbook: runbook://payment
	// --------------------------------------------------
	server.AddResource(
		&mcp.Resource{
			URI:         "runbook://payment",
			Name:        "runbook-payment",
			Title:       "Payment Provider Runbook",
			Description: "Operational runbook for external payment provider timeouts and failures.",
			MIMEType:    "text/markdown",
		},
		func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			content := loadRunbook("payment.md", embeddedPaymentRunbook)
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "text/markdown",
						Text:     content,
					},
				},
			}, nil
		},
	)
}
