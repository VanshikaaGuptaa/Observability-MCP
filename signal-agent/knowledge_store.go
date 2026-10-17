package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const knowledgeFile = "knowledge.json"

var (
	knowledgeFileMu sync.Mutex

	ErrStaticArticleCannotBeModified = errors.New("static runbooks are always verified and cannot be modified")
	ErrArticleNotFound              = errors.New("knowledge article not found")
)

// normalizeSlug turns strings into URL-safe slugs.
func normalizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == '-' || r == '_' || r == ' ' || r == '/' {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteRune('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// ensureArticleID guarantees every article has a deterministic stable ID.
func ensureArticleID(a *KnowledgeArticle) string {
	if strings.TrimSpace(a.ID) != "" {
		return strings.TrimSpace(a.ID)
	}
	fields := parseSignature(a.ErrorSignature)
	srv := normalizeSlug(fields["service"])
	dep := normalizeSlug(fields["dependency"])
	if srv != "" && dep != "" {
		a.ID = fmt.Sprintf("learned-%s-%s", srv, dep)
		return a.ID
	}
	h := sha256.Sum256([]byte(strings.TrimSpace(a.ErrorSignature) + "\n" + strings.TrimSpace(a.Title)))
	a.ID = fmt.Sprintf("learned-%x", h[:6])
	return a.ID
}

// atomicWriteFile safely writes data to a temporary file in the target directory
// and renames it atomically to prevent corruption or partial writes.
func atomicWriteFile(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	tmpFile, err := os.CreateTemp(dir, "knowledge-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		// Fallback for Windows where destination already exists, or Linux bind-mounted files
		_ = os.Remove(filename)
		if err2 := os.Rename(tmpName, filename); err2 != nil {
			if writeErr := os.WriteFile(filename, data, 0644); writeErr != nil {
				return fmt.Errorf("rename temp file: %w (fallback write: %v)", err, writeErr)
			}
		}
	}
	return nil
}

// LoadSavedKnowledge reads all persisted articles from knowledge.json.
// Concurrency-safe; ensures all returned articles have stable IDs and origin "learned".
func LoadSavedKnowledge() ([]KnowledgeArticle, error) {
	knowledgeFileMu.Lock()
	defer knowledgeFileMu.Unlock()

	data, err := os.ReadFile(knowledgeFile)
	if os.IsNotExist(err) {
		return []KnowledgeArticle{}, nil
	}
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return []KnowledgeArticle{}, nil
	}

	var articles []KnowledgeArticle
	if err := json.Unmarshal(data, &articles); err != nil {
		return nil, err
	}

	for i := range articles {
		ensureArticleID(&articles[i])
		if articles[i].MatchOrigin == "" {
			articles[i].MatchOrigin = "learned"
		}
	}

	return articles, nil
}

// UpdateLearnedVerification toggles the verification status of a learned article.
// Rejects static runbooks. Preserves all other article fields and unknown metadata.
func UpdateLearnedVerification(id string, verified bool) (*KnowledgeArticle, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrArticleNotFound
	}
	if IsStaticArticleID(id) {
		return nil, ErrStaticArticleCannotBeModified
	}

	knowledgeFileMu.Lock()
	defer knowledgeFileMu.Unlock()

	data, err := os.ReadFile(knowledgeFile)
	if os.IsNotExist(err) {
		return nil, ErrArticleNotFound
	}
	if err != nil {
		return nil, err
	}

	var rawItems []map[string]any
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return nil, err
	}

	var articles []KnowledgeArticle
	if err := json.Unmarshal(data, &articles); err != nil {
		return nil, err
	}

	targetIndex := -1
	for i := range articles {
		artID := ensureArticleID(&articles[i])
		rawID, _ := rawItems[i]["id"].(string)
		if artID == id || rawID == id {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return nil, ErrArticleNotFound
	}

	// Preserve unknown metadata by updating the raw map
	rawItems[targetIndex]["id"] = id
	rawItems[targetIndex]["verified"] = verified

	articles[targetIndex].ID = id
	articles[targetIndex].Verified = verified
	articles[targetIndex].MatchOrigin = "learned"

	updatedData, err := json.MarshalIndent(rawItems, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := atomicWriteFile(knowledgeFile, updatedData); err != nil {
		return nil, err
	}

	res := articles[targetIndex]
	return &res, nil
}

// SaveKnowledgeArticle returns true only when a new article was inserted.
// Newly learned articles default to verified: false and never overwrite user verification.
func SaveKnowledgeArticle(article KnowledgeArticle) (bool, error) {
	knowledgeFileMu.Lock()
	defer knowledgeFileMu.Unlock()

	data, err := os.ReadFile(knowledgeFile)
	var rawItems []map[string]any
	var articles []KnowledgeArticle

	if err == nil && len(strings.TrimSpace(string(data))) > 0 {
		_ = json.Unmarshal(data, &rawItems)
		_ = json.Unmarshal(data, &articles)
	}

	for _, existing := range articles {
		if sameArticle(existing, article) {
			return false, nil
		}
	}

	ensureArticleID(&article)
	article.MatchOrigin = ""
	article.Verified = false // AI-learned knowledge always starts unverified

	rawBytes, err := json.Marshal(article)
	if err != nil {
		return false, err
	}
	var rawMap map[string]any
	if err := json.Unmarshal(rawBytes, &rawMap); err != nil {
		return false, err
	}
	delete(rawMap, "match_origin")

	rawItems = append(rawItems, rawMap)

	out, err := json.MarshalIndent(rawItems, "", "  ")
	if err != nil {
		return false, err
	}

	if err := atomicWriteFile(knowledgeFile, out); err != nil {
		return false, err
	}

	return true, nil
}

func sameArticle(a, b KnowledgeArticle) bool {
	if a.ID != "" && b.ID != "" && a.ID == b.ID {
		return true
	}

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

	article := KnowledgeArticle{
		Title:          "Learned Incident: " + service + " - " + dependency,
		Summary:        explanation,
		MatchedOn:      []string{signature},
		ErrorSignature: signature,
		Source:         "AI-generated",
		Verified:       false,
		MatchOrigin:    "learned",
		Steps: []string{
			"Review the AI-generated diagnosis against the collected logs and metrics.",
			"Validate the suggested remediation before applying it.",
			"Re-run the investigation to confirm whether the issue is resolved.",
		},
	}
	ensureArticleID(&article)
	return article
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
