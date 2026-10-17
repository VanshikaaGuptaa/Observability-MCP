package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"
)

// textGenerator is the minimal Gemini capability the analyzer needs.
// It exists so tests can substitute a fake without network access.
type textGenerator interface {
	GenerateText(
		ctx context.Context,
		model string,
		prompt string,
		maxOutputTokens int32,
	) (string, error)
}

// generatorFactory creates a generator bound to a specific API key.
type generatorFactory func(ctx context.Context, apiKey string) (textGenerator, error)

type genaiGenerator struct {
	client *genai.Client
}

func (g *genaiGenerator) GenerateText(
	ctx context.Context,
	model string,
	prompt string,
	maxOutputTokens int32,
) (string, error) {
	var config *genai.GenerateContentConfig
	if maxOutputTokens > 0 {
		config = &genai.GenerateContentConfig{MaxOutputTokens: maxOutputTokens}
	}

	response, err := g.client.Models.GenerateContent(
		ctx,
		model,
		genai.Text(prompt),
		config,
	)
	if err != nil {
		return "", err
	}
	if response == nil {
		return "", nil
	}
	return response.Text(), nil
}

func newGenaiGenerator(ctx context.Context, apiKey string) (textGenerator, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	return &genaiGenerator{client: client}, nil
}

type GeminiAnalyzer struct {
	keys         *GeminiKeyStore
	model        string
	newGenerator generatorFactory
	retryDelays  []time.Duration

	// The client is cached per key so that a key change produces a new
	// client and no stale client keeps using an old key.
	mu        sync.Mutex
	cachedKey string
	cached    textGenerator
}

func NewGeminiAnalyzer(cfg Config, keys *GeminiKeyStore) *GeminiAnalyzer {
	return &GeminiAnalyzer{
		keys:         keys,
		model:        cfg.GeminiModel,
		newGenerator: newGenaiGenerator,
		retryDelays: []time.Duration{
			1 * time.Second,
			2 * time.Second,
			4 * time.Second,
		},
	}
}

// generatorFor returns a generator bound to key. The lock is held only
// while selecting/creating the client, never during a Gemini request.
func (g *GeminiAnalyzer) generatorFor(
	ctx context.Context,
	key string,
) (textGenerator, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cached != nil && g.cachedKey == key {
		return g.cached, nil
	}

	gen, err := g.newGenerator(ctx, key)
	if err != nil {
		log.Printf("Gemini client creation failed: %s", redactSecret(err.Error(), key))
		return nil, &GeminiError{
			Kind:    "client",
			Message: "Could not initialise the Gemini client.",
		}
	}

	g.cachedKey = key
	g.cached = gen
	return gen, nil
}

func (g *GeminiAnalyzer) Analyze(
	ctx context.Context,
	service string,
	duration string,
	evidence any,
) (string, error) {
	// Snapshot the key once so every retry in this request uses the
	// same key, even if Settings change concurrently.
	key, source := g.keys.Active()
	if key == "" {
		return "", ErrGeminiNotConfigured
	}

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

	gen, err := g.generatorFor(ctx, key)
	if err != nil {
		return "", err
	}

	var lastErr *GeminiError

	for attempt := 0; attempt <= len(g.retryDelays); attempt++ {
		text, err := gen.GenerateText(ctx, g.model, prompt, 0)

		if err == nil {
			if strings.TrimSpace(text) == "" {
				return "", &GeminiError{
					Kind:    "empty",
					Message: "Gemini returned an empty response.",
				}
			}
			return text, nil
		}

		lastErr = classifyGeminiError(err)

		// Do not retry invalid requests, authentication failures,
		// or other permanent client errors.
		if !lastErr.Retryable || attempt == len(g.retryDelays) {
			break
		}

		delay := g.retryDelays[attempt]
		log.Printf(
			"Gemini request failed (attempt %d/%d, key source=%s); retrying in %s: %s",
			attempt+1,
			len(g.retryDelays)+1,
			source,
			delay,
			redactSecret(err.Error(), key),
		)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}

	log.Printf(
		"Gemini analysis failed (key source=%s): %s",
		source,
		lastErr.Message,
	)
	return "", lastErr
}

// TestConnection performs one minimal generation request with the active
// key and model. It does not call MCP or run an investigation.
func (g *GeminiAnalyzer) TestConnection(ctx context.Context) (string, error) {
	key, source := g.keys.Active()
	if key == "" {
		return source, ErrGeminiNotConfigured
	}

	gen, err := g.generatorFor(ctx, key)
	if err != nil {
		return source, err
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	_, err = gen.GenerateText(ctx, g.model, "Reply with the single word OK.", 16)
	if err != nil {
		safe := classifyGeminiError(err)
		log.Printf(
			"Gemini connection test failed (key source=%s): %s | %s",
			source,
			safe.Message,
			redactSecret(err.Error(), key),
		)
		return source, safe
	}
	return source, nil
}

// GeminiError is a sanitized error that is safe to show to users.
type GeminiError struct {
	Kind      string
	Message   string
	Retryable bool
}

func (e *GeminiError) Error() string { return e.Message }

// classifyGeminiError converts an upstream error into a safe message.
// The raw upstream text is never included in the result.
func classifyGeminiError(err error) *GeminiError {
	if errors.Is(err, context.DeadlineExceeded) {
		return &GeminiError{Kind: "timeout", Message: "The Gemini request timed out.", Retryable: true}
	}
	if errors.Is(err, context.Canceled) {
		return &GeminiError{Kind: "canceled", Message: "The Gemini request was cancelled."}
	}

	code := 0
	var apiErr genai.APIError
	var apiErrPtr *genai.APIError
	if errors.As(err, &apiErr) {
		code = apiErr.Code
	} else if errors.As(err, &apiErrPtr) && apiErrPtr != nil {
		code = apiErrPtr.Code
	}

	text := strings.ToLower(err.Error())
	has := func(parts ...string) bool {
		for _, p := range parts {
			if strings.Contains(text, p) {
				return true
			}
		}
		return false
	}

	switch {
	case has("api_key_invalid", "api key not valid", "api key expired", "invalid api key"),
		code == 401, (code == 0 && has("401", "unauthenticated")):
		return &GeminiError{Kind: "invalid_key", Message: "Gemini rejected the API key. Check that the key is correct and active."}
	case code == 403, (code == 0 && has("403", "permission_denied")):
		return &GeminiError{Kind: "permission", Message: "The API key does not have permission to use the configured Gemini model."}
	case code == 404, (code == 0 && has("404", "not_found")):
		return &GeminiError{Kind: "model", Message: "The configured Gemini model was not found or is unavailable for this key."}
	case code == 429, (code == 0 && has("429", "resource_exhausted")):
		return &GeminiError{Kind: "rate_limit", Message: "Gemini rate limit or quota exceeded. Try again later.", Retryable: true}
	case code == 400, (code == 0 && has("400", "invalid_argument")):
		return &GeminiError{Kind: "bad_request", Message: "Gemini rejected the request as invalid."}
	case code >= 500, (code == 0 && has("500", "502", "503", "504", "unavailable", "internal")):
		return &GeminiError{Kind: "unavailable", Message: "Gemini is temporarily unavailable. Try again shortly.", Retryable: true}
	}

	return &GeminiError{Kind: "unknown", Message: "The Gemini request failed."}
}

// redactSecret removes key material from text intended for logs.
func redactSecret(text, secret string) string {
	if secret == "" {
		return text
	}
	return strings.ReplaceAll(text, secret, "[REDACTED]")
}
