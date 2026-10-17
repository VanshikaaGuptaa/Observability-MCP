package main

import (
	"errors"
	"strings"
	"sync"
	"unicode"
)

// Key sources reported to the frontend. The key itself is never reported.
const (
	GeminiKeySourceUser = "user"
	GeminiKeySourceEnv  = "environment"
	GeminiKeySourceNone = "none"
)

// maxGeminiKeyLength bounds user input. Real Gemini keys are far shorter.
const maxGeminiKeyLength = 256

var (
	// ErrGeminiNotConfigured is returned when neither a user-provided key
	// nor an environment fallback key is available.
	ErrGeminiNotConfigured = errors.New(
		"Gemini API key is not configured. Add a key in Settings or set GEMINI_API_KEY.",
	)
	errGeminiKeyBlank   = errors.New("API key must not be blank")
	errGeminiKeyTooLong = errors.New("API key is too long")
	errGeminiKeyInvalid = errors.New("API key must not contain whitespace or control characters")
)

// GeminiKeyStatus is the non-sensitive configuration state.
type GeminiKeyStatus struct {
	Configured     bool   `json:"configured"`
	Source         string `json:"source"`
	EnvFallback    bool   `json:"env_fallback_available"`
	UserConfigured bool   `json:"user_key_configured"`
}

// GeminiKeyStore holds the runtime Gemini key configuration in memory.
//
// Precedence: user-provided key, then environment key, then unconfigured.
// The user key is never written to disk and is lost on restart.
type GeminiKeyStore struct {
	mu      sync.RWMutex
	userKey string
	envKey  string
}

func NewGeminiKeyStore(envKey string) *GeminiKeyStore {
	return &GeminiKeyStore{envKey: strings.TrimSpace(envKey)}
}

// ValidateGeminiKey checks the shape of a user-provided key without
// contacting Gemini. A key passing validation is not necessarily valid.
func ValidateGeminiKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errGeminiKeyBlank
	}
	if len(key) > maxGeminiKeyLength {
		return "", errGeminiKeyTooLong
	}
	for _, r := range key {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", errGeminiKeyInvalid
		}
	}
	return key, nil
}

// SetUserKey replaces the in-memory user key.
func (s *GeminiKeyStore) SetUserKey(key string) error {
	clean, err := ValidateGeminiKey(key)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.userKey = clean
	s.mu.Unlock()
	return nil
}

// ClearUserKey removes the user key so the environment fallback (if any)
// becomes active immediately.
func (s *GeminiKeyStore) ClearUserKey() {
	s.mu.Lock()
	s.userKey = ""
	s.mu.Unlock()
}

// Active returns a consistent snapshot of the key to use and its source.
func (s *GeminiKeyStore) Active() (key string, source string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.userKey != "" {
		return s.userKey, GeminiKeySourceUser
	}
	if s.envKey != "" {
		return s.envKey, GeminiKeySourceEnv
	}
	return "", GeminiKeySourceNone
}

// Status reports configuration state without revealing any key material.
func (s *GeminiKeyStore) Status() GeminiKeyStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := GeminiKeyStatus{
		EnvFallback:    s.envKey != "",
		UserConfigured: s.userKey != "",
		Source:         GeminiKeySourceNone,
	}
	switch {
	case s.userKey != "":
		status.Configured = true
		status.Source = GeminiKeySourceUser
	case s.envKey != "":
		status.Configured = true
		status.Source = GeminiKeySourceEnv
	}
	return status
}
