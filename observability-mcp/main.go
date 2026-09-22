package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	deployments "observability-mcp/internal/deployments"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
	"observability-mcp/internal/resources"
	"observability-mcp/internal/tools"
)

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

	app := tools.NewServer(
		prometheusClient,
		lokiClient,
		deploymentClient,
	)

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "observability-mcp",
			Version: "1.0.0",
		},
		nil,
	)

	// Register MCP Tools
	tools.RegisterTools(mcpServer, app)

	// Register MCP Resources (topology and runbooks)
	resources.RegisterResources(mcpServer)

	// Run MCP server over standard I/O transport
	if err := mcpServer.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		log.Fatal(err)
	}
}
