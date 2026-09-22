package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-mcp/internal/errors"
	investigation "observability-mcp/internal/investigation"
	"observability-mcp/internal/validation"
)

type InvestigateServiceInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 10m or 1h"`
}

type InvestigateIncidentInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 10m or 1h"`
}

func (s *Server) InvestigateService(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input InvestigateServiceInput,
) (*mcp.CallToolResult, *investigation.Investigation, error) {

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, nil, err
	}

	if input.Duration == "" {
		input.Duration = "10m"
	}

	duration, err := validation.ValidateDuration(input.Duration)
	if err != nil {
		return nil, nil, err
	}

	result, err := s.investigation.Investigate(
		ctx,
		input.Service,
		duration,
	)
	if err != nil {
		return nil, nil, errors.NewInternalError(
			fmt.Sprintf("investigation failed for service %s", input.Service),
			err,
		)
	}

	return nil, result, nil
}

func (s *Server) InvestigateIncident(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input InvestigateIncidentInput,
) (*mcp.CallToolResult, *investigation.Incident, error) {

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, nil, err
	}

	if input.Duration == "" {
		input.Duration = "10m"
	}

	duration, err := validation.ValidateDuration(input.Duration)
	if err != nil {
		return nil, nil, err
	}

	investigationResult, err := s.investigation.Investigate(
		ctx,
		input.Service,
		duration,
	)
	if err != nil {
		return nil, nil, errors.NewInternalError(
			fmt.Sprintf("investigation failed for service %s", input.Service),
			err,
		)
	}

	incident := investigation.BuildIncident(investigationResult)
	if incident == nil {
		return nil, nil, errors.NewInternalError("unable to build incident from investigation", nil)
	}

	return nil, incident, nil
}
