// Package observability owns the runtime telemetry surface: Prometheus
// metrics, the /metrics HTTP handler, and a middleware that records
// per-request counters and histograms.
package observability

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
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
		// PromQL: histogram_quantile(0.99, sum by (le) (rate(agent_shaker_http_request_duration_seconds_bucket[5m])))
		//   P99 latency per route.
		// PromQL: sum by (status) (rate(agent_shaker_http_request_duration_seconds_count[5m]))
		//   Request rate split by response status.
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "agent_shaker",
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help: "HTTP request duration in seconds, labelled by method, " +
					"normalised route pattern and response status.",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"method", "route", "status"},
		),
		// PromQL: agent_shaker_http_requests_in_flight
		//   Instantaneous concurrency; a value pinned at the pool limit means
		//   requests are queueing rather than being served.
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

// uuidPattern matches a canonical UUID in any of the hyphenated forms, and is
// the only shape used for entity identifiers in the URL space.
var uuidPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)

// routeLabel renders a bounded Prometheus label for a request path.
//
// r.URL.Path cannot be used directly. Every entity route carries a UUID, so
// labelling by raw path creates one time series per object ever touched — a
// handful of requests against a project would mint thousands of series and
// make the metric unusable, which is a production cost, not a cosmetic issue.
//
// Go's ServeMux does expose the matched pattern on the request it dispatches
// to, but only on a shallow copy the router creates internally. This
// middleware wraps the mux from the outside, so that value is not reachable
// from here; normalising the identifiers is the fix that does not depend on
// restructuring the route table.
func routeLabel(r *http.Request, statusCode int) string {
	// A 404 means the path matched no route, so nothing about it is safe to
	// keep: an unauthenticated client can request arbitrary paths and mint a
	// series per request. Unmatched requests are one bucket by construction.
	if statusCode == http.StatusNotFound {
		return "unmatched"
	}
	return uuidPattern.ReplaceAllString(r.URL.Path, "{id}")
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

		elapsed := time.Since(start).Seconds()
		observer := m.requestDuration.WithLabelValues(
			r.Method,
			routeLabel(r, wrapped.statusCode),
			strconv.Itoa(wrapped.statusCode),
		)

		// Attach the trace id as an exemplar so a latency spike in a dashboard
		// links straight to the trace that produced it. otelhttp sits outside
		// this middleware, so the server span is already in the context here.
		// When tracing is disabled there is no valid span and the observation is
		// recorded without an exemplar.
		if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
			if eo, ok := observer.(prometheus.ExemplarObserver); ok {
				eo.ObserveWithExemplar(elapsed, prometheus.Labels{
					"trace_id": sc.TraceID().String(),
				})
				return
			}
		}
		observer.Observe(elapsed)
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

// Hijack delegates to the wrapped writer so WebSocket upgrades keep working
// while this middleware is in the chain.
//
// Embedding http.ResponseWriter alone hides the optional http.Hijacker
// interface, and gorilla/websocket's Upgrader fails the handshake with
// "response does not implement http.Hijacker". Asserting the interface here —
// rather than skipping the route — means /ws is still counted and timed.
func (s *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := s.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errNotHijacker
	}
	// Once the connection is hijacked no HTTP status will ever be written, so
	// record the 101 ourselves; otherwise the request lands in the 2xx bucket
	// and the duration histogram misreports every WebSocket as an error.
	s.statusCode = http.StatusSwitchingProtocols
	s.wroteHeader = true
	return hj.Hijack()
}

// Flush delegates to the wrapped writer when it supports flushing, which
// streaming handlers rely on to flush each chunk.
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap exposes the underlying writer to http.ResponseController (Go 1.20+),
// which uses it for deadlines and for hijacking-aware writes.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

var errNotHijacker = errors.New("middleware: underlying http.ResponseWriter does not implement http.Hijacker")
