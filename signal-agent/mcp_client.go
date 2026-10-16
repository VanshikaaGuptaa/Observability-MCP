package main

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os"
	"os/exec"
)

type MCPClient struct {
	session *mcp.ClientSession
}

func ConnectMCP(ctx context.Context, cfg Config) (*MCPClient, error) {
	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "signalsleuth-agent",
			Version: "0.1.0",
		},
		nil,
	)

	cmd := exec.CommandContext(
		ctx,
		cfg.MCPCommand,
		"run",
		".",
	)
	cmd.Dir = cfg.MCPWorkingDir

	// The MCP protocol uses stdout. Send process diagnostics to
	// the agent's stderr, never into the protocol stream.
	cmd.Stderr = os.Stderr

	transport := &mcp.CommandTransport{
		Command: cmd,
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to MCP server: %w", err)
	}

	return &MCPClient{
		session: session,
	}, nil
}

func (c *MCPClient) Close() error {
	return c.session.Close()
}

func (c *MCPClient) InvestigateIncident(
	ctx context.Context,
	service string,
	duration string,
) (any, error) {
	result, err := c.session.CallTool(
		ctx,
		&mcp.CallToolParams{
			Name: "investigate_incident",
			Arguments: map[string]any{
				"service":  service,
				"duration": duration,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("call investigate_incident: %w", err)
	}

	if result.IsError {
		return nil, fmt.Errorf("MCP tool returned an error")
	}

	if result.StructuredContent != nil {
		return result.StructuredContent, nil
	}

	// Fallback for servers that return textual tool content.
	return result.Content, nil
}
