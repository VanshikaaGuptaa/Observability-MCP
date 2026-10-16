
package main

import (
	"encoding/json"
	"os"
	"strings"
)

const knowledgeFile = "knowledge.json"

// LoadSavedKnowledge loads articles persisted from previous investigations.
func LoadSavedKnowledge() ([]KnowledgeArticle, error) {
	data, err := os.ReadFile(knowledgeFile)
	if os.IsNotExist(err) {
		return []KnowledgeArticle{}, nil
	}
	if err != nil {
		return nil, err
	}

	var articles []KnowledgeArticle
	if err := json.Unmarshal(data, &articles); err != nil {
		return nil, err
	}
	return articles, nil
}

// SaveKnowledgeArticle persists an article unless an equivalent one exists.
func SaveKnowledgeArticle(article KnowledgeArticle) error {
	articles, err := LoadSavedKnowledge()
	if err != nil {
		return err
	}

	signature := strings.TrimSpace(strings.ToLower(article.ErrorSignature))

	for _, existing := range articles {
		if signature != "" &&
			strings.EqualFold(strings.TrimSpace(existing.ErrorSignature), signature) {
			return nil // Already saved this error.
		}

		if strings.EqualFold(
			strings.TrimSpace(existing.Title),
			strings.TrimSpace(article.Title),
		) {
			return nil // Avoid duplicate titles too.
		}
	}

	articles = append(articles, article)

	data, err := json.MarshalIndent(articles, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(knowledgeFile, data, 0644)
}

// FindSavedKnowledge returns saved articles whose error signature
// appears in the current incident evidence.
func FindSavedKnowledge(evidence any) []KnowledgeArticle {
	articles, err := LoadSavedKnowledge()
	if err != nil {
		return []KnowledgeArticle{}
	}

	raw, err := json.Marshal(evidence)
	if err != nil {
		return []KnowledgeArticle{}
	}

	evidenceText := strings.ToLower(string(raw))
	var matches []KnowledgeArticle

	for _, article := range articles {
		signature := strings.ToLower(strings.TrimSpace(article.ErrorSignature))
		if signature != "" && strings.Contains(evidenceText, signature) {
			matches = append(matches, article)
		}
	}

	return matches
}

func BuildLearnedKnowledge(service string, evidence any, explanation string) KnowledgeArticle {
	raw, _ := json.Marshal(evidence)

	var decoded any
	_ = json.Unmarshal(raw, &decoded)

	signature := findErrorText(decoded)
	if signature == "" {
		signature = strings.TrimSpace(service + " incident")
	}

	title := "Learned Incident: " + signature
	if len(title) > 100 {
		title = title[:100]
	}

	return KnowledgeArticle{
		Title:         title,
		Summary:       explanation,
		MatchedOn:     []string{signature},
		ErrorSignature: signature,
		Source:        "AI-generated",
		Verified:      false,
		Steps: []string{
			"Review the AI-generated diagnosis against the collected logs and metrics.",
			"Validate the suggested remediation before applying it.",
			"Re-run the investigation to confirm whether the issue is resolved.",
		},
	}
}

// findErrorText searches nested evidence for an error or message string.
func findErrorText(value any) string {
	switch item := value.(type) {
	case map[string]any:
		for _, key := range []string{"error", "message", "msg"} {
			if text, ok := item[key].(string); ok &&
				text != "" &&
				(strings.Contains(strings.ToLower(text), "error") ||
					strings.Contains(strings.ToLower(text), "fail") ||
					strings.Contains(strings.ToLower(text), "refused") ||
					strings.Contains(strings.ToLower(text), "timeout")) {
				return text
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
