package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"orderflow/internal/failure"
	"orderflow/internal/metrics"
	"orderflow/internal/service"
)

func TestPaymentHandler_PaymentFailureScenario(t *testing.T) {
	m := metrics.New()
	sim := failure.NewSimulator()
	svc := service.NewPaymentService(m, sim)
	h := NewPaymentHandler(svc)

	// 1. In payment_failure scenario: must return HTTP 502 Bad Gateway
	sim.SetScenario(failure.PaymentFailure)

	payload := map[string]interface{}{
		"orderId": 201,
		"amount":  1500.0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ProcessPayment(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502 Bad Gateway, got %d. Response: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp["error"] != "payment provider unavailable" {
		t.Errorf("expected error 'payment provider unavailable', got '%s'", resp["error"])
	}

	// 2. Revert to normal: verify bad request body returns 400
	sim.SetScenario(failure.Normal)

	badReq := httptest.NewRequest(http.MethodPost, "/api/payment", bytes.NewReader([]byte("invalid json")))
	badReq.Header.Set("Content-Type", "application/json")
	badW := httptest.NewRecorder()

	h.ProcessPayment(badW, badReq)

	if badW.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request for malformed body, got %d", badW.Code)
	}
}
