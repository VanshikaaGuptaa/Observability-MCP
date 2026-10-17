package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// ---------- Settings API tests ----------

type fakeTester struct {
	err error
}

func (f *fakeTester) TestConnection(ctx context.Context) (string, error) {
	return "", f.err
}

func newSettingsAPI(envKey string, testErr error) (*API, *GeminiKeyStore) {
	store := NewGeminiKeyStore(envKey)
	return &API{settings: store, tester: &fakeTester{err: testErr}}, store
}

func doSettings(
	t *testing.T,
	handler http.HandlerFunc,
	method, path, body string,
	jsonCT bool,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:50000"
	if jsonCT {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestSettingsSaveGetDeleteLifecycle(t *testing.T) {
	const secret = "AIzaUSER-secret-value"
	api, store := newSettingsAPI("env-secret", nil)

	rec := doSettings(t, api.GeminiSettings, http.MethodPut, "/api/settings/gemini",
		`{"api_key":"`+secret+`"}`, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("save failed: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), secret) || strings.Contains(rec.Body.String(), "env-secret") {
		t.Fatal("PUT response leaked a key")
	}
	if key, src := store.Active(); key != secret || src != GeminiKeySourceUser {
		t.Fatalf("runtime config not updated: %s", src)
	}

	rec = doSettings(t, api.GeminiSettings, http.MethodGet, "/api/settings/gemini", "", false)
	var status GeminiKeyStatus
	_ = json.Unmarshal(rec.Body.Bytes(), &status)
	if !status.Configured || status.Source != GeminiKeySourceUser || !status.EnvFallback {
		t.Fatalf("unexpected status %+v", status)
	}
	if strings.Contains(rec.Body.String(), secret) || strings.Contains(rec.Body.String(), "env-secret") {
		t.Fatal("GET response leaked a key")
	}
	if strings.Contains(rec.Body.String(), "AIza") {
		t.Fatal("GET response contains key material")
	}

	rec = doSettings(t, api.GeminiSettings, http.MethodDelete, "/api/settings/gemini", "", true)
	_ = json.Unmarshal(rec.Body.Bytes(), &status)
	if rec.Code != http.StatusOK || status.Source != GeminiKeySourceEnv {
		t.Fatalf("delete should restore env fallback: %d %+v", rec.Code, status)
	}
}

func TestSettingsDeleteWithoutFallback(t *testing.T) {
	api, store := newSettingsAPI("", nil)
	_ = store.SetUserKey("k")

	rec := doSettings(t, api.GeminiSettings, http.MethodDelete, "/api/settings/gemini", "", true)
	var status GeminiKeyStatus
	_ = json.Unmarshal(rec.Body.Bytes(), &status)
	if status.Configured || status.Source != GeminiKeySourceNone {
		t.Fatalf("expected unconfigured, got %+v", status)
	}
}

func TestSettingsRejectsBadInput(t *testing.T) {
	api, store := newSettingsAPI("", nil)

	cases := []struct {
		name   string
		method string
		body   string
		json   bool
		want   int
	}{
		{"blank", http.MethodPut, `{"api_key":"   "}`, true, http.StatusBadRequest},
		{"missing", http.MethodPut, `{}`, true, http.StatusBadRequest},
		{"malformed", http.MethodPut, `{"api_key":`, true, http.StatusBadRequest},
		{"unknown field", http.MethodPut, `{"api_key":"k","x":1}`, true, http.StatusBadRequest},
		{"two objects", http.MethodPut, `{"api_key":"k"}{"api_key":"j"}`, true, http.StatusBadRequest},
		{"wrong content type", http.MethodPut, `{"api_key":"k"}`, false, http.StatusUnsupportedMediaType},
		{"bad method", http.MethodPost, `{}`, true, http.StatusMethodNotAllowed},
		{"too large", http.MethodPut, `{"api_key":"` + strings.Repeat("a", 8000) + `"}`, true, http.StatusBadRequest},
	}
	for _, c := range cases {
		rec := doSettings(t, api.GeminiSettings, c.method, "/api/settings/gemini", c.body, c.json)
		if rec.Code != c.want {
			t.Errorf("%s: expected %d, got %d (%s)", c.name, c.want, rec.Code, rec.Body.String())
		}
	}
	if store.Status().Configured {
		t.Fatal("rejected input must not configure a key")
	}
}

func TestSettingsRejectsNonLoopback(t *testing.T) {
	api, store := newSettingsAPI("", nil)
	req := httptest.NewRequest(http.MethodPut, "/api/settings/gemini", strings.NewReader(`{"api_key":"k"}`))
	req.RemoteAddr = "192.168.1.50:40000"
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	api.GeminiSettings(rec, req)

	if rec.Code != http.StatusForbidden || store.Status().Configured {
		t.Fatalf("remote request must be rejected, got %d", rec.Code)
	}
}

func TestSettingsConnectionTestEndpoint(t *testing.T) {
	const secret = "AIzaLEAK"

	// Success.
	api, _ := newSettingsAPI("", nil)
	rec := doSettings(t, api.TestGeminiSettings, http.MethodPost, "/api/settings/gemini/test", "", true)
	var resp geminiTestResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != http.StatusOK || !resp.OK {
		t.Fatalf("expected ok, got %d %s", rec.Code, rec.Body.String())
	}

	// Failure with a sanitized error.
	api, store := newSettingsAPI("", classifyGeminiError(errors.New("API key not valid "+secret)))
	_ = store.SetUserKey(secret)
	rec = doSettings(t, api.TestGeminiSettings, http.MethodPost, "/api/settings/gemini/test", "", true)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.OK || resp.Message == "" {
		t.Fatalf("expected failure message, got %+v", resp)
	}
	if strings.Contains(rec.Body.String(), secret) {
		t.Fatal("test endpoint leaked the key")
	}

	// Not configured.
	api, _ = newSettingsAPI("", ErrGeminiNotConfigured)
	rec = doSettings(t, api.TestGeminiSettings, http.MethodPost, "/api/settings/gemini/test", "", true)
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.OK || !strings.Contains(resp.Message, "not configured") {
		t.Fatalf("expected not configured message, got %+v", resp)
	}

	// Simple (non-JSON) POSTs are refused to block cross-site requests.
	rec = doSettings(t, api.TestGeminiSettings, http.MethodPost, "/api/settings/gemini/test", "", false)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rec.Code)
	}
}

// ---------- Investigation router regression tests ----------

type fakeMCP struct {
	evidence any
	calls    atomic.Int32
}

func (f *fakeMCP) InvestigateIncident(ctx context.Context, service, duration string) (any, error) {
	f.calls.Add(1)
	return f.evidence, nil
}

type fakeAnalyzer struct {
	calls atomic.Int32
	text  string
	err   error
}

func (f *fakeAnalyzer) Analyze(ctx context.Context, service, duration string, evidence any) (string, error) {
	f.calls.Add(1)
	return f.text, f.err
}

// postgresEvidence mirrors the shape of MCP evidence for a degraded
// PostgreSQL dependency on /api/orders.
func postgresEvidence() any {
	return map[string]any{
		"service":  "orderflow",
		"status":   "degraded",
		"severity": "medium",
		"evidence": []any{
			map[string]any{"signal": "postgresql", "source": "prometheus", "value": "degraded"},
		},
		"timeline": []any{
			map[string]any{
				"dependency": "postgresql",
				"endpoint":   "/api/orders",
				"event_type": "dependency_error",
				"message":    "postgresql dependency degraded",
			},
		},
	}
}

const postgresSignature = "service=orderflow|dependency=postgresql|failure=degraded|endpoint=/api/orders"

// withKnowledge runs the test in a temp dir containing knowledge.json.
func withKnowledge(t *testing.T, articles []KnowledgeArticle) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if articles != nil {
		data, _ := json.Marshal(articles)
		if err := os.WriteFile(filepath.Join(dir, knowledgeFile), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func runInvestigation(t *testing.T, api *API) InvestigationResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/investigations",
		strings.NewReader(`{"service":"orderflow","duration":"5m"}`))
	rec := httptest.NewRecorder()
	api.Investigate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("investigation failed: %d %s", rec.Code, rec.Body.String())
	}
	var resp InvestigationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return resp
}

func verifiedPostgresArticle() KnowledgeArticle {
	return KnowledgeArticle{
		Title:          "Learned Incident: orderflow - postgresql",
		Summary:        "Verified diagnosis.",
		MatchedOn:      []string{postgresSignature},
		Steps:          []string{"Check PostgreSQL."},
		ErrorSignature: postgresSignature,
		Source:         "AI-generated",
		Verified:       true,
	}
}

func TestRouterVerifiedMatchSkipsGeminiEvenWithKey(t *testing.T) {
	withKnowledge(t, []KnowledgeArticle{verifiedPostgresArticle()})

	mcp := &fakeMCP{evidence: postgresEvidence()}
	ai := &fakeAnalyzer{text: "AI analysis"}
	resp := runInvestigation(t, &API{mcp: mcp, gemini: ai})

	if mcp.calls.Load() != 1 {
		t.Fatal("MCP evidence must always be collected")
	}
	if ai.calls.Load() != 0 {
		t.Fatal("Gemini must not be called for a verified knowledge match")
	}
	if !resp.AISkipped || resp.InvestigationMode != "knowledge" || resp.MatchScore < KnowledgeSkipThreshold {
		t.Fatalf("unexpected routing: %+v", resp)
	}
	if resp.Evidence == nil || resp.AIError != "" {
		t.Fatal("live evidence must be returned and no AI error reported")
	}
}

func TestRouterVerifiedMatchWorksWithoutAnyKey(t *testing.T) {
	withKnowledge(t, []KnowledgeArticle{verifiedPostgresArticle()})

	// A real analyzer with no key configured.
	analyzer := NewGeminiAnalyzer(Config{GeminiModel: "m"}, NewGeminiKeyStore(""))
	resp := runInvestigation(t, &API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: analyzer})

	if !resp.AISkipped || resp.AIError != "" {
		t.Fatalf("verified knowledge must work without a key: %+v", resp)
	}
}

func TestRouterUnverifiedArticleCannotSkipGemini(t *testing.T) {
	article := verifiedPostgresArticle()
	article.Verified = false
	withKnowledge(t, []KnowledgeArticle{article})

	ai := &fakeAnalyzer{text: "AI analysis"}
	resp := runInvestigation(t, &API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: ai})

	if ai.calls.Load() != 1 || resp.AISkipped || resp.InvestigationMode != "ai" {
		t.Fatalf("unverified article must not bypass Gemini: calls=%d %+v", ai.calls.Load(), resp)
	}
}

func TestRouterNonMatchingVerifiedArticleUsesGemini(t *testing.T) {
	redis := "service=orderflow|dependency=redis|failure=degraded|endpoint=/api/products"
	withKnowledge(t, []KnowledgeArticle{{
		Title: "Redis", Summary: "s", ErrorSignature: redis,
		MatchedOn: []string{redis}, Verified: true,
	}})

	ai := &fakeAnalyzer{text: "AI analysis"}
	resp := runInvestigation(t, &API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: ai})

	if ai.calls.Load() != 1 || resp.AISkipped {
		t.Fatalf("non-matching article must not skip Gemini: %+v", resp)
	}
	if resp.AIExplanation != "AI analysis" {
		t.Fatalf("expected AI explanation, got %q", resp.AIExplanation)
	}
}

func TestRouterMissingKeyStillReturnsEvidence(t *testing.T) {
	withKnowledge(t, nil)

	mcp := &fakeMCP{evidence: postgresEvidence()}
	analyzer := NewGeminiAnalyzer(Config{GeminiModel: "m"}, NewGeminiKeyStore(""))
	resp := runInvestigation(t, &API{mcp: mcp, gemini: analyzer})

	if mcp.calls.Load() != 1 || resp.Evidence == nil {
		t.Fatal("evidence must be collected without a key")
	}
	if !strings.Contains(resp.AIError, "not configured") {
		t.Fatalf("expected configuration error, got %q", resp.AIError)
	}
	if _, err := os.Stat(knowledgeFile); err == nil {
		t.Fatal("a failed AI run must not be learned")
	}
}

func TestRouterGeminiFailureFallback(t *testing.T) {
	withKnowledge(t, nil)

	ai := &fakeAnalyzer{err: &GeminiError{Kind: "unavailable", Message: "Gemini is temporarily unavailable. Try again shortly."}}
	resp := runInvestigation(t, &API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: ai})

	if resp.Evidence == nil || !strings.Contains(resp.AIExplanation, "AI explanation unavailable") {
		t.Fatalf("fallback explanation expected: %+v", resp)
	}
	if resp.AIError == "" {
		t.Fatal("sanitized AI error expected")
	}
}

func TestRouterLearnsOnlyFromSuccessfulGemini(t *testing.T) {
	withKnowledge(t, nil)

	ai := &fakeAnalyzer{text: "### Summary\nAI diagnosis"}
	runInvestigation(t, &API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: ai})

	saved, err := LoadSavedKnowledge()
	if err != nil || len(saved) != 1 {
		t.Fatalf("expected one learned article, got %d (%v)", len(saved), err)
	}
	if saved[0].Verified || saved[0].Source != "AI-generated" || saved[0].Summary != "### Summary\nAI diagnosis" {
		t.Fatalf("unexpected learned article: %+v", saved[0])
	}
}

func TestRouterDeterministicExplanationIsNotLearned(t *testing.T) {
	withKnowledge(t, []KnowledgeArticle{verifiedPostgresArticle()})

	runInvestigation(t, &API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: &fakeAnalyzer{}})

	saved, _ := LoadSavedKnowledge()
	if len(saved) != 1 {
		t.Fatalf("knowledge-based run must not save a new article, have %d", len(saved))
	}
}

func TestInvestigationResponseFieldsCompatible(t *testing.T) {
	withKnowledge(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/investigations",
		strings.NewReader(`{"service":"orderflow","duration":"5m"}`))
	rec := httptest.NewRecorder()
	(&API{mcp: &fakeMCP{evidence: postgresEvidence()}, gemini: &fakeAnalyzer{text: "x"}}).Investigate(rec, req)

	var raw map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &raw)
	for _, field := range []string{
		"service", "duration", "evidence", "ai_explanation", "duration_ms",
		"knowledge", "investigation_mode", "ai_skipped", "match_score", "decision_reason",
	} {
		if _, ok := raw[field]; !ok {
			t.Errorf("missing response field %q", field)
		}
	}
}
