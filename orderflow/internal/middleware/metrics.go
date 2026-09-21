package middleware

import (
	"net/http"
	"strconv"
	"time"

	"orderflow/internal/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {

	r.status = statusCode

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(
	body []byte,
) (int, error) {

	if r.status == 0 {
		r.status = http.StatusOK
	}

	return r.ResponseWriter.Write(body)
}

func Metrics(
	m *metrics.Metrics,
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			route := r.Pattern

			if route == "" {
				route = r.URL.Path
			}

			m.HTTPRequestsInFlight.
				WithLabelValues(
					r.Method,
					route,
				).
				Inc()

			defer m.HTTPRequestsInFlight.
				WithLabelValues(
					r.Method,
					route,
				).
				Dec()

			recorder := &statusRecorder{
				ResponseWriter: w,
			}

			next.ServeHTTP(
				recorder,
				r,
			)

			status := recorder.status

			if status == 0 {
				status = http.StatusOK
			}

			duration := time.Since(start)

			m.HTTPRequestsTotal.
				WithLabelValues(
					r.Method,
					route,
					strconv.Itoa(status),
				).
				Inc()

			m.HTTPRequestDuration.
				WithLabelValues(
					r.Method,
					route,
				).
				Observe(
					duration.Seconds(),
				)
		},
	)
}
