package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

type updateVerificationRequest struct {
	Verified *bool `json:"verified"`
}

// HandleKnowledge handles GET /api/knowledge.
// Returns all static runbooks and learned articles in a consistent response.
func (a *API) HandleKnowledge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	staticArticles := GetAllStaticKnowledge()
	savedArticles, err := LoadSavedKnowledge()
	if err != nil {
		log.Printf("Failed to load saved knowledge for GET /api/knowledge: %v", err)
		savedArticles = []KnowledgeArticle{}
	}

	all := make([]KnowledgeArticle, 0, len(staticArticles)+len(savedArticles))
	for _, art := range staticArticles {
		art.MatchOrigin = "static"
		art.Verified = true
		all = append(all, art)
	}
	for _, art := range savedArticles {
		art.MatchOrigin = "learned"
		all = append(all, art)
	}

	writeJSON(w, http.StatusOK, all)
}

// HandleKnowledgeItem routes subpaths of /api/knowledge/, notably:
// PATCH /api/knowledge/{id}/verification
func (a *API) HandleKnowledgeItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/knowledge/")
	path = strings.Trim(path, "/")

	// Expected route: {id}/verification
	if !strings.HasSuffix(path, "/verification") {
		writeAPIError(w, http.StatusNotFound, "not found")
		return
	}

	id := strings.TrimSuffix(path, "/verification")
	id = strings.TrimSpace(strings.Trim(id, "/"))

	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "article ID is required")
		return
	}

	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Reject attempts to modify static runbooks immediately
	if IsStaticArticleID(id) {
		writeAPIError(w, http.StatusBadRequest, "static runbooks are always verified and cannot be modified")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req updateVerificationRequest
	if err := decoder.Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid JSON: verified boolean is required")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, "request must contain a single JSON object")
		return
	}

	if req.Verified == nil {
		writeAPIError(w, http.StatusBadRequest, "field 'verified' is required and must be a boolean")
		return
	}

	updated, err := UpdateLearnedVerification(id, *req.Verified)
	if err != nil {
		switch {
		case errors.Is(err, ErrStaticArticleCannotBeModified):
			writeAPIError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrArticleNotFound):
			writeAPIError(w, http.StatusNotFound, "article not found")
		default:
			log.Printf("Failed to update verification for %q: %v", id, err)
			writeAPIError(w, http.StatusInternalServerError, "failed to update verification status")
		}
		return
	}

	log.Printf("Knowledge verification updated: id=%q verified=%v", updated.ID, updated.Verified)
	writeJSON(w, http.StatusOK, updated)
}
