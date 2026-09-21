package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"orderflow/internal/cache"
	"orderflow/internal/config"
	"orderflow/internal/failure"
	"orderflow/internal/handler"
	"orderflow/internal/metrics"
	"orderflow/internal/middleware"
	"orderflow/internal/repository"
	"orderflow/internal/service"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	cfg := config.Load()

	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelInfo,
			},
		),
	)

	slog.SetDefault(logger)

	ctx := context.Background()
	registry := prometheus.NewRegistry()

	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(
			prometheus.ProcessCollectorOpts{},
		),
	)

	applicationMetrics := metrics.New()
	failureSimulator := failure.NewSimulator()

	if err := applicationMetrics.Register(
		registry,
	); err != nil {

		logger.Error(
			"failed to register metrics",
			"error",
			err,
		)

		os.Exit(1)
	}
	// -------------------------
	// PostgreSQL
	// -------------------------

	db, err := repository.NewDatabasePool(
		ctx,
		cfg,
	)

	if err != nil {
		logger.Error(
			"failed to initialize database",
			"error",
			err,
		)

		os.Exit(1)
	}

	defer db.Close()

	// -------------------------
	// Redis
	// -------------------------

	redisClient, err := cache.NewRedisClient(
		ctx,
		cfg,
	)

	if err != nil {
		logger.Error(
			"failed to initialize redis",
			"error",
			err,
		)

		os.Exit(1)
	}

	defer redisClient.Close()

	// -------------------------
	// Repositories
	// -------------------------

	productRepository :=
		repository.NewProductRepository(
			db,
			applicationMetrics,
			failureSimulator,
		)

	orderRepository :=
		repository.NewOrderRepository(
			db,
			applicationMetrics,
			failureSimulator,
		)

	// -------------------------
	// Cache
	// -------------------------

	productCache :=
		cache.NewProductCache(
			redisClient,
			applicationMetrics,
			failureSimulator,
		)

	// -------------------------
	// Services
	// -------------------------

	productService :=
		service.NewProductService(
			productRepository,
			productCache,
			failureSimulator,
		)

	orderService :=
		service.NewOrderService(
			orderRepository,
			productRepository,
			applicationMetrics,
			failureSimulator,
		)

	paymentService :=
		service.NewPaymentService(
			applicationMetrics,
			failureSimulator,
		)

	// -------------------------
	// Handlers
	// -------------------------
	deploymentHandler := handler.NewDeploymentHandler()
	productHandler :=
		handler.NewProductHandler(
			productService,
		)

	orderHandler :=
		handler.NewOrderHandler(
			orderService,
		)

	paymentHandler :=
		handler.NewPaymentHandler(
			paymentService,
		)

	healthHandler :=
		handler.NewHealthHandler(
			db,
			redisClient,
		)
	failureHandler := handler.NewFailureHandler(failureSimulator)
	// -------------------------
	// HTTP routes
	// -------------------------

	mux := http.NewServeMux()
	mux.Handle(
		"/metrics",
		promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{},
		),
	)
	mux.HandleFunc(
		"/api/products",
		func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			productHandler.GetProducts(w, r)
		},
	)

	mux.HandleFunc(
		"/api/products/",
		func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			productHandler.GetProduct(w, r)
		},
	)

	mux.HandleFunc(
		"/api/orders",
		func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			orderHandler.CreateOrder(w, r)
		},
	)

	mux.HandleFunc(
		"/api/payment",
		func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			paymentHandler.ProcessPayment(w, r)
		},
	)

	mux.HandleFunc(
		"/api/health",
		healthHandler.Health,
	)
	mux.HandleFunc(
		"POST /api/admin/failure-scenario",
		failureHandler.SetScenario,
	)

	mux.HandleFunc(
		"GET /api/admin/failure-scenario",
		failureHandler.GetScenario,
	)
	mux.HandleFunc(
		"GET /api/deployments",
		deploymentHandler.GetDeployments,
	)
	// -------------------------
	// HTTP server
	// -------------------------

	handlerWithMetrics := middleware.Metrics(
		applicationMetrics,
		mux,
	)

	handlerWithLogging := middleware.Logging(
		logger,
		handlerWithMetrics,
	)

	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           handlerWithLogging,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// -------------------------
	// Start server
	// -------------------------

	go func() {

		logger.Info(
			"order-service started",
			"port",
			cfg.AppPort,
		)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			logger.Error(
				"server failed",
				"error",
				err,
			)

			os.Exit(1)
		}
	}()

	// -------------------------
	// Graceful shutdown
	// -------------------------

	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-stop

	logger.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(
		shutdownCtx,
	); err != nil {

		logger.Error(
			"graceful shutdown failed",
			"error",
			err,
		)
	}
}
