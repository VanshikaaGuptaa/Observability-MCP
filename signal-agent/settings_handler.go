package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
)

// geminiSettings is the configuration surface used by the Settings API.
type geminiSettings interface {
	Status() GeminiKeyStatus
	SetUserKey(key string) error
	ClearUserKey()
}

// connectionTester performs a minimal Gemini request with the active key.
type connectionTester interface {
	TestConnection(ctx context.Context) (string, error)
}

type geminiKeyRequest struct {
	APIKey string `json:"api_key"`
}

type geminiTestResponse struct {
	GeminiKeyStatus
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiErrorResponse{Error: message})
}

// isLoopbackRequest reports whether the TCP peer is on this machine.
// The Vite dev proxy connects from localhost, so the dashboard works,
// while other machines on the network cannot change the shared key.
func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// requireJSON forces a JSON content type on mutating requests. Browsers
// must send a CORS preflight for this, which the API never approves, so
// a third-party web page cannot change or test the key (CSRF).
func requireJSON(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

func (a *API) settingsGuard(w http.ResponseWriter, r *http.Request) bool {
	if a.settings == nil || a.tester == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "Settings are not available.")
		return false
	}
	if !a.allowRemoteSettings && !isLoopbackRequest(r) {
		writeAPIError(w, http.StatusForbidden, "Settings can only be changed from the local machine.")
		return false
	}
	if r.Method != http.MethodGet && !requireJSON(r) {
		writeAPIError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json.")
		return false
	}
	return true
}

// GeminiSettings handles GET, PUT and DELETE on /api/settings/gemini.
func (a *API) GeminiSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodPut, http.MethodDelete:
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if !a.settingsGuard(w, r) {
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.settings.Status())

	case http.MethodPut:
		r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
		defer r.Body.Close()

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		var req geminiKeyRequest
		if err := decoder.Decode(&req); err != nil {
			writeAPIError(w, http.StatusBadRequest, "Invalid JSON request.")
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			writeAPIError(w, http.StatusBadRequest, "Request must contain a single JSON object.")
			return
		}

		if err := a.settings.SetUserKey(req.APIKey); err != nil {
			// Validation messages never contain the submitted key.
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Printf("Gemini API key updated (source=user)")
		writeJSON(w, http.StatusOK, a.settings.Status())

	case http.MethodDelete:
		a.settings.ClearUserKey()
		status := a.settings.Status()
		log.Printf("Gemini user API key removed (active source=%s)", status.Source)
		writeJSON(w, http.StatusOK, status)
	}
}

// TestGeminiSettings handles POST /api/settings/gemini/test.
func (a *API) TestGeminiSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !a.settingsGuard(w, r) {
		return
	}

	_, err := a.tester.TestConnection(r.Context())
	status := a.settings.Status()

	if err == nil {
		writeJSON(w, http.StatusOK, geminiTestResponse{
			GeminiKeyStatus: status,
			OK:              true,
			Message:         "Connection successful.",
		})
		return
	}

	message := "The Gemini connection test failed."
	var safe *GeminiError
	switch {
	case errors.Is(err, ErrGeminiNotConfigured):
		message = ErrGeminiNotConfigured.Error()
	case errors.As(err, &safe):
		message = safe.Message
	}

	// 200 with ok=false: the test ran, the key just did not work.
	writeJSON(w, http.StatusOK, geminiTestResponse{
		GeminiKeyStatus: status,
		OK:              false,
		Message:         message,
	})
}
