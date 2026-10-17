package main

import (
	"encoding/json"
	"os"
	"strings"
)

const knowledgeFile = "knowledge.json"

func LoadSavedKnowledge() ([]KnowledgeArticle, error) {
	data, err := os.ReadFile(knowledgeFile)
	if os.IsNotExist(err) {
		return []KnowledgeArticle{}, nil
	}
	if err != nil {
		return nil, err
	}

	var articles []KnowledgeArticle
	if len(strings.TrimSpace(string(data))) == 0 {
		return []KnowledgeArticle{}, nil
	}

	if err := json.Unmarshal(data, &articles); err != nil {
		return nil, err
	}

	return articles, nil
}

// SaveKnowledgeArticle returns true only when a new article was inserted.
func SaveKnowledgeArticle(article KnowledgeArticle) (bool, error) {
	articles, err := LoadSavedKnowledge()
	if err != nil {
		return false, err
	}

	for _, existing := range articles {
		if sameArticle(existing, article) {
			return false, nil
		}
	}

	article.MatchOrigin = ""
	articles = append(articles, article)

	data, err := json.MarshalIndent(articles, "", "  ")
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(knowledgeFile, data, 0644); err != nil {
		return false, err
	}

	return true, nil
}

func sameArticle(a, b KnowledgeArticle) bool {
	sigA := normalize(a.ErrorSignature)
	sigB := normalize(b.ErrorSignature)

	if sigA != "" && sigB != "" && sigA == sigB {
		return true
	}

	return normalize(a.Title) != "" &&
		normalize(a.Title) == normalize(b.Title)
}

// FindSavedKnowledge returns all saved articles that match the evidence.
func FindSavedKnowledge(evidence any) []KnowledgeArticle {
	articles, err := LoadSavedKnowledge()
	if err != nil {
		return []KnowledgeArticle{}
	}

	var matches []KnowledgeArticle

	for _, article := range articles {
		if articleMatchesEvidence(article, evidence) {
			article.MatchOrigin = "saved"
			matches = append(matches, article)
		}
	}

	return matches
}

func articleMatchesEvidence(article KnowledgeArticle, evidence any) bool {
	if strings.TrimSpace(article.ErrorSignature) == "" {
		return false
	}

	raw := mustJSON(evidence)
	evidenceText := strings.ToLower(string(raw))

	signature := strings.TrimSpace(article.ErrorSignature)

	// Legacy signatures are matched as phrases.
	if !strings.Contains(signature, "|") {
		return strings.Contains(evidenceText, strings.ToLower(signature))
	}

	fields := parseSignature(signature)
	if len(fields) == 0 {
		return false
	}

	// Evidence is normalized into searchable values and field values.
	var decoded any
	if json.Unmarshal(raw, &decoded) != nil {
		return false
	}

	values := collectEvidenceValues(decoded)
	matchedFields := 0

	for key, expected := range fields {
		expected = normalize(expected)

		if expected == "" || expected == "unknown" {
			continue
		}

		if key == "service" {
			// Service is known from the investigation request even if
			// the MCP evidence does not repeat it.
			if expected == "orderflow" &&
				strings.Contains(evidenceText, "orderflow") {
				matchedFields++
			}
			continue
		}

		if evidenceFieldMatches(key, expected, values, evidenceText) {
			matchedFields++
			continue
		}

		// A concrete mismatch in an identifiable field means this
		// article should not be reused for the current incident.
		if key == "dependency" && hasDifferentDependency(expected, values) {
			return false
		}

		if key == "endpoint" && hasDifferentEndpoint(expected, values) {
			return false
		}
	}

	// Require at least two meaningful matching fields. This avoids
	// matching a database article solely because "postgresql" appears
	// in a generic summary or timeline.
	return matchedFields >= 2
}

func parseSignature(signature string) map[string]string {
	result := make(map[string]string)

	for _, part := range strings.Split(signature, "|") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}

		key = normalize(key)
		value = normalize(value)

		if key != "" {
			result[key] = value
		}
	}

	return result
}

func collectEvidenceValues(value any) map[string][]string {
	result := make(map[string][]string)

	var walk func(any)
	walk = func(node any) {
		switch item := node.(type) {
		case map[string]any:
			for key, val := range item {
				k := normalize(key)
				if s, ok := val.(string); ok {
					result[k] = append(result[k], normalize(s))
				}
				walk(val)
			}
		case []any:
			for _, child := range item {
				walk(child)
			}
		}
	}

	walk(value)
	return result
}

func evidenceFieldMatches(
	key string,
	expected string,
	values map[string][]string,
	evidenceText string,
) bool {
	candidateKeys := map[string][]string{
		"dependency":   {"dependency", "dependency_name", "name"},
		"endpoint":     {"endpoint", "path", "route"},
		"failure":      {"failure_type", "event_type", "type", "error", "message", "msg"},
		"failure_type": {"failure_type", "event_type", "type", "error", "message", "msg"},
	}

	keys := candidateKeys[key]
	if len(keys) == 0 {
		return strings.Contains(evidenceText, expected)
	}

	for _, candidateKey := range keys {
		for _, value := range values[candidateKey] {
			if value == expected || strings.Contains(value, expected) {
				return true
			}
		}
	}

	return false
}

func hasDifferentDependency(expected string, values map[string][]string) bool {
	for _, value := range values["dependency"] {
		value = normalize(value)
		if value != "" && value != expected && value != "unknown" {
			return true
		}
	}
	for _, value := range values["dependency_name"] {
		value = normalize(value)
		if value != "" && value != expected && value != "unknown" {
			return true
		}
	}
	return false
}

func hasDifferentEndpoint(expected string, values map[string][]string) bool {
	for _, key := range []string{"endpoint", "path", "route"} {
		for _, value := range values[key] {
			value = normalize(value)
			if strings.HasPrefix(value, "/") &&
				value != expected &&
				!strings.Contains(value, expected) {
				return true
			}
		}
	}
	return false
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func BuildLearnedKnowledge(
	service string,
	evidence any,
	explanation string,
) KnowledgeArticle {
	raw := mustJSON(evidence)

	var decoded any
	_ = json.Unmarshal(raw, &decoded)

	dependency := findStringByKeys(decoded,
		"dependency", "dependency_name")

	endpoint := findStringByKeys(decoded,
		"endpoint", "path", "route")

	failureType := findStringByKeys(decoded,
		"failure_type", "event_type", "type")

	if failureType == "" {
    failureType = findErrorText(decoded)
}

if dependency == "" {
    dependency = "unknown"
}
if endpoint == "" {
    endpoint = "unknown"
}
if failureType == "" {
    // Record the observed incident state rather than using "unknown"
    // when the evidence confirms a degraded dependency.
    status := evidenceStatus(evidence)

    if status == "degraded" {
        failureType = "degraded"
    } else {
        failureType = "unknown"
    }
}

	signature := BuildErrorSignature(
		service,
		dependency,
		failureType,
		endpoint,
	)

	return KnowledgeArticle{
		Title:          "Learned Incident: " + service + " - " + dependency,
		Summary:        explanation,
		MatchedOn:      []string{signature},
		ErrorSignature: signature,
		Source:         "AI-generated",
		Verified:       false,
		Steps: []string{
			"Review the AI-generated diagnosis against the collected logs and metrics.",
			"Validate the suggested remediation before applying it.",
			"Re-run the investigation to confirm whether the issue is resolved.",
		},
	}
}

func BuildErrorSignature(
	service string,
	dependency string,
	failureType string,
	endpoint string,
) string {
	return "service=" + normalize(service) +
		"|dependency=" + normalize(dependency) +
		"|failure=" + normalize(failureType) +
		"|endpoint=" + normalize(endpoint)
}

func findStringByKeys(value any, keys ...string) string {
	wanted := make(map[string]bool, len(keys))
	for _, key := range keys {
		wanted[normalize(key)] = true
	}

	switch item := value.(type) {
	case map[string]any:
		// Prefer a field with the requested name.
		for key, val := range item {
			if wanted[normalize(key)] {
				if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		for _, child := range item {
			if result := findStringByKeys(child, keys...); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range item {
			if result := findStringByKeys(child, keys...); result != "" {
				return result
			}
		}
	}

	return ""
}

func findErrorText(value any) string {
	switch item := value.(type) {
	case map[string]any:
		for _, key := range []string{"error", "message", "msg"} {
			if s, ok := item[key].(string); ok &&
				containsAny(strings.ToLower(s),
					"error", "fail", "refused", "timeout") {
				return strings.TrimSpace(s)
			}
		}
		for _, child := range item {
			if result := findErrorText(child); result != "" {
				return result
			}
		}
	case []any:
		for _, child := range item {
			if result := findErrorText(child); result != "" {
				return result
			}
		}
	}

	return ""
}
