package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.opentelemetry.io/otel/trace"
)

// TestRouteLabelBoundsCardinality is the regression test for the label that
// made the request histogram unusable.
//
// Labelling by r.URL.Path put a UUID into every series, so touching one project
// minted a new series per object. Two requests to the same route with
// different ids must land in the same bucket; that is the entire property the
// metric depends on.
func TestRouteLabelBoundsCardinality(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		statusCode int
		want       string
	}{
		{
			name:       "uuid is replaced",
			path:       "/api/projects/8bc63d36-b2be-4ee8-8c8c-931448246fd8/tasks",
			statusCode: http.StatusOK,
			want:       "/api/projects/{id}/tasks",
		},
		{
			name:       "several uuids collapse",
			path:       "/api/projects/8bc63d36-b2be-4ee8-8c8c-931448246fd8/tasks/3af19652-3a0a-4d00-8cb0-a893095c615e",
			statusCode: http.StatusOK,
			want:       "/api/projects/{id}/tasks/{id}",
		},
		{
			name:       "uppercase uuid is also normalised",
			path:       "/api/projects/8BC63D36-B2BE-4EE8-8C8C-931448246FD8",
			statusCode: http.StatusOK,
			want:       "/api/projects/{id}",
		},
		{
			name:       "static route is untouched",
			path:       "/api/projects",
			statusCode: http.StatusOK,
			want:       "/api/projects",
		},
		{
			name:       "a2a task id is normalised",
			path:       "/a2a/v1/tasks/321d5cd0-db4d-425e-8d0c-06d0774666f1",
			statusCode: http.StatusOK,
			want:       "/a2a/v1/tasks/{id}",
		},
		{
			name: "unmatched paths collapse to one bucket",
			// Without this, a client can request any path it likes and mint a
			// series per request: a cardinality problem, and a cheap way to make
			// the endpoint expensive.
			path:       "/not/a/real/route/at/all",
			statusCode: http.StatusNotFound,
			want:       "unmatched",
		},
		{
			name:       "a 404 on an otherwise normal route still reports unmatched",
			path:       "/api/projects/8bc63d36-b2be-4ee8-8c8c-931448246fd8",
			statusCode: http.StatusNotFound,
			want:       "unmatched",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if got := routeLabel(r, tc.statusCode); got != tc.want {
				t.Errorf("routeLabel(%q, %d) = %q, want %q", tc.path, tc.statusCode, got, tc.want)
			}
		})
	}
}

// TestInstrumentCollapsesDistinctIDsIntoOneSeries proves the property end to
// end: two requests to the same route with different ids must produce one
// series, not two. Counting series rather than checking a label value keeps the
// test honest about what the metric is for.
func TestInstrumentCollapsesDistinctIDsIntoOneSeries(t *testing.T) {
	m := New()
	handler := m.Instrument(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{
		"/api/projects/8bc63d36-b2be-4ee8-8c8c-931448246fd8/tasks",
		"/api/projects/3af19652-3a0a-4d00-8cb0-a893095c615e/tasks",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	}

	if got := testutil.CollectAndCount(m.requestDuration, "agent_shaker_http_request_duration_seconds"); got != 1 {
		t.Errorf("recorded %d series, want 1: distinct ids must not create distinct series", got)
	}
}

// seriesCount reports how many distinct label combinations the duration
// histogram currently holds. A child is only created when a request is
// observed, so this is both "how many routes have been hit" and "how many
// time series exist".
func seriesCount(t *testing.T, m *Metrics) int {
	t.Helper()
	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, f := range families {
		if f.GetName() == "agent_shaker_http_request_duration_seconds" {
			return len(f.GetMetric())
		}
	}
	return 0
}

// observationCount reports the total number of recorded observations, across
// every series. Counting series is not enough: a label combination can exist
// with zero samples, which would let a dropped observation pass unnoticed.
func observationCount(t *testing.T, m *Metrics) uint64 {
	t.Helper()
	families, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	var total uint64
	for _, f := range families {
		if f.GetName() != "agent_shaker_http_request_duration_seconds" {
			continue
		}
		for _, mm := range f.GetMetric() {
			total += mm.GetHistogram().GetSampleCount()
		}
	}
	return total
}

// TestInstrumentRecordsWithoutTracing guards the ordering constraint: metrics
// must never depend on tracing being configured. With no span in the context
// the observation still has to be recorded.
func TestInstrumentRecordsWithoutTracing(t *testing.T) {
	m := New()
	handler := m.Instrument(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))

	if got := observationCount(t, m); got != 1 {
		t.Errorf("recorded %d observations, want 1: metrics must not depend on tracing", got)
	}
}

// TestInstrumentAttachesExemplarFromSpanContext covers the link from a latency
// spike back to the trace that produced it. The exemplar rides on the existing
// histogram, so the series count must not change when a trace is present.
func TestInstrumentAttachesExemplarFromSpanContext(t *testing.T) {
	m := New()
	handler := m.Instrument(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		TraceFlags: trace.FlagsSampled,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req = req.WithContext(trace.ContextWithSpanContext(req.Context(), sc))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := seriesCount(t, m); got != 1 {
		t.Errorf("recorded %d series, want 1: the exemplar must not create a series", got)
	}
	if got := observationCount(t, m); got != 1 {
		t.Errorf("recorded %d observations, want 1", got)
	}
}

// TestInstrumentRecordsInFlightDuringRequest checks the gauge is raised while a
// request is in flight, which is what makes it usable as a saturation signal.
func TestInstrumentRecordsInFlightDuringRequest(t *testing.T) {
	m := New()

	var inFlight float64
	recording := m.Instrument(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		inFlight = testutil.ToFloat64(m.requestsInFlight)
	}))

	recording.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/projects", nil))

	if inFlight != 1 {
		t.Errorf("in-flight gauge inside the handler = %v, want 1", inFlight)
	}
	if after := testutil.ToFloat64(m.requestsInFlight); after != 0 {
		t.Errorf("in-flight gauge after the request = %v, want 0", after)
	}
}
