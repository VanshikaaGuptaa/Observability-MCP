package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type InvestigationRequest struct {
	Service  string `json:"service"`
	Duration string `json:"duration"`
}

type InvestigationResponse struct {
	Service       string             `json:"service"`
	Duration      string             `json:"duration"`
	Evidence      any                `json:"evidence"`
	AIExplanation string             `json:"ai_explanation"`
	DurationMS    int64              `json:"duration_ms"`
	Knowledge     []KnowledgeArticle `json:"knowledge"`
}

type API struct {
	mcp    *MCPClient
	gemini *GeminiAnalyzer
}

func evidenceStatus(evidence any) string {
	raw, err := json.Marshal(evidence)
	if err != nil {
		return ""
	}

	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return ""
	}

	if status, ok := result["status"].(string); ok {
		return strings.ToLower(strings.TrimSpace(status))
	}

	if nested, ok := result["evidence"].(map[string]any); ok {
		if status, ok := nested["status"].(string); ok {
			return strings.ToLower(strings.TrimSpace(status))
		}
	}

	return ""
}

func (a *API) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (a *API) Investigate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	defer r.Body.Close()

	var req InvestigationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}

	if req.Service == "" {
		req.Service = "orderflow"
	}
	if req.Service != "orderflow" {
		http.Error(w, "unsupported service", http.StatusBadRequest)
		return
	}

	if req.Duration == "" {
		req.Duration = "10m"
	}

	switch req.Duration {
	case "5m", "10m", "15m", "1h":
	default:
		http.Error(w, "unsupported duration", http.StatusBadRequest)
		return
	}

	start := time.Now()
	ctx := r.Context()

	evidence, err := a.mcp.InvestigateIncident(
		ctx,
		req.Service,
		req.Duration,
	)
	if err != nil {
		log.Printf("MCP investigation failed: %v", err)
		http.Error(w, "MCP investigation failed", http.StatusBadGateway)
		return
	}

	explanation, err := a.gemini.Analyze(
		ctx,
		req.Service,
		req.Duration,
		evidence,
	)
	aiSucceeded := err == nil

	if err != nil {
		log.Printf("Gemini analysis failed: %v", err)
		explanation = "AI explanation unavailable. Review the collected metrics and logs below for incident evidence."
	}

	// Always gather all applicable knowledge sources.
	known := FindKnowledge(evidence)
	saved := FindSavedKnowledge(evidence)

	knowledge := make([]KnowledgeArticle, 0, len(known)+len(saved)+1)

	for _, article := range known {
		article.MatchOrigin = "static"
		appendUniqueKnowledge(&knowledge, article)
	}

	for _, article := range saved {
		article.MatchOrigin = "saved"
		appendUniqueKnowledge(&knowledge, article)
	}

	status := evidenceStatus(evidence)
	incidentUnhealthy := status != "" &&
		status != "healthy" &&
		status != "normal"

	// Learn only if AI succeeded, evidence indicates an incident, and
	// no equivalent saved article already matched. Static generic
	// runbooks do not block learning.
	if incidentUnhealthy && aiSucceeded && len(saved) == 0 {
		article := BuildLearnedKnowledge(
			req.Service,
			evidence,
			explanation,
		)

		inserted, saveErr := SaveKnowledgeArticle(article)
		if saveErr != nil {
			log.Printf("Failed to save learned article: %v", saveErr)
		} else if inserted {
			article.MatchOrigin = "learned"
			appendUniqueKnowledge(&knowledge, article)
			log.Printf("Learned and saved new article: %s", article.Title)
		} else {
			// It already exists. Fetch the stored copy and label it saved.
			log.Printf("Matching saved article already exists: %s", article.Title)

			refreshed := FindSavedKnowledge(evidence)
			for _, savedArticle := range refreshed {
				savedArticle.MatchOrigin = "saved"
				appendUniqueKnowledge(&knowledge, savedArticle)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(InvestigationResponse{
		Service:       req.Service,
		Duration:      req.Duration,
		Evidence:      evidence,
		AIExplanation: explanation,
		DurationMS:    time.Since(start).Milliseconds(),
		Knowledge:     knowledge,
	})
}

func appendUniqueKnowledge(
	articles *[]KnowledgeArticle,
	candidate KnowledgeArticle,
) {
	for i, existing := range *articles {
		if candidate.ErrorSignature != "" &&
			strings.EqualFold(
				strings.TrimSpace(existing.ErrorSignature),
				strings.TrimSpace(candidate.ErrorSignature),
			) {
			// Preserve the origin of the more useful existing result.
			// Static and saved copies of the same article should not duplicate.
			return
		}

		if strings.EqualFold(
			strings.TrimSpace(existing.Title),
			strings.TrimSpace(candidate.Title),
		) {
			// Prefer the saved/learned version if the title collides
			// with a generic static runbook.
			if candidate.MatchOrigin == "saved" ||
				candidate.MatchOrigin == "learned" {
				(*articles)[i] = candidate
			}
			return
		}
	}

	*articles = append(*articles, candidate)
}
