package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"orderflow/internal/metrics"
	"orderflow/internal/model"
)

type ProductCache struct {
	client  *redis.Client
	ttl     time.Duration
	metrics *metrics.Metrics
}

func NewProductCache(
	client *redis.Client,
	metrics *metrics.Metrics,
) *ProductCache {

	return &ProductCache{
		client:  client,
		ttl:     5 * time.Minute,
		metrics: metrics,
	}
}

func (c *ProductCache) Get(
	ctx context.Context,
	productID int64,
) (*model.Product, error) {

	key := fmt.Sprintf("product:%d", productID)

	value, err := c.client.Get(ctx, key).Result()

	if err == redis.Nil {
		c.metrics.CacheMissesTotal.
			WithLabelValues("redis").
			Inc()

		return nil, nil
	}

	if err != nil {

		c.metrics.CacheErrorsTotal.
			WithLabelValues("redis").
			Inc()

		return nil, fmt.Errorf(
			"redis get failed: %w",
			err,
		)
	}

	var product model.Product

	if err := json.Unmarshal([]byte(value), &product); err != nil {
		return nil, fmt.Errorf("failed to decode cached product: %w", err)
	}

	return &product, nil
}

func (c *ProductCache) Set(
	ctx context.Context,
	product *model.Product,
) error {

	key := fmt.Sprintf("product:%d", product.ID)

	value, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("failed to encode product: %w", err)
	}

	if err := c.client.Set(
		ctx,
		key,
		value,
		c.ttl,
	).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}
