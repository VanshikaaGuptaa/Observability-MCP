package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"orderflow/internal/metrics"
	"orderflow/internal/model"
)

type OrderRepository struct {
	db      *pgxpool.Pool
	metrics *metrics.Metrics
}

func NewOrderRepository(
	db *pgxpool.Pool,
	metrics *metrics.Metrics,
) *OrderRepository {
	return &OrderRepository{
		db:      db,
		metrics: metrics,
	}
}

func (r *OrderRepository) Create(
	ctx context.Context,
	order *model.Order,
) error {
	start := time.Now()

	defer func() {
		r.metrics.DatabaseOperationDuration.
			WithLabelValues("create_order").
			Observe(
				time.Since(start).Seconds(),
			)
	}()

	tx, err := r.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	orderQuery := `
		INSERT INTO orders
			(customer_name, total_amount, status)
		VALUES
			($1, $2, $3)
		RETURNING id, created_at
	`

	err = tx.QueryRow(
		ctx,
		orderQuery,
		order.CustomerName,
		order.TotalAmount,
		order.Status,
	).Scan(
		&order.ID,
		&order.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items
			(order_id, product_id, quantity, unit_price)
		VALUES
			($1, $2, $3, $4)
	`

	for _, item := range order.Items {

		_, err = tx.Exec(
			ctx,
			itemQuery,
			order.ID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
		)

		if err != nil {
			return fmt.Errorf("failed to create order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit order: %w", err)
	}

	r.metrics.DatabaseOperationsTotal.
		WithLabelValues("create_order").
		Inc()

	return nil
}
