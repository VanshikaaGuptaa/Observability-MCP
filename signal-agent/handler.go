package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"fmt"
	"strings"
)

type InvestigationRequest struct {
	Service  string `json:"service"`
	Duration string `json:"duration"`
}

type InvestigationResponse struct {
	Service       string `json:"service"`
	Duration      string `json:"duration"`
	Evidence      any    `json:"evidence"`
	AIExplanation string `json:"ai_explanation"`
	DurationMS    int64  `json:"duration_ms"`
	Knowledge []KnowledgeArticle `json:"knowledge"`
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
        return status
    }

    // Your MCP response may wrap the incident under "evidence".
    if nested, ok := result["evidence"].(map[string]any); ok {
        if status, ok := nested["status"].(string); ok {
            return status
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

	// Phase 1 supports the existing Orderflow service only.
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
	if req.Duration != "5m" &&
		req.Duration != "10m" &&
		req.Duration != "15m" &&
		req.Duration != "1h" {
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
		http.Error(w, "MCP investigation failed", http.StatusBadGateway)
		return
	}

	
explanation, err := a.gemini.Analyze(
	ctx, req.Service, req.Duration, evidence,
)
aiSucceeded := err == nil

if err != nil {
	log.Printf("Gemini analysis failed: %v", err)
	explanation = "AI explanation unavailable. Review the collected metrics and logs below for incident evidence."
}

known := FindKnowledge(evidence)
saved := FindSavedKnowledge(evidence)

knowledge := append(known, saved...)

// Learn only when no existing article matches and AI analysis succeeded.
status := strings.ToLower(
    fmt.Sprint(evidenceStatus(evidence)),
)

if len(known) == 0 &&
    len(saved) == 0 &&
    aiSucceeded &&
    status != "healthy" &&
    status != "normal" {
	article := BuildLearnedKnowledge(
		req.Service,
		evidence,
		explanation,
	)

	if err := SaveKnowledgeArticle(article); err != nil {
		log.Printf("Failed to save knowledge article: %v", err)
	} else {
		knowledge = append(knowledge, article)
		log.Printf("Saved new knowledge article: %s", article.Title)
	}
}


	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(InvestigationResponse{
		Service:       req.Service,
		Duration:      req.Duration,
		Evidence:      evidence,
		AIExplanation: explanation,
		DurationMS:    time.Since(start).Milliseconds(),
		Knowledge: knowledge,
	})
}
