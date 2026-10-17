package main

import (
	"encoding/json"
	"strings"
)

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte{}
	}
	return data
}

type KnowledgeArticle struct {
	Title          string   `json:"title"`
	Summary        string   `json:"summary"`
	MatchedOn      []string `json:"matched_on"`
	Steps          []string `json:"steps"`
	ErrorSignature string   `json:"error_signature,omitempty"`
	Source         string   `json:"source,omitempty"`
	Verified       bool     `json:"verified"`
	MatchOrigin    string   `json:"match_origin,omitempty"`
}

type runbook struct {
	title   string
	summary string
	steps   []string
}

var runbooks = []runbook{
	{
		title:   "PostgreSQL Connection Failure",
		summary: "Troubleshoot database connectivity and order creation failures.",
		steps: []string{
			"Check whether PostgreSQL is running and reachable.",
			"Inspect database connection settings and connection-pool errors.",
			"Check Orderflow logs for the underlying database error.",
			"Retry order creation after connectivity is restored.",
		},
	},
	{
		title:   "Redis Connectivity and Cache Failures",
		summary: "Investigate Redis connection errors and cache degradation.",
		steps: []string{
			"Check Redis availability and connectivity.",
			"Inspect Redis errors in Orderflow logs.",
			"Verify cache operations recover after connectivity is restored.",
		},
	},
	{
		title:   "Payment Processing Failures",
		summary: "Investigate failed payment requests and payment dependency errors.",
		steps: []string{
			"Inspect payment endpoint errors and recent logs.",
			"Check payment dependency availability and configuration.",
			"Retry a test payment after the failure is resolved.",
		},
	},
	{
		title:   "HTTP 5xx Error Spike",
		summary: "Find the endpoints responsible for server errors.",
		steps: []string{
			"Identify endpoints with elevated 5xx error rates.",
			"Correlate failures with logs and dependency errors.",
			"Verify error rates return to normal after remediation.",
		},
	},
	{
		title:   "High Latency and Slow Requests",
		summary: "Investigate slow endpoints and increased request latency.",
		steps: []string{
			"Identify endpoints with elevated P95 or P99 latency.",
			"Inspect logs and dependencies for slow operations.",
			"Repeat the latency check after addressing the bottleneck.",
		},
	},
}

// FindKnowledge returns all static runbooks relevant to the evidence.
func FindKnowledge(evidence any) []KnowledgeArticle {
	rawBytes := mustJSON(evidence)
	raw := strings.ToLower(string(rawBytes))

	var data any
	_ = json.Unmarshal(rawBytes, &data)

	var matches []KnowledgeArticle

	add := func(book runbook, reason string) {
		matches = append(matches, KnowledgeArticle{
			Title:       book.title,
			Summary:     book.summary,
			MatchedOn:   []string{reason},
			Steps:       book.steps,
			Source:      "static",
			Verified:    true,
			MatchOrigin: "static",
		})
	}

	// Match dependency-specific failures independently.
	if containsAny(raw,
		"connection refused",
		"database connection failed",
		"postgresql connection failed",
		"postgres connection failed",
		"database operation failed",
		"postgresql dependency degraded",
		"postgres dependency degraded",
		"dependency=postgresql",
		"dependency=postgres",
	) {
		add(runbooks[0], "PostgreSQL/database failure evidence")
	}

	if containsAny(raw,
		"redis_failure",
		"redis connection refused",
		"redis connectivity failed",
		"redis dependency degraded",
		"dependency=redis",
	) {
		add(runbooks[1], "Redis failure or dependency degradation")
	}

	if containsAny(raw,
		"payment_failure",
		"payment processing failed",
		"payment dependency degraded",
		"/api/payment",
	) {
		add(runbooks[2], "Payment failure evidence")
	}

	errorRate, hasErrorRate := findNumber(data, "error_rate")
	hasHTTP500 := containsAny(raw,
		"http 500",
		`"status":500`,
		`"status": 500`,
		`"status_code":500`,
		`"status_code": 500`,
	)

	if (hasErrorRate && errorRate > 0.05) || hasHTTP500 {
		add(runbooks[3], "Elevated HTTP error rate or HTTP 500")
	}

	p95, hasP95 := findNumber(data, "p95_latency")
	p99, hasP99 := findNumber(data, "p99_latency")

	// These thresholds assume latency is reported in seconds.
	if (hasP95 && p95 > 0.5) || (hasP99 && p99 > 1.0) {
		add(runbooks[4], "High P95/P99 latency")
	}

	return matches
}

func containsAny(text string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, strings.ToLower(phrase)) {
			return true
		}
	}
	return false
}

// findNumber recursively searches nested evidence for a numeric field.
func findNumber(value any, targetKey string) (float64, bool) {
	switch item := value.(type) {
	case map[string]any:
		for key, val := range item {
			if strings.EqualFold(key, targetKey) {
				switch number := val.(type) {
				case float64:
					return number, true
				case int:
					return float64(number), true
				}
			}

			if number, ok := findNumber(val, targetKey); ok {
				return number, true
			}
		}

	case []any:
		for _, val := range item {
			if number, ok := findNumber(val, targetKey); ok {
				return number, true
			}
		}
	}

	return 0, false
}
