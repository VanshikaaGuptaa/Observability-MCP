package service

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"orderflow/internal/failure"
	"orderflow/internal/metrics"
)

type PaymentService struct {
	metrics   *metrics.Metrics
	simulator *failure.Simulator
}

func NewPaymentService(
	metrics *metrics.Metrics,
	simulator *failure.Simulator,
) *PaymentService {

	return &PaymentService{
		metrics:   metrics,
		simulator: simulator,
	}
}

type PaymentRequest struct {
	OrderID int64   `json:"orderId"`
	Amount  float64 `json:"amount"`
}

type PaymentResponse struct {
	OrderID int64  `json:"orderId"`
	Status  string `json:"status"`
}

func (s *PaymentService) ProcessPayment(
	ctx context.Context,
	request PaymentRequest,
) (*PaymentResponse, error) {

	if request.OrderID <= 0 {
		return nil, fmt.Errorf("invalid order id")
	}

	if request.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	// Temporary simulated payment processing.
	time.Sleep(100 * time.Millisecond)

	if s.simulator.IsEnabled(failure.PaymentFailure) {
		s.metrics.PaymentFailureTotal.
			WithLabelValues("provider_unavailable").
			Inc()

		slog.Error(
			"payment processing failed: payment provider unavailable",
			"operation", "process_payment",
			"dependency", "payment_gateway",
			"reason", "provider_unavailable",
			"error", "payment provider unavailable",
			"order_id", request.OrderID,
			"amount", request.Amount,
		)

		return nil, fmt.Errorf(
			"payment provider unavailable",
		)
	}

	if rand.Float64() < 0.05 {

		s.metrics.PaymentFailureTotal.
			WithLabelValues("provider_unavailable").
			Inc()

		slog.Error(
			"payment processing failed: payment provider unavailable",
			"operation", "process_payment",
			"dependency", "payment_gateway",
			"reason", "provider_unavailable",
			"error", "payment provider unavailable",
			"order_id", request.OrderID,
			"amount", request.Amount,
		)

		return nil, fmt.Errorf(
			"payment provider unavailable",
		)
	}

	s.metrics.PaymentSuccessTotal.Inc()

	return &PaymentResponse{
		OrderID: request.OrderID,
		Status:  "SUCCESS",
	}, nil
}
