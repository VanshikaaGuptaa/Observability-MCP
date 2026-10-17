package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/genai"
)

// fakeGenerator records which key created it and returns scripted results.
type fakeGenerator struct {
	key   string
	calls *atomic.Int32
	fn    func(key string) (string, error)
}

func (f *fakeGenerator) GenerateText(
	ctx context.Context,
	model, prompt string,
	maxTokens int32,
) (string, error) {
	f.calls.Add(1)
	return f.fn(f.key)
}

func newTestAnalyzer(
	store *GeminiKeyStore,
	fn func(key string) (string, error),
) (*GeminiAnalyzer, *atomic.Int32) {
	calls := &atomic.Int32{}
	a := NewGeminiAnalyzer(Config{GeminiModel: "test-model"}, store)
	a.retryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	a.newGenerator = func(ctx context.Context, key string) (textGenerator, error) {
		return &fakeGenerator{key: key, calls: calls, fn: fn}, nil
	}
	return a, calls
}

func TestKeyStorePrecedenceAndFallback(t *testing.T) {
	store := NewGeminiKeyStore("env-key")

	if key, src := store.Active(); key != "env-key" || src != GeminiKeySourceEnv {
		t.Fatalf("expected env fallback, got %q/%s", key, src)
	}

	if err := store.SetUserKey("  user-key  "); err != nil {
		t.Fatal(err)
	}
	if key, src := store.Active(); key != "user-key" || src != GeminiKeySourceUser {
		t.Fatalf("user key must take precedence, got %q/%s", key, src)
	}

	store.ClearUserKey()
	if key, src := store.Active(); key != "env-key" || src != GeminiKeySourceEnv {
		t.Fatalf("removing user key must restore env fallback, got %q/%s", key, src)
	}
}

func TestKeyStoreUnconfigured(t *testing.T) {
	store := NewGeminiKeyStore("")
	_ = store.SetUserKey("user-key")
	store.ClearUserKey()

	st := store.Status()
	if st.Configured || st.Source != GeminiKeySourceNone || st.EnvFallback {
		t.Fatalf("expected unconfigured status, got %+v", st)
	}
}

func TestKeyStoreRejectsInvalidKeys(t *testing.T) {
	store := NewGeminiKeyStore("")
	for _, bad := range []string{"", "   ", "\t\n", "abc def", strings.Repeat("a", 300)} {
		if err := store.SetUserKey(bad); err == nil {
			t.Errorf("expected rejection for %q", bad)
		}
	}
	if store.Status().Configured {
		t.Fatal("invalid keys must not configure the store")
	}
}

func TestAnalyzeMissingKeyReturnsSafeError(t *testing.T) {
	a, calls := newTestAnalyzer(NewGeminiKeyStore(""), func(string) (string, error) {
		return "should not be called", nil
	})

	_, err := a.Analyze(context.Background(), "orderflow", "5m", map[string]any{})
	if !errors.Is(err, ErrGeminiNotConfigured) {
		t.Fatalf("expected ErrGeminiNotConfigured, got %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("Gemini must not be called without a key")
	}
}

func TestAnalyzeUsesLatestKey(t *testing.T) {
	store := NewGeminiKeyStore("env-key")
	a, _ := newTestAnalyzer(store, func(key string) (string, error) {
		return "analysis with " + key, nil
	})

	check := func(want string) {
		t.Helper()
		got, err := a.Analyze(context.Background(), "orderflow", "5m", nil)
		if err != nil {
			t.Fatal(err)
		}
		if got != "analysis with "+want {
			t.Fatalf("expected key %q to be used, got %q", want, got)
		}
	}

	check("env-key")
	_ = store.SetUserKey("user-1")
	check("user-1")
	_ = store.SetUserKey("user-2")
	check("user-2")
	store.ClearUserKey()
	check("env-key")
}

func TestAnalyzeSanitizesUpstreamErrors(t *testing.T) {
	const secret = "AIzaSECRET-should-never-leak"
	store := NewGeminiKeyStore("")
	_ = store.SetUserKey(secret)

	a, calls := newTestAnalyzer(store, func(key string) (string, error) {
		return "", genai.APIError{
			Code:    400,
			Status:  "INVALID_ARGUMENT",
			Message: "API key not valid: " + key + " API_KEY_INVALID",
		}
	})

	_, err := a.Analyze(context.Background(), "orderflow", "5m", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked secret: %v", err)
	}
	var safe *GeminiError
	if !errors.As(err, &safe) || safe.Kind != "invalid_key" {
		t.Fatalf("expected invalid_key GeminiError, got %#v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("permanent errors must not be retried, got %d calls", calls.Load())
	}
}

func TestAnalyzeRetriesTransientErrorsWithSameKey(t *testing.T) {
	store := NewGeminiKeyStore("")
	_ = store.SetUserKey("key-a")

	var mu sync.Mutex
	var seen []string
	attempt := 0

	a, _ := newTestAnalyzer(store, func(key string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, key)
		attempt++
		if attempt == 1 {
			// Change the key mid-request; the in-flight request must keep
			// using the key it started with.
			_ = store.SetUserKey("key-b")
			return "", genai.APIError{Code: 503, Status: "UNAVAILABLE"}
		}
		return "ok", nil
	})

	if _, err := a.Analyze(context.Background(), "orderflow", "5m", nil); err != nil {
		t.Fatal(err)
	}
	for _, k := range seen {
		if k != "key-a" {
			t.Fatalf("in-flight request switched keys: %v", seen)
		}
	}
}

func TestClassifyGeminiError(t *testing.T) {
	cases := map[string]error{
		"invalid_key": genai.APIError{Code: 400, Message: "API key not valid. API_KEY_INVALID"},
		"permission":  genai.APIError{Code: 403},
		"model":       genai.APIError{Code: 404},
		"rate_limit":  genai.APIError{Code: 429},
		"unavailable": genai.APIError{Code: 503},
		"timeout":     context.DeadlineExceeded,
		"unknown":     errors.New("weird"),
	}
	for want, err := range cases {
		if got := classifyGeminiError(err); got.Kind != want {
			t.Errorf("%v: expected %s, got %s", err, want, got.Kind)
		}
	}
}

func TestTestConnection(t *testing.T) {
	const secret = "AIzaTEST-secret"
	store := NewGeminiKeyStore("")

	a, _ := newTestAnalyzer(store, func(key string) (string, error) {
		if key == "good-key" {
			return "OK", nil
		}
		return "", genai.APIError{Code: 400, Message: "API key not valid " + key}
	})

	if _, err := a.TestConnection(context.Background()); !errors.Is(err, ErrGeminiNotConfigured) {
		t.Fatalf("expected not configured, got %v", err)
	}

	_ = store.SetUserKey("good-key")
	if _, err := a.TestConnection(context.Background()); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	_ = store.SetUserKey(secret)
	_, err := a.TestConnection(context.Background())
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("expected sanitized failure, got %v", err)
	}
}

func TestConcurrentKeyUpdatesAndAnalysis(t *testing.T) {
	store := NewGeminiKeyStore("env-key")
	a, _ := newTestAnalyzer(store, func(key string) (string, error) {
		return key, nil
	})

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if j%3 == 0 {
					store.ClearUserKey()
				} else {
					_ = store.SetUserKey("user-key-" + string(rune('a'+i)))
				}
				_ = store.Status()
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				got, err := a.Analyze(ctx, "orderflow", "5m", nil)
				if err != nil {
					t.Error(err)
					return
				}
				if got != "env-key" && !strings.HasPrefix(got, "user-key-") {
					t.Errorf("unexpected key used: %q", got)
					return
				}
			}
		}()
	}
	wg.Wait()
}
