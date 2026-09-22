package tools

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-mcp/internal/errors"
	"observability-mcp/internal/validation"
)

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

func (s *Server) GetRecentDeployments(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RecentDeploymentsInput,
) (*mcp.CallToolResult, RecentDeploymentsOutput, error) {

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, RecentDeploymentsOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "24h"
	}

	duration, err := validation.ValidateDuration(input.Duration)
	if err != nil {
		return nil, RecentDeploymentsOutput{}, err
	}

	deployments, err := s.deployments.GetDeployments(ctx)
	if err != nil {
		return nil, RecentDeploymentsOutput{}, errors.NewBackendUnavailableError("orderflow-deployments", err)
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
