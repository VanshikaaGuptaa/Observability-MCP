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
	Service           string             `json:"service"`
	Duration          string             `json:"duration"`
	Evidence          any                `json:"evidence"`
	AIExplanation     string             `json:"ai_explanation"`
	DurationMS        int64              `json:"duration_ms"`
	Knowledge         []KnowledgeArticle `json:"knowledge"`
	InvestigationMode string             `json:"investigation_mode"`
	AISkipped         bool               `json:"ai_skipped"`
	MatchScore        float64            `json:"match_score"`
	DecisionReason    string             `json:"decision_reason"`
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
	if err := json.Unmarshal(raw, &result); err != nil {
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

	// Reject an additional JSON value after the request object.
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		http.Error(w, "request must contain a single JSON object", http.StatusBadRequest)
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

	// Step 1: Always collect fresh evidence from MCP.
	// Knowledge matching must never bypass evidence collection.
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

	// Step 2: Gather every applicable knowledge source before deciding
	// whether Gemini is needed.
	// Collect articles for display.
	known := FindKnowledge(evidence)
	saved := FindSavedKnowledge(evidence)

	// Load ALL saved articles for router evaluation.
	// Do not rely on FindSavedKnowledge's separate matching rules.
	allSaved, loadErr := LoadSavedKnowledge()
	if loadErr != nil {
		log.Printf("Failed to load saved knowledge for routing: %v", loadErr)
		allSaved = []KnowledgeArticle{}
	}

	knowledge := make([]KnowledgeArticle, 0, len(known)+len(saved)+1)

	for _, article := range known {
		article.MatchOrigin = "static"
		appendUniqueKnowledge(&knowledge, article)
	}

	for _, article := range saved {
		article.MatchOrigin = "saved"
		appendUniqueKnowledge(&knowledge, article)
	}

	// Build a separate candidate list for the router.
	routingCandidates := make([]KnowledgeArticle, 0, len(known)+len(allSaved))

	for _, article := range known {
		article.MatchOrigin = "static"
		routingCandidates = append(routingCandidates, article)
	}

	for _, article := range allSaved {
		article.MatchOrigin = "saved"
		routingCandidates = append(routingCandidates, article)
	}

	decision := RouteKnowledge(
		req.Service,
		evidence,
		routingCandidates,
	)

	log.Printf(
		"Knowledge routing: evidence_status=%q static_articles=%d saved_articles=%d "+
			"candidates=%d verified_candidates=%d mode=%s score=%.2f reason=%s",
		evidenceStatus(evidence),
		len(known),
		len(allSaved),
		len(routingCandidates),
		countVerified(routingCandidates),
		decision.Mode,
		decision.MatchScore,
		decision.DecisionReason,
	)

	knowledge = make([]KnowledgeArticle, 0, len(known)+len(saved)+1)

	for _, article := range known {
		article.MatchOrigin = "static"
		appendUniqueKnowledge(&knowledge, article)
	}

	for _, article := range saved {
		article.MatchOrigin = "saved"
		appendUniqueKnowledge(&knowledge, article)
	}

	// Step 3: Route the investigation using verified knowledge only.
	// RouteKnowledge must exclude unverified learned articles from
	// eligibility to skip Gemini.
	decision = RouteKnowledge(
		req.Service,
		evidence,
		knowledge,
	)

	explanation := ""
	aiSucceeded := false

	// Step 4: Either use a deterministic runbook-based explanation
	// or ask Gemini to analyze the collected evidence.
	if decision.SkipAI && decision.MatchedArticle != nil {
		explanation = deterministicKnowledgeExplanation(
			req.Service,
			evidence,
			*decision.MatchedArticle,
			decision.MatchScore,
		)

		log.Printf(
			"Skipping Gemini: service=%s article=%q score=%.2f reason=%q",
			req.Service,
			decision.MatchedArticle.Title,
			decision.MatchScore,
			decision.DecisionReason,
		)
	} else {
		explanation, err = a.gemini.Analyze(
			ctx,
			req.Service,
			req.Duration,
			evidence,
		)

		aiSucceeded = err == nil

		if err != nil {
			log.Printf("Gemini analysis failed: %v", err)
			explanation = "AI explanation unavailable. Review the collected metrics and logs below for incident evidence."
		}

		log.Printf(
			"Gemini investigation: service=%s reason=%q",
			req.Service,
			decision.DecisionReason,
		)
	}

	// Step 5: Learn from an incident only when Gemini actually ran
	// successfully. A deterministic runbook explanation is not new
	// AI-generated knowledge and must not be learned as such.
	status := evidenceStatus(evidence)
	incidentUnhealthy := status != "" &&
		status != "healthy" &&
		status != "normal"

	if !decision.SkipAI &&
		incidentUnhealthy &&
		aiSucceeded &&
		len(saved) == 0 {

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

			log.Printf(
				"Learned and saved new article: %s",
				article.Title,
			)
		} else {
			// An equivalent article already exists. Reload matches so
			// the response can include the persisted version.
			log.Printf(
				"Matching saved article already exists: %s",
				article.Title,
			)

			refreshed := FindSavedKnowledge(evidence)
			for _, savedArticle := range refreshed {
				savedArticle.MatchOrigin = "saved"
				appendUniqueKnowledge(&knowledge, savedArticle)
			}
		}
	}

	// Step 6: Return both the investigation result and routing decision.
	w.Header().Set("Content-Type", "application/json")

	response := InvestigationResponse{
		Service:           req.Service,
		Duration:          req.Duration,
		Evidence:          evidence,
		AIExplanation:     explanation,
		DurationMS:        time.Since(start).Milliseconds(),
		Knowledge:         knowledge,
		InvestigationMode: decision.Mode,
		AISkipped:         decision.SkipAI,
		MatchScore:        decision.MatchScore,
		DecisionReason:    decision.DecisionReason,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode investigation response: %v", err)
	}
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
			// Avoid duplicate articles with the same signature.
			// Prefer a verified article over an unverified duplicate.
			if candidate.Verified && !existing.Verified {
				(*articles)[i] = candidate
			}
			return
		}

		if strings.EqualFold(
			strings.TrimSpace(existing.Title),
			strings.TrimSpace(candidate.Title),
		) {
			// Avoid duplicate titles. Prefer saved or learned content
			// when it provides a more specific match than a generic
			// static article.
			if (candidate.MatchOrigin == "saved" ||
				candidate.MatchOrigin == "learned") &&
				(!existing.Verified || candidate.Verified) {
				(*articles)[i] = candidate
			}
			return
		}
	}

	*articles = append(*articles, candidate)
}
func countVerified(articles []KnowledgeArticle) int {
	count := 0
	for _, article := range articles {
		if article.Verified {
			count++
		}
	}
	return count
}
