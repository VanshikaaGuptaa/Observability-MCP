
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
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	MatchedOn []string `json:"matched_on"`
	Steps     []string `json:"steps"`
	
ErrorSignature string `json:"error_signature,omitempty"`
Source         string `json:"source,omitempty"`
Verified       bool   `json:"verified"`

}

type runbook struct {
	title    string
	summary  string
	keywords []string
	steps    []string
}

var runbooks = []runbook{
	{
		title:   "PostgreSQL Connection Failure",
		summary: "Troubleshoot database connectivity and order creation failures.",
		keywords: []string{
			"postgresql", "postgres", "database",
			"connection refused", "database failure",
		},
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
		keywords: []string{
			"redis", "cache failure", "redis_failure",
		},
		steps: []string{
			"Check Redis availability and connectivity.",
			"Inspect Redis errors in Orderflow logs.",
			"Verify cache operations recover after connectivity is restored.",
		},
	},
	{
		title:   "Payment Processing Failures",
		summary: "Investigate failed payment requests and payment dependency errors.",
		keywords: []string{
			"payment", "payment_failure", "/api/payment",
		},
		steps: []string{
			"Inspect payment endpoint errors and recent logs.",
			"Check payment dependency availability and configuration.",
			"Retry a test payment after the failure is resolved.",
		},
	},
	{
		title:   "HTTP 5xx Error Spike",
		summary: "Find the endpoints responsible for server errors.",
		keywords: []string{
			"error_rate", "error logs", "http 500", "5xx",
		},
		steps: []string{
			"Identify endpoints with elevated 5xx error rates.",
			"Correlate failures with logs and dependency errors.",
			"Verify error rates return to normal after remediation.",
		},
	},
	{
		title:   "High Latency and Slow Requests",
		summary: "Investigate slow endpoints and increased request latency.",
		keywords: []string{
			"p95_latency", "p99_latency", "high latency",
			"slow", "latency spike",
		},
		steps: []string{
			"Identify endpoints with elevated P95 or P99 latency.",
			"Inspect logs and dependencies for slow operations.",
			"Repeat the latency check after addressing the bottleneck.",
		},
	},
}

// FindKnowledge matches runbooks against the serialized investigation evidence.

func FindKnowledge(evidence any) []KnowledgeArticle {
	rawBytes := mustJSON(evidence)
	raw := strings.ToLower(string(rawBytes))

	var data any
	_ = json.Unmarshal(rawBytes, &data)

	var matches []KnowledgeArticle

	add := func(book runbook, matchedOn ...string) {
		matches = append(matches, KnowledgeArticle{
			Title:     book.title,
			Summary:   book.summary,
			MatchedOn: matchedOn,
			Steps:     book.steps,
		})
	}

	// PostgreSQL: require actual connection failure evidence.
	if strings.Contains(raw, "connection refused") ||
		strings.Contains(raw, "database connection refused") {
		add(runbooks[0], "connection refused")
	}

	// Redis: require specific Redis failure evidence.
	if strings.Contains(raw, "redis_failure") ||
		strings.Contains(raw, "redis connection refused") ||
		strings.Contains(raw, "redis connectivity failed") {
		add(runbooks[1], "redis failure evidence")
	}

	// Payments: require payment-specific failure evidence.
	if strings.Contains(raw, "payment_failure") ||
		strings.Contains(raw, "payment processing failed") ||
		strings.Contains(raw, "/api/payment") && strings.Contains(raw, "degraded") {
		add(runbooks[2], "payment failure evidence")
	}

	// HTTP 5xx: require evidence of server errors or a significant error rate.
	errorRate, hasErrorRate := findNumber(data, "error_rate")
	hasHTTP500 := strings.Contains(raw, "http 500") ||
		strings.Contains(raw, `"status":500`) ||
		strings.Contains(raw, `"status": 500`)

	if (hasErrorRate && errorRate > 0.05) || hasHTTP500 {
		add(runbooks[3], "elevated error rate or HTTP 500")
	}

	// Latency: only recommend this runbook when latency is actually high.
	p95, hasP95 := findNumber(data, "p95_latency")
	p99, hasP99 := findNumber(data, "p99_latency")

	if (hasP95 && p95 > 0.5) || (hasP99 && p99 > 1.0) {
		add(runbooks[4], "high P95/P99 latency")
	}

	return matches
}

// findNumber searches nested JSON objects for a numeric field.
func findNumber(value any, targetKey string) (float64, bool) {
	switch item := value.(type) {
	case map[string]any:
		for key, val := range item {
			if strings.EqualFold(key, targetKey) {
				if number, ok := val.(float64); ok {
					return number, true
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
