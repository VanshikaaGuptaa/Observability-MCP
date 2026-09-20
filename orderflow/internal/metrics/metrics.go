package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	HTTPRequestsTotal *prometheus.CounterVec

	HTTPRequestDuration *prometheus.HistogramVec

	HTTPRequestsInFlight *prometheus.GaugeVec

	OrdersCreatedTotal prometheus.Counter

	OrdersFailedTotal *prometheus.CounterVec

	PaymentSuccessTotal prometheus.Counter

	PaymentFailureTotal *prometheus.CounterVec

	OrderProcessingDuration prometheus.Histogram

	CacheHitsTotal *prometheus.CounterVec

	CacheMissesTotal *prometheus.CounterVec

	CacheErrorsTotal *prometheus.CounterVec

	DatabaseOperationsTotal *prometheus.CounterVec

	DatabaseOperationDuration *prometheus.HistogramVec
}

func New() *Metrics {

	return &Metrics{

		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests.",
			},
			[]string{
				"method",
				"route",
				"status",
			},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "orderflow",
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets: []float64{
					0.005,
					0.01,
					0.025,
					0.05,
					0.1,
					0.25,
					0.5,
					1,
					2.5,
					5,
					10,
				},
			},
			[]string{
				"method",
				"route",
			},
		),

		HTTPRequestsInFlight: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "orderflow",
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being processed.",
			},
			[]string{
				"method",
				"route",
			},
		),

		OrdersCreatedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "orders_created_total",
				Help:      "Total number of successfully created orders.",
			},
		),

		OrdersFailedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "orders_failed_total",
				Help:      "Total number of failed order creation attempts.",
			},
			[]string{
				"reason",
			},
		),

		PaymentSuccessTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "payment_success_total",
				Help:      "Total number of successful payments.",
			},
		),

		PaymentFailureTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "payment_failure_total",
				Help:      "Total number of failed payments.",
			},
			[]string{
				"reason",
			},
		),

		OrderProcessingDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "orderflow",
				Name:      "order_processing_duration_seconds",
				Help:      "Time spent processing order creation.",
				Buckets: []float64{
					0.01,
					0.025,
					0.05,
					0.1,
					0.25,
					0.5,
					1,
					2.5,
					5,
				},
			},
		),

		CacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits.",
			},
			[]string{
				"cache",
			},
		),

		CacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses.",
			},
			[]string{
				"cache",
			},
		),

		CacheErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "cache_errors_total",
				Help:      "Total number of cache errors.",
			},
			[]string{
				"cache",
			},
		),

		DatabaseOperationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "orderflow",
				Name:      "database_operations_total",
				Help:      "Total number of database operations.",
			},
			[]string{
				"operation",
			},
		),

		DatabaseOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "orderflow",
				Name:      "database_operation_duration_seconds",
				Help:      "Database operation duration in seconds.",
				Buckets: []float64{
					0.001,
					0.005,
					0.01,
					0.025,
					0.05,
					0.1,
					0.25,
					0.5,
					1,
					2.5,
				},
			},
			[]string{
				"operation",
			},
		),
	}
}
func (m *Metrics) Register(
	registry *prometheus.Registry,
) error {

	collectors := []prometheus.Collector{
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,

		m.OrdersCreatedTotal,
		m.OrdersFailedTotal,

		m.PaymentSuccessTotal,
		m.PaymentFailureTotal,

		m.OrderProcessingDuration,

		m.CacheHitsTotal,
		m.CacheMissesTotal,
		m.CacheErrorsTotal,

		m.DatabaseOperationsTotal,
		m.DatabaseOperationDuration,
	}

	for _, collector := range collectors {

		if err := registry.Register(collector); err != nil {
			return err
		}
	}

	return nil
}
