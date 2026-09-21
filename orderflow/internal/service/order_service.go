package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"orderflow/internal/failure"
	"orderflow/internal/metrics"
	"orderflow/internal/model"
	"orderflow/internal/repository"
)

type OrderService struct {
	orderRepository   *repository.OrderRepository
	productRepository *repository.ProductRepository
	metrics           *metrics.Metrics
	simulator         *failure.Simulator
}

func NewOrderService(
	orderRepository *repository.OrderRepository,
	productRepository *repository.ProductRepository,
	metrics *metrics.Metrics,
	simulator *failure.Simulator,

) *OrderService {

	return &OrderService{
		orderRepository:   orderRepository,
		productRepository: productRepository,
		metrics:           metrics,
		simulator:         simulator,
	}
}

func (s *OrderService) CreateOrder(
	ctx context.Context,
	request model.CreateOrderRequest,
) (*model.Order, error) {
	s.simulator.ApplyLatency()
	start := time.Now()

	defer func() {
		s.metrics.OrderProcessingDuration.
			Observe(
				time.Since(start).Seconds(),
			)
	}()
	if request.CustomerName == "" {
		s.metrics.OrdersFailedTotal.
			WithLabelValues("validation").
			Inc()

		return nil, fmt.Errorf(
			"customer name is required",
		)
	}

	if len(request.Items) == 0 {
		s.metrics.OrdersFailedTotal.
			WithLabelValues("validation").
			Inc()
		return nil, fmt.Errorf("at least one order item is required")
	}

	order := &model.Order{
		CustomerName: request.CustomerName,
		Status:       "CREATED",
		CreatedAt:    time.Now(),
		Items:        make([]model.OrderItem, 0),
	}

	var total float64

	for _, requestItem := range request.Items {

		if requestItem.Quantity <= 0 {
			s.metrics.OrdersFailedTotal.
				WithLabelValues("validation").
				Inc()
			return nil, fmt.Errorf(
				"quantity must be greater than zero",
			)
		}

		product, err := s.productRepository.FindByID(
			ctx,
			requestItem.ProductID,
		)

		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return nil, fmt.Errorf(
					"product %d not found",
					requestItem.ProductID,
				)
			}

			s.metrics.OrdersFailedTotal.
				WithLabelValues("database").
				Inc()

			return nil, err
		}

		if product.StockQuantity < requestItem.Quantity {
			return nil, fmt.Errorf(
				"insufficient stock for product %d",
				product.ID,
			)
		}

		item := model.OrderItem{
			ProductID: requestItem.ProductID,
			Quantity:  requestItem.Quantity,
			UnitPrice: product.Price,
		}

		order.Items = append(order.Items, item)

		total += product.Price * float64(requestItem.Quantity)
	}

	order.TotalAmount = total

	if err := s.orderRepository.Create(
		ctx,
		order,
	); err != nil {
		s.metrics.OrdersFailedTotal.
			WithLabelValues("database").
			Inc()

		return nil, err
	}

	s.metrics.OrdersCreatedTotal.Inc()

	return order, nil
}
