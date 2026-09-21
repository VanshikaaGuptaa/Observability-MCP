package service

import (
	"context"
	"fmt"
	"log/slog"

	"orderflow/internal/cache"
	"orderflow/internal/failure"
	"orderflow/internal/model"
	"orderflow/internal/repository"
)

type ProductService struct {
	repository *repository.ProductRepository
	cache      *cache.ProductCache
	simulator  *failure.Simulator
}

func NewProductService(
	repository *repository.ProductRepository,
	cache *cache.ProductCache,
	simulator *failure.Simulator,
) *ProductService {

	return &ProductService{
		repository: repository,
		cache:      cache,
		simulator:  simulator,
	}
}

func (s *ProductService) GetProducts(
	ctx context.Context,
) ([]model.Product, error) {

	if s.simulator.IsEnabled(failure.ErrorSpike) {
		slog.Error(
			"error spike simulated in product catalog",
			"operation", "get_products",
			"dependency", "catalog_service",
			"error", "internal server error: service experiencing unexpected failure spike",
		)

		return nil, fmt.Errorf("internal server error: catalog error spike")
	}

	return s.repository.FindAll(ctx)
}

func (s *ProductService) GetProduct(
	ctx context.Context,
	id int64,
) (*model.Product, error) {

	cachedProduct, err := s.cache.Get(ctx, id)

	if err != nil {
		// Cache failure should not break the application.
		cachedProduct = nil
	}

	if cachedProduct != nil {
		return cachedProduct, nil
	}

	product, err := s.repository.FindByID(ctx, id)

	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, product); err != nil {
		// Cache write failure shouldn't break the request.
		slog.Warn(
			"failed to cache product",
			"operation", "cache_set",
			"product_id", product.ID,
			"error", err,
		)
	}

	return product, nil
}
