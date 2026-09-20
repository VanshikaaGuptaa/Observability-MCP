package service

import (
	"context"
	"fmt"

	"orderflow/internal/cache"
	"orderflow/internal/model"
	"orderflow/internal/repository"
)

type ProductService struct {
	repository *repository.ProductRepository
	cache      *cache.ProductCache
}

func NewProductService(
	repository *repository.ProductRepository,
	cache *cache.ProductCache,
) *ProductService {

	return &ProductService{
		repository: repository,
		cache:      cache,
	}
}

func (s *ProductService) GetProducts(
	ctx context.Context,
) ([]model.Product, error) {

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
		fmt.Printf("warning: failed to cache product: %v\n", err)
	}

	return product, nil
}
