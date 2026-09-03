package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	requestTotal     *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

func NewMetrics() *Metrics {
	m := &Metrics{
		requestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests received.",
			},
			[]string{"method", "route", "status"},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),

		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Current number of HTTP requests being processed.",
			},
		),
	}

	prometheus.MustRegister(
		m.requestTotal,
		m.requestDuration,
		m.requestsInFlight,
	)

	return m
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriterHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (m *Metrics) Observe(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		m.requestTotal.
			WithLabelValues(
				r.Method,
				route,
				strconv.Itoa(rw.statusCode),
			).Inc()

		m.requestDuration.
			WithLabelValues(r.Method, route).
			Observe(time.Since(start).Seconds())
	})
}
