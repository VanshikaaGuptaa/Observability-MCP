package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func setupTestKnowledgeDir(t *testing.T, initial []KnowledgeArticle) string {
	t.Helper()
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	if initial != nil {
		data, err := json.MarshalIndent(initial, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, knowledgeFile), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestListKnowledgeHubArticles(t *testing.T) {
	setupTestKnowledgeDir(t, []KnowledgeArticle{
		{
			ID:             "learned-test-1",
			Title:          "Learned Test 1",
			Summary:        "Summary 1",
			ErrorSignature: "service=orderflow|dependency=postgresql|failure=degraded|endpoint=/api/orders",
			Verified:       false,
			Source:         "AI-generated",
		},
	})

	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/knowledge", nil)
	rec := httptest.NewRecorder()
	api.HandleKnowledge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var articles []KnowledgeArticle
	if err := json.Unmarshal(rec.Body.Bytes(), &articles); err != nil {
		t.Fatalf("failed to decode articles: %v", err)
	}

	// 5 static runbooks + 1 learned article = 6 total
	if len(articles) != 6 {
		t.Fatalf("expected 6 articles (5 static + 1 learned), got %d", len(articles))
	}

	staticCount := 0
	learnedCount := 0
	for _, a := range articles {
		if a.ID == "" {
			t.Errorf("article %q missing ID", a.Title)
		}
		if a.MatchOrigin == "static" {
			staticCount++
			if !a.Verified {
				t.Errorf("static article %q must be verified", a.Title)
			}
		} else if a.MatchOrigin == "learned" {
			learnedCount++
			if a.ID != "learned-test-1" {
				t.Errorf("unexpected learned ID %q", a.ID)
			}
			if a.Verified {
				t.Errorf("learned article was set to unverified, got verified")
			}
		}
	}

	if staticCount != 5 {
		t.Errorf("expected 5 static articles, got %d", staticCount)
	}
	if learnedCount != 1 {
		t.Errorf("expected 1 learned article, got %d", learnedCount)
	}
}

func TestToggleLearnedVerificationLifecycle(t *testing.T) {
	setupTestKnowledgeDir(t, []KnowledgeArticle{
		{
			ID:             "learned-postgres-1",
			Title:          "Learned Postgres",
			Summary:        "Postgres error",
			ErrorSignature: "service=orderflow|dependency=postgresql|failure=degraded|endpoint=/api/orders",
			Verified:       false,
			Source:         "AI-generated",
		},
	})

	api := &API{}

	// 1. Verify the article (false -> true)
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/knowledge/learned-postgres-1/verification",
		strings.NewReader(`{"verified": true}`))
	patchReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.HandleKnowledgeItem(rec, patchReq)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var updated KnowledgeArticle
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to parse updated article: %v", err)
	}
	if !updated.Verified {
		t.Fatalf("expected verified to be true, got false")
	}

	// Confirm persistence on disk
	saved, err := LoadSavedKnowledge()
	if err != nil || len(saved) != 1 {
		t.Fatalf("expected 1 saved article on disk, got %d (%v)", len(saved), err)
	}
	if !saved[0].Verified {
		t.Fatalf("expected persisted article to have verified=true")
	}

	// 2. Unverify the article (true -> false)
	patchReq2 := httptest.NewRequest(http.MethodPatch, "/api/knowledge/learned-postgres-1/verification",
		strings.NewReader(`{"verified": false}`))
	patchReq2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	api.HandleKnowledgeItem(rec2, patchReq2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var updated2 KnowledgeArticle
	_ = json.Unmarshal(rec2.Body.Bytes(), &updated2)
	if updated2.Verified {
		t.Fatalf("expected verified to be false, got true")
	}

	saved2, _ := LoadSavedKnowledge()
	if saved2[0].Verified {
		t.Fatalf("expected persisted article to have verified=false after toggle")
	}
}

func TestRejectStaticArticleVerificationChange(t *testing.T) {
	setupTestKnowledgeDir(t, nil)
	api := &API{}

	// Attempt to change a static runbook
	req := httptest.NewRequest(http.MethodPatch, "/api/knowledge/static-postgresql-failure/verification",
		strings.NewReader(`{"verified": false}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.HandleKnowledgeItem(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "static runbooks are always verified") {
		t.Fatalf("unexpected error message: %s", rec.Body.String())
	}
}

func TestRejectUnknownArticleID(t *testing.T) {
	setupTestKnowledgeDir(t, []KnowledgeArticle{})
	api := &API{}

	req := httptest.NewRequest(http.MethodPatch, "/api/knowledge/non-existent-id/verification",
		strings.NewReader(`{"verified": true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.HandleKnowledgeItem(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRejectInvalidVerificationPayloads(t *testing.T) {
	setupTestKnowledgeDir(t, []KnowledgeArticle{
		{ID: "learned-1", Title: "Learned 1", Verified: false},
	})
	api := &API{}

	cases := []struct {
		name string
		body string
		code int
	}{
		{"empty json", `{}`, http.StatusBadRequest},
		{"malformed json", `{"verified":`, http.StatusBadRequest},
		{"string instead of bool", `{"verified": "yes"}`, http.StatusBadRequest},
		{"number instead of bool", `{"verified": 1}`, http.StatusBadRequest},
		{"extra fields", `{"verified": true, "extra": 123}`, http.StatusBadRequest},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/api/knowledge/learned-1/verification",
				strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			api.HandleKnowledgeItem(rec, req)

			if rec.Code != c.code {
				t.Errorf("expected status %d, got %d: %s", c.code, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestConcurrentVerificationUpdates(t *testing.T) {
	setupTestKnowledgeDir(t, []KnowledgeArticle{
		{ID: "learned-a", Title: "Article A", Verified: false},
		{ID: "learned-b", Title: "Article B", Verified: true},
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			_, err := UpdateLearnedVerification("learned-a", idx%2 == 0)
			if err != nil {
				t.Errorf("concurrent update failed: %v", err)
			}
		}(i)
		go func(idx int) {
			defer wg.Done()
			_, err := UpdateLearnedVerification("learned-b", idx%2 != 0)
			if err != nil {
				t.Errorf("concurrent update failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	saved, err := LoadSavedKnowledge()
	if err != nil {
		t.Fatalf("file corrupted after concurrent writes: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 saved articles, got %d", len(saved))
	}
}

func TestVerificationAffectsInvestigationRouter(t *testing.T) {
	sig := "service=orderflow|dependency=postgresql|failure=degraded|endpoint=/api/orders"
	article := KnowledgeArticle{
		ID:             "learned-orderflow-postgres",
		Title:          "Learned Incident: orderflow - postgresql",
		Summary:        "Database failure diagnosis",
		MatchedOn:      []string{sig},
		ErrorSignature: sig,
		Source:         "AI-generated",
		Verified:       false, // Initially unverified!
	}

	setupTestKnowledgeDir(t, []KnowledgeArticle{article})

	mcp := &fakeMCP{evidence: postgresEvidence()}
	ai := &fakeAnalyzer{text: "AI Gemini Response"}
	api := &API{mcp: mcp, gemini: ai}

	// Case 1: Article is UNVERIFIED -> Must NOT skip Gemini!
	resp1 := runInvestigation(t, api)
	if resp1.AISkipped {
		t.Fatalf("unverified article must NOT skip Gemini")
	}
	if ai.calls.Load() != 1 {
		t.Fatalf("expected Gemini to be called once, got %d", ai.calls.Load())
	}

	// Case 2: User verifies the article via UpdateLearnedVerification
	_, err := UpdateLearnedVerification("learned-orderflow-postgres", true)
	if err != nil {
		t.Fatalf("failed to verify article: %v", err)
	}

	// Case 3: Re-investigate -> The router must reload knowledge.json, see verified=true, and SKIP Gemini!
	resp2 := runInvestigation(t, api)
	if !resp2.AISkipped {
		t.Fatalf("verified matching article MUST skip Gemini")
	}
	if resp2.InvestigationMode != "knowledge" {
		t.Fatalf("expected knowledge investigation mode, got %s", resp2.InvestigationMode)
	}
	// ai.calls should STILL be 1 (Gemini was NOT called for investigation 2)
	if ai.calls.Load() != 1 {
		t.Fatalf("Gemini should have been skipped, but call count increased to %d", ai.calls.Load())
	}

	// Case 4: User UNVERIFIES the article again
	_, err = UpdateLearnedVerification("learned-orderflow-postgres", false)
	if err != nil {
		t.Fatalf("failed to unverify article: %v", err)
	}

	// Case 5: Re-investigate -> Router sees verified=false and calls Gemini again!
	resp3 := runInvestigation(t, api)
	if resp3.AISkipped {
		t.Fatalf("unverified article after toggle must NOT skip Gemini")
	}
	if ai.calls.Load() != 2 {
		t.Fatalf("expected Gemini to be called again (call count 2), got %d", ai.calls.Load())
	}
}

func TestNewlyLearnedArticleDefaultsToUnverified(t *testing.T) {
	setupTestKnowledgeDir(t, nil)

	art := BuildLearnedKnowledge("orderflow", postgresEvidence(), "Generated analysis")
	if art.Verified {
		t.Fatalf("newly built learned article must default to verified: false")
	}

	inserted, err := SaveKnowledgeArticle(art)
	if err != nil || !inserted {
		t.Fatalf("failed to save article: %v", err)
	}

	saved, err := LoadSavedKnowledge()
	if err != nil || len(saved) != 1 {
		t.Fatalf("expected 1 saved article, got %d (%v)", len(saved), err)
	}
	if saved[0].Verified {
		t.Fatalf("persisted learned article must have verified: false")
	}
}
