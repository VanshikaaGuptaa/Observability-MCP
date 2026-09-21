package service

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"orderflow/internal/failure"
	"orderflow/internal/metrics"
)

func TestPaymentService_PaymentFailureScenario(t *testing.T) {
	m := metrics.New()
	sim := failure.NewSimulator()
	svc := NewPaymentService(m, sim)

	// 1. Verify payment_failure scenario fails deterministically
	sim.SetScenario(failure.PaymentFailure)
	req := PaymentRequest{OrderID: 101, Amount: 100.0}

	initialFailures := testutil.ToFloat64(m.PaymentFailureTotal.WithLabelValues("provider_unavailable"))

	resp, err := svc.ProcessPayment(context.Background(), req)
	if err == nil {
		t.Fatalf("expected payment error in payment_failure scenario, got nil")
	}
	if err.Error() != "payment provider unavailable" {
		t.Errorf("expected 'payment provider unavailable', got '%v'", err)
	}
	if resp != nil {
		t.Fatalf("expected nil response on failure, got %+v", resp)
	}

	// Verify Prometheus metric incremented
	newFailures := testutil.ToFloat64(m.PaymentFailureTotal.WithLabelValues("provider_unavailable"))
	if newFailures != initialFailures+1 {
		t.Errorf("expected PaymentFailureTotal to increase by 1, got %v -> %v", initialFailures, newFailures)
	}

	// 2. Revert to normal: verify validation errors and non-failure path
	sim.SetScenario(failure.Normal)

	// Invalid order ID
	_, err = svc.ProcessPayment(context.Background(), PaymentRequest{OrderID: 0, Amount: 100.0})
	if err == nil {
		t.Fatalf("expected validation error for zero order ID")
	}

	// Invalid amount
	_, err = svc.ProcessPayment(context.Background(), PaymentRequest{OrderID: 101, Amount: 0})
	if err == nil {
		t.Fatalf("expected validation error for zero amount")
	}
}
