package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"observability-mcp/internal/errors"
	"observability-mcp/internal/validation"
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type SearchLogsInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Query    string `json:"query" jsonschema:"text to search for in the logs"`
	Duration string `json:"duration" jsonschema:"the time window such as 10m or 1h"`
}

type SearchLogsOutput struct {
	Service  string     `json:"service"`
	Query    string     `json:"query"`
	Duration string     `json:"duration"`
	Logs     []LogEntry `json:"logs"`
}

type RecentErrorsInput struct {
	Service  string `json:"service" jsonschema:"the name of the service"`
	Duration string `json:"duration" jsonschema:"the time window such as 10m or 1h"`
}

type RecentErrorsOutput struct {
	Service  string     `json:"service"`
	Duration string     `json:"duration"`
	Errors   []LogEntry `json:"errors"`
}

func (s *Server) SearchLogs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchLogsInput,
) (*mcp.CallToolResult, SearchLogsOutput, error) {

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, SearchLogsOutput{}, err
	}

	if err := validation.ValidateQuery(input.Query); err != nil {
		return nil, SearchLogsOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "10m"
	}

	duration, err := validation.ValidateDuration(input.Duration)
	if err != nil {
		return nil, SearchLogsOutput{}, err
	}

	end := time.Now()
	start := end.Add(-duration)

	logQL := fmt.Sprintf(
		`{service_name="%s"} |= %q`,
		input.Service,
		input.Query,
	)

	result, err := s.loki.QueryRange(
		ctx,
		logQL,
		start,
		end,
		50,
	)

	if err != nil {
		return nil, SearchLogsOutput{}, errors.NewBackendUnavailableError("loki", err)
	}

	output := SearchLogsOutput{
		Service:  input.Service,
		Query:    input.Query,
		Duration: input.Duration,
		Logs:     []LogEntry{},
	}

	if result != nil {
		for _, stream := range result.Data.Result {
			for _, value := range stream.Values {
				if len(value) < 2 {
					continue
				}

				timestamp := value[0]
				rawLog := value[1]

				var parsed struct {
					Time  string `json:"time"`
					Level string `json:"level"`
					Msg   string `json:"msg"`
				}

				if err := json.Unmarshal([]byte(rawLog), &parsed); err != nil {
					// Fallback to raw log as message if not JSON formatted
					output.Logs = append(
						output.Logs,
						LogEntry{
							Timestamp: timestamp,
							Level:     "INFO",
							Message:   rawLog,
						},
					)
					continue
				}

				output.Logs = append(
					output.Logs,
					LogEntry{
						Timestamp: timestamp,
						Level:     parsed.Level,
						Message:   parsed.Msg,
					},
				)
			}
		}
	}

	return nil, output, nil
}

func (s *Server) GetRecentErrors(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RecentErrorsInput,
) (*mcp.CallToolResult, RecentErrorsOutput, error) {

	if err := validation.ValidateService(input.Service); err != nil {
		return nil, RecentErrorsOutput{}, err
	}

	if input.Duration == "" {
		input.Duration = "10m"
	}

	duration, err := validation.ValidateDuration(input.Duration)
	if err != nil {
		return nil, RecentErrorsOutput{}, err
	}

	end := time.Now()
	start := end.Add(-duration)

	logQL := fmt.Sprintf(
		`{service_name="%s"} | json | level="ERROR"`,
		input.Service,
	)

	result, err := s.loki.QueryRange(
		ctx,
		logQL,
		start,
		end,
		50,
	)

	if err != nil {
		return nil, RecentErrorsOutput{}, errors.NewBackendUnavailableError("loki", err)
	}

	output := RecentErrorsOutput{
		Service:  input.Service,
		Duration: input.Duration,
		Errors:   []LogEntry{},
	}

	if result != nil {
		for _, stream := range result.Data.Result {
			for _, value := range stream.Values {
				if len(value) < 2 {
					continue
				}

				timestamp := value[0]
				rawLog := value[1]

				var parsed struct {
					Time  string `json:"time"`
					Level string `json:"level"`
					Msg   string `json:"msg"`
				}

				if err := json.Unmarshal([]byte(rawLog), &parsed); err != nil {
					output.Errors = append(
						output.Errors,
						LogEntry{
							Timestamp: timestamp,
							Level:     "ERROR",
							Message:   rawLog,
						},
					)
					continue
				}

				output.Errors = append(
					output.Errors,
					LogEntry{
						Timestamp: timestamp,
						Level:     parsed.Level,
						Message:   parsed.Msg,
					},
				)
			}
		}
	}

	return nil, output, nil
}
