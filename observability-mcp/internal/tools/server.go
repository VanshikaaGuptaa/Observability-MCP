package tools

import (
	deployments "observability-mcp/internal/deployments"
	investigation "observability-mcp/internal/investigation"
	loki "observability-mcp/internal/loki"
	prom "observability-mcp/internal/prometheus"
)

type Server struct {
	prometheus    *prom.Client
	loki          *loki.Client
	deployments   *deployments.Client
	investigation *investigation.Service
}

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
