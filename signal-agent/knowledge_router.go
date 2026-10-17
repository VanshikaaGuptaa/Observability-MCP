package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

const KnowledgeSkipThreshold = 0.90

type KnowledgeDecision struct {
	SkipAI          bool             `json:"ai_skipped"`
	Mode            string           `json:"investigation_mode"`
	MatchScore      float64          `json:"match_score"`
	MatchedArticle  *KnowledgeArticle `json:"matched_article,omitempty"`
	DecisionReason  string           `json:"decision_reason"`
}

// RouteKnowledge selects the best verified article for the current evidence.
// Unverified articles may be shown in the UI, but cannot bypass AI diagnosis.
func RouteKnowledge(
	service string,
	evidence any,
	articles []KnowledgeArticle,
) KnowledgeDecision {
	var best *KnowledgeArticle
	bestScore := 0.0

	for i := range articles {
		article := &articles[i]

		if !article.Verified {
			continue
		}

		score, eligible := scoreArticle(service, evidence, *article)
		if eligible && score > bestScore {
			bestScore = score
			best = article
		}
	}

	if best == nil {
		return KnowledgeDecision{
			Mode:           "ai",
			DecisionReason: "No verified article matches the current incident sufficiently.",
		}
	}

	if bestScore < KnowledgeSkipThreshold {
		return KnowledgeDecision{
			Mode:           "ai",
			MatchScore:     bestScore,
			MatchedArticle: best,
			DecisionReason: "The best verified article is below the skip threshold.",
		}
	}

	return KnowledgeDecision{
		SkipAI:          true,
		Mode:            "knowledge",
		MatchScore:      bestScore,
		MatchedArticle:  best,
		DecisionReason: "A verified article matches the current incident above the threshold.",
	}
}

func scoreArticle(
	service string,
	evidence any,
	article KnowledgeArticle,
) (float64, bool) {
	signature := parseSignature(article.ErrorSignature)
	if len(signature) == 0 {
		return 0, false
	}
log.Printf(
    "Checking knowledge article=%q verified=%v signature=%q",
    article.Title,
    article.Verified,
    article.ErrorSignature,
)
	raw, err := json.Marshal(evidence)
	if err != nil {
		
		return 0, false
	}

	text := strings.ToLower(string(raw))
	service = strings.ToLower(strings.TrimSpace(service))

	// Critical dimensions must be explicitly present in the signature
	// and consistent with the live evidence.
	dependency := signature["dependency"]
	failure := signature["failure"]
	endpoint := signature["endpoint"]
	signatureService := signature["service"]

	if signatureService == "" || dependency == "" || failure == "" {
		log.Printf(
    "Checking knowledge article=%q verified=%v signature=%q",
    article.Title,
    article.Verified,
    article.ErrorSignature,
)
		return 0, false
	}

	if signatureService != service {
		log.Printf(
    "Checking knowledge article=%q verified=%v signature=%q",
    article.Title,
    article.Verified,
    article.ErrorSignature,
)
		return 0, false
	}

	if !evidenceContains(text, dependency) ||
		!evidenceContains(text, failure) {
			log.Printf(
    "Checking knowledge article=%q verified=%v signature=%q",
    article.Title,
    article.Verified,
    article.ErrorSignature,
)
		return 0, false
	}

	score := 0.60 // service + dependency + failure type matched

	if endpoint != "" &&
		endpoint != "unknown" &&
		evidenceContains(text, endpoint) {
		score += 0.20
	} else {
		// Endpoint-specific articles cannot be treated as a full match
		// when the evidence does not confirm that endpoint.
		return score, true
	}

	// The article must not contradict the live dependency status.
	// A healthy current dependency is not necessarily a contradiction
	// for historical incidents, so this check is deliberately conservative.
	if evidenceHasContradictoryStatus(text, dependency) {
		log.Printf(
    "Checking knowledge article=%q verified=%v signature=%q",
    article.Title,
    article.Verified,
    article.ErrorSignature,
)
		return 0, false
	}

	// Remaining score is reserved for corroborating evidence.
	// Do not award it merely because a generic keyword appears.
	if evidenceContains(text, failure) &&
		evidenceContains(text, endpoint) {
		score += 0.20
	}

	return score, true
}


func evidenceContains(text, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "unknown" {
		return false
	}
	return strings.Contains(text, value)
}

func evidenceHasContradictoryStatus(text, dependency string) bool {
	// Fail closed if the payload contains an explicit healthy status for
	// the named dependency. A later improvement can distinguish current
	// health from historical window status using structured JSON fields.
	dependency = strings.ToLower(strings.TrimSpace(dependency))
	if dependency == "" {
		return true
	}

	hasDependency := strings.Contains(text, dependency)
	hasHealthy := strings.Contains(text, `"current_status":"healthy"`) ||
		strings.Contains(text, `"current_status": "healthy"`)

	return hasDependency && hasHealthy
}

func deterministicKnowledgeExplanation(
	service string,
	evidence any,
	article KnowledgeArticle,
	score float64,
) string {
	raw, err := json.Marshal(evidence)
	if err != nil {
		return fmt.Sprintf(
			"Knowledge-based investigation for %s.\n\nVerified article: %s",
			service,
			article.Title,
		)
	}

	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Sprintf(
			"Knowledge-based investigation for %s.\n\nVerified article: %s",
			service,
			article.Title,
		)
	}

	// MCP evidence may be nested under "evidence".
	payload := root
	if nested, ok := root["evidence"].(map[string]any); ok {
		payload = nested
	}

	var signals []any
	if list, ok := payload["evidence"].([]any); ok {
		signals = list
	}

	getSignal := func(name string) any {
		for _, item := range signals {
			signal, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if strings.EqualFold(fmt.Sprint(signal["signal"]), name) {
				return signal["value"]
			}
		}
		return nil
	}

	status := fmt.Sprint(payload["status"])
	if status == "<nil>" {
		status = "unknown"
	}

	severity := fmt.Sprint(payload["severity"])
	if severity == "<nil>" {
		severity = "unspecified"
	}

	errorRate := getSignal("error_rate")
	errorLogs := getSignal("error_logs")
	dependency := getSignal("postgresql")
	endpointHealth := getSignal("endpoint_health")

	var summary strings.Builder

	fmt.Fprintf(&summary,
		"Knowledge-based investigation for %s.\n\n", service)

	fmt.Fprintf(&summary,
		"**Incident status:** %s\n", status)
	fmt.Fprintf(&summary,
		"**Severity:** %s\n\n", severity)

	fmt.Fprintf(&summary,
		"**Matched verified runbook:** %s\n", article.Title)
	fmt.Fprintf(&summary,
		"**Knowledge match score:** %.0f%%\n\n", score*100)

	summary.WriteString("**Evidence summary**\n")

	if errorRate != nil {
		fmt.Fprintf(&summary,
			"- Error rate: %v\n", errorRate)
	}
	if errorLogs != nil {
		fmt.Fprintf(&summary,
			"- Error logs: %v\n", errorLogs)
	}
	if dependency != nil {
		fmt.Fprintf(&summary,
			"- PostgreSQL dependency: %v\n", dependency)
	}
	if endpointHealth != nil {
		fmt.Fprintf(&summary,
			"- Endpoint health: %v\n", endpointHealth)
	}

	if article.Summary != "" {
		fmt.Fprintf(&summary,
			"\n**Runbook guidance**\n%s\n", article.Summary)
	}

	if len(article.Steps) > 0 {
		summary.WriteString("\n**Recommended checks**\n")
		for i, step := range article.Steps {
			fmt.Fprintf(&summary, "%d. %s\n", i+1, step)
		}
	}

	summary.WriteString(
		"\n**Investigation method:** AI diagnosis was skipped because a verified knowledge article met the configured match threshold. " +
			"The match score measures agreement with the current matching rules; it is not a probability that the root cause is correct.\n",
	)

	return summary.String()
}