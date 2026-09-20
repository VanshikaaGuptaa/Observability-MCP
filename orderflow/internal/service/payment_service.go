package service

import (
	"context"
	"fmt"
	"math/rand"
	"orderflow/internal/metrics"
	"time"
)

type PaymentService struct {
	metrics *metrics.Metrics
}

func NewPaymentService(
	metrics *metrics.Metrics,
) *PaymentService {

	return &PaymentService{
		metrics: metrics,
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
	// Later we will deliberately introduce latency
	// and failure scenarios here.

	time.Sleep(100 * time.Millisecond)

	if rand.Float64() < 0.05 {

		s.metrics.PaymentFailureTotal.
			WithLabelValues("provider_unavailable").
			Inc()

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
