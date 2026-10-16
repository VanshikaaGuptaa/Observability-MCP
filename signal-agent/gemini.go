package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
    "time"
	"google.golang.org/genai"
)

type GeminiAnalyzer struct {
	client *genai.Client
	model  string
}

func NewGeminiAnalyzer(cfg Config) (*GeminiAnalyzer, error) {
	client, err := genai.NewClient(
		context.Background(),
		&genai.ClientConfig{
			APIKey: cfg.GeminiAPIKey,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create Gemini client: %w", err)
	}

	return &GeminiAnalyzer{
		client: client,
		model:  cfg.GeminiModel,
	}, nil
}

func (g *GeminiAnalyzer) Analyze(
    ctx context.Context,
    service string,
    duration string,
    evidence any,
) (string, error) {
    evidenceJSON, err := json.MarshalIndent(evidence, "", "  ")
    if err != nil {
        return "", fmt.Errorf("serialize MCP evidence: %w", err)
    }

    prompt := fmt.Sprintf(`
You are SignalSleuth AI, an assistant for investigating software incidents.

Analyze the supplied monitoring evidence.

Rules:
- Use only facts supported by the evidence.
- Separate observations from hypotheses.
- Do not invent metrics, logs, or timestamps.
- State uncertainty when evidence is incomplete.
- Recommend safe diagnostic checks.
- Be concise.

Return these sections:
Summary
Observed evidence
Leading hypothesis
Alternative explanations
Recommended next checks
Remaining uncertainty

Service: %s
Time window: %s

Evidence:
%s
`, service, duration, string(evidenceJSON))

    delays := []time.Duration{
        1 * time.Second,
        2 * time.Second,
        4 * time.Second,
    }

    var lastErr error

    for attempt := 0; attempt <= len(delays); attempt++ {
        response, err := g.client.Models.GenerateContent(
            ctx,
            g.model,
            genai.Text(prompt),
            nil,
        )

        if err == nil {
            if response == nil || strings.TrimSpace(response.Text()) == "" {
                return "", fmt.Errorf("Gemini returned an empty response")
            }
            return response.Text(), nil
        }

        lastErr = err

        // Do not retry invalid requests, authentication failures,
        // or other permanent client errors.
        errText := strings.ToLower(err.Error())
        retryable := strings.Contains(errText, "503") ||
            strings.Contains(errText, "unavailable") ||
            strings.Contains(errText, "429") ||
            strings.Contains(errText, "resource_exhausted") ||
            strings.Contains(errText, "500") ||
            strings.Contains(errText, "502") ||
            strings.Contains(errText, "504")

        if !retryable || attempt == len(delays) {
            break
        }

        delay := delays[attempt]
        fmt.Printf(
            "Gemini request failed (attempt %d/%d); retrying in %s: %v\n",
            attempt+1,
            len(delays)+1,
            delay,
            err,
        )

        timer := time.NewTimer(delay)
        select {
        case <-ctx.Done():
            timer.Stop()
            return "", ctx.Err()
        case <-timer.C:
        }
    }

    return "", fmt.Errorf(
        "Gemini failed after retries: %w",
        lastErr,
    )
}