package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"orderflow/internal/failure"
	"orderflow/internal/metrics"
	"orderflow/internal/model"
)

type ProductRepository struct {
	db        *pgxpool.Pool
	metrics   *metrics.Metrics
	simulator *failure.Simulator
}

func NewProductRepository(
	db *pgxpool.Pool,
	metrics *metrics.Metrics,
	simulator *failure.Simulator,
) *ProductRepository {

	return &ProductRepository{
		db:        db,
		metrics:   metrics,
		simulator: simulator,
	}
}

func (r *ProductRepository) FindAll(ctx context.Context) ([]model.Product, error) {
	if r.simulator.IsEnabled(failure.DBFailure) {
		slog.Error(
			"database operation failed: connection refused",
			"operation", "find_all_products",
			"dependency", "postgres",
			"error", "database connection refused",
		)

		return nil, fmt.Errorf("database connection refused")
	}

	start := time.Now()

	defer func() {

		r.metrics.DatabaseOperationDuration.
			WithLabelValues("find_all_products").
			Observe(
				time.Since(start).Seconds(),
			)
	}()
	query := `
		SELECT
			id,
			name,
			description,
			price,
			stock_quantity,
			created_at,
			updated_at
		FROM products
		ORDER BY id
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []model.Product

	for rows.Next() {
		var product model.Product

		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.StockQuantity,
			&product.CreatedAt,
			&product.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed while reading products: %w", err)
	}
	r.metrics.DatabaseOperationsTotal.
		WithLabelValues("find_all_products").
		Inc()
	return products, nil
}

func (r *ProductRepository) FindByID(
	ctx context.Context,
	id int64,
) (*model.Product, error) {
	if r.simulator.IsEnabled(failure.DBFailure) {
		slog.Error(
			"database operation failed: connection refused",
			"operation", "find_product_by_id",
			"dependency", "postgres",
			"error", "database connection refused",
			"product_id", id,
		)

		return nil, fmt.Errorf("database connection refused")
	}

	start := time.Now()

	defer func() {

		r.metrics.DatabaseOperationDuration.
			WithLabelValues("find_product_by_id").
			Observe(
				time.Since(start).Seconds(),
			)
	}()
	query := `
		SELECT
			id,
			name,
			description,
			price,
			stock_quantity,
			created_at,
			updated_at
		FROM products
		WHERE id = $1
	`

	var product model.Product

	err := r.db.QueryRow(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.StockQuantity,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("product not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to find product: %w", err)
	}
	r.metrics.DatabaseOperationsTotal.
		WithLabelValues("find_product_by_id").
		Inc()
	return &product, nil
}
