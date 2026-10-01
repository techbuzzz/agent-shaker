package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/observability"
)

// hijackableRecorder is a ResponseWriter that supports hijacking, standing in
// for the one net/http hands a real server.
type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *http.ResponseController, error) {
	return nil, nil, nil
}

// probeHandler reports whether the ResponseWriter it receives still exposes
// http.Hijacker. That is the exact capability gorilla/websocket's Upgrader
// requires.
func probeHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(http.Hijacker); !ok {
			t.Errorf("ResponseWriter does not implement http.Hijacker: %T", w)
		}
		w.WriteHeader(http.StatusSwitchingProtocols)
	})
}

// TestWebSocketRoutePreservesHijacker guards the /ws route against a
// regression that is invisible to unit tests of the individual middlewares:
// a middleware in the outer chain silently wrapping the ResponseWriter makes
// the WebSocket handshake fail at runtime with
// "response does not implement http.Hijacker".
//
// The chain under test is the real one from newServeMux.
func TestWebSocketRoutePreservesHijacker(t *testing.T) {
	obs := observability.New()

	chain := middleware.Apply(
		probeHandler(t),
		middleware.RequestID,
		otelhttp.NewMiddleware("agent-shaker.http",
			otelhttp.WithFilter(skipWebSocketTrace),
		),
		obs.Instrument,
		middleware.Recovery,
		middleware.SecurityHeaders(false, ""),
	)

	rec := &httptest.ResponseRecorder{}
	req := httptest.NewRequest(http.MethodGet, "/ws?project_id=abc", nil)
	chain.ServeHTTP(rec, req)
}

// TestHTTPRouteIsNotFiltered is the counterpart: the otelhttp filter must
// exclude /ws and nothing else. It asserts the filter's own decision so the
// guarantee is stated in terms of the contract rather than side effects.
func TestHTTPRouteIsNotFiltered(t *testing.T) {
	tests := []struct {
		path       string
		wantPassed bool
	}{
		{path: "/ws", wantPassed: false},
		{path: "/api/projects", wantPassed: true},
		{path: "/api/tasks", wantPassed: true},
		{path: "/healthz", wantPassed: true},
		{path: "/metrics", wantPassed: true},
	}

	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		if got := skipWebSocketTrace(req); got != tc.wantPassed {
			t.Errorf("skipWebSocketTrace(%s) = %v, want %v", tc.path, got, tc.wantPassed)
		}
	}
}

// TestHijackerWiredThroughMetricsAndLogger covers the two custom wrappers
// directly: both must forward Hijack/Flush instead of hiding the interfaces.
func TestHijackerWiredThroughMetricsAndLogger(t *testing.T) {
	obs := observability.New()

	for name, wrap := range map[string]func(http.Handler) http.Handler{
		"Instrument": obs.Instrument,
		"Logger":     middleware.Logger,
	} {
		t.Run(name, func(t *testing.T) {
			var sawHijacker, sawFlusher bool
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, sawHijacker = w.(http.Hijacker)
				_, sawFlusher = w.(http.Flusher)
			})
			wrap(inner).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ws", nil))
			if !sawHijacker {
				t.Error("wrapped writer does not expose http.Hijacker")
			}
			if !sawFlusher {
				t.Error("wrapped writer does not expose http.Flusher")
			}
		})
	}
}
