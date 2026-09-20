package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(
	db *pgxpool.Pool,
	redis *redis.Client,
) *HealthHandler {

	return &HealthHandler{
		db:    db,
		redis: redis,
	}
}

func (h *HealthHandler) Health(
	w http.ResponseWriter,
	r *http.Request,
) {

	ctx, cancel := context.WithTimeout(
		r.Context(),
		2*time.Second,
	)
	defer cancel()

	dbStatus := "UP"
	redisStatus := "UP"

	if err := h.db.Ping(ctx); err != nil {
		dbStatus = "DOWN"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "DOWN"
	}

	status := "UP"

	if dbStatus == "DOWN" || redisStatus == "DOWN" {
		status = "DEGRADED"
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]interface{}{
			"service": "order-service",
			"status":  status,
			"dependencies": map[string]string{
				"postgres": dbStatus,
				"redis":    redisStatus,
			},
		},
	)
}
