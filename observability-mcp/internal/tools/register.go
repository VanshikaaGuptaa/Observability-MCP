package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterTools registers all 9 observability MCP tools on the given MCP server.
func RegisterTools(mcpServer *mcp.Server, app *Server) {
	// 1. get_service_health
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_service_health",
			Description: "Get the current health status of an observability service (healthy, down, unknown).",
		},
		app.GetServiceHealth,
	)

	// 2. get_request_rate
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_request_rate",
			Description: "Get the average HTTP request rate of an observability service over a specified time window.",
		},
		app.GetRequestRate,
	)

	// 3. get_error_rate
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_error_rate",
			Description: "Get the HTTP 5xx error rate of an observability service over a specified time window.",
		},
		app.GetErrorRate,
	)

	// 4. get_latency
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_latency",
			Description: "Get P50, P95, and P99 HTTP latency for an observability service endpoint.",
		},
		app.GetLatency,
	)

	// 5. search_logs
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "search_logs",
			Description: "Search logs for an observability service over a specified time window.",
		},
		app.SearchLogs,
	)

	// 6. get_recent_errors
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_recent_errors",
			Description: "Get recent error logs for an observability service.",
		},
		app.GetRecentErrors,
	)

	// 7. get_recent_deployments
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "get_recent_deployments",
			Description: "Get recent deployments for an observability service.",
		},
		app.GetRecentDeployments,
	)

	// 8. investigate_service
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "investigate_service",
			Description: "Investigate a service by correlating health, request rate, error rate, latency, recent errors, and recent deployments.",
		},
		app.InvestigateService,
	)

	// 9. investigate_incident
	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "investigate_incident",
			Description: "Analyze observability evidence and produce a structured incident containing symptoms, evidence, timeline, and possible causes.",
		},
		app.InvestigateIncident,
	)
}
