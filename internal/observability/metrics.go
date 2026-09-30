// Package observability owns the runtime telemetry surface: Prometheus
// metrics, the /metrics HTTP handler, and a middleware that records
// per-request counters and histograms.
package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics owns the Prometheus collectors used by the server. Construct once
// at startup and pass the same instance to the instrumenting middleware and
// the /metrics handler.
type Metrics struct {
	registry         *prometheus.Registry
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

// New creates a Metrics with the default Go runtime + process collectors
// registered. The HTTP request duration histogram uses sensible buckets for
// a typical web service.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	m := &Metrics{
		registry: reg,
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "agent_shaker",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path", "status"},
		),
		requestsInFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "agent_shaker",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Number of HTTP requests currently being handled.",
		}),
	}
	reg.MustRegister(m.requestDuration, m.requestsInFlight)
	return m
}

// Registry exposes the underlying registry for advanced use (e.g. registering
// custom collectors from other packages).
func (m *Metrics) Registry() *prometheus.Registry {
	return m.registry
}

// Handler returns an http.Handler that serves the Prometheus metrics in the
// text exposition format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		Registry:          m.registry,
		EnableOpenMetrics: true,
	})
}

// Instrument wraps next so each request is counted in the duration histogram
// and tracked by the in-flight gauge. Status codes are bucketed via
// strconv.Itoa so Prometheus label cardinality stays bounded by route × status.
func (m *Metrics) Instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()
		wrapped := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		m.requestDuration.WithLabelValues(
			r.Method,
			r.URL.Path,
			strconv.Itoa(wrapped.statusCode),
		).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.wroteHeader {
		return
	}
	s.statusCode = code
	s.wroteHeader = true
	s.ResponseWriter.WriteHeader(code)
}
