package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"orderflow/internal/config"
)

func NewRedisClient(
	ctx context.Context,
	cfg config.Config,
) (*redis.Client, error) {

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"%s:%s",
			cfg.RedisHost,
			cfg.RedisPort,
		),
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}
