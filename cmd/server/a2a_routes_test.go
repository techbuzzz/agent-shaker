package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	a2aserver "github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/observability"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

// noAuth is a pass-through stand-in for the auth guards, which the route table
// composes unconditionally. A nil Middleware would be *called* while the chain
// is being built and panic there, before any of this is under test.
func noAuth(next http.Handler) http.Handler { return next }

// a2aTestMux builds the real route table from newServeMux with just enough
// wiring for the A2A surface. Anything not under test is left nil: a nil
// handler is still *registered*, so routing is still exercised, and a request
// that reaches one panics into the Recovery middleware (500) rather than
// silently 404-ing. That distinction is exactly what these tests assert on.
func a2aTestMux(t *testing.T) http.Handler {
	t.Helper()

	store := task.NewMemoryStore(t.TempDir())
	manager := task.NewManager(store, nil, "")

	d := routeDeps{
		a2aHandler:       a2aserver.NewA2AHandler(manager),
		streamingHandler: a2aserver.NewStreamingHandler(manager),
		artifactHandler:  a2aserver.NewArtifactHandler(a2aserver.NewDatabaseContextStorage(nil), ""),
		agentCardHandler: a2aserver.NewAgentCardHandler("test", "", false),
		obs:              observability.New(),
		auth:             noAuth,
		wsAuth:           noAuth,
		accessLog:        middleware.Logger,
	}

	mux, err := newServeMux(d)
	if err != nil {
		t.Fatalf("newServeMux: %v", err)
	}
	return mux
}

// TestA2APathsResolveThroughTheSubMux guards the http.StripPrefix in the A2A
// branch of the route table.
//
// Go's ServeMux matches a subtree pattern without rewriting the request, so
// the sub-mux saw "/a2a/v1/tasks" while its patterns are registered as
// "/tasks". Every A2A endpoint therefore 404'd — and the failure presented as
// an edge-routing problem rather than a routing-table bug, which is why it
// survived until the A2A surface was actually published.
//
// A 404 here means the prefix was not stripped. Anything else (200, or 500 from
// a nil handler deeper in) means the route matched.
func TestA2APathsResolveThroughTheSubMux(t *testing.T) {
	mux := a2aTestMux(t)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/a2a/v1/tasks"},
		{http.MethodGet, "/a2a/v1/tasks/does-not-exist"},
		{http.MethodDelete, "/a2a/v1/tasks/does-not-exist"},
		{http.MethodPost, "/a2a/v1/message"},
		{http.MethodPost, "/a2a/v1/message:stream"},
		{http.MethodGet, "/a2a/v1/artifacts"},
		{http.MethodGet, "/a2a/v1/artifacts/does-not-exist"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			body := strings.NewReader(`{"message":{"role":"user","content":"probe"}}`)

			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			// Status alone cannot tell the two 404s apart: a request for a
			// missing task or artifact is *supposed* to 404. What
			// distinguishes them is the writer. A routing miss falls through
			// to http.NotFound, which emits a plain-text "404 page not found";
			// a handler that matched always emits the JSON error envelope.
			if isRouterMiss(rec) {
				t.Fatalf("%s %s did not reach a handler: the /a2a/v1 prefix was not stripped before reaching the sub-mux (body: %q)",
					tc.method, tc.path, rec.Body.String())
			}
		})
	}
}

// isRouterMiss reports whether a 404 came from http.ServeMux failing to match a
// pattern rather than from a handler that matched and declined.
func isRouterMiss(rec *httptest.ResponseRecorder) bool {
	if rec.Code != http.StatusNotFound {
		return false
	}
	return strings.HasPrefix(rec.Body.String(), "404 page not found")
}

// TestA2AListTasksReturnsAPayload pins the happy path, so the test above cannot
// be satisfied by a routing change that merely stops 404-ing.
func TestA2AListTasksReturnsAPayload(t *testing.T) {
	mux := a2aTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var payload struct {
		Tasks []any `json:"tasks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, rec.Body.String())
	}
	if payload.Tasks == nil {
		t.Error(`response has no "tasks" key; an empty task list must serialise as [] not null`)
	}
}

// TestA2ARequiresAuthentication keeps the machine surface behind the API key.
// The reverse proxy at the edge relies on this rather than adding its own gate
// for these paths.
func TestA2ARequiresAuthentication(t *testing.T) {
	obs := observability.New()
	const key = "test-key-0123456789abcdef"

	authMW, err := middleware.RequireAPIKey(middleware.AuthConfig{
		Enabled: true,
		Keys:    []string{key},
	})
	if err != nil {
		t.Fatalf("RequireAPIKey: %v", err)
	}

	store := task.NewMemoryStore(t.TempDir())
	manager := task.NewManager(store, nil, "")

	mux, err := newServeMux(routeDeps{
		a2aHandler:       a2aserver.NewA2AHandler(manager),
		streamingHandler: a2aserver.NewStreamingHandler(manager),
		artifactHandler:  a2aserver.NewArtifactHandler(a2aserver.NewDatabaseContextStorage(nil), ""),
		agentCardHandler: a2aserver.NewAgentCardHandler("test", "", true),
		obs:              obs,
		auth:             authMW,
		wsAuth:           authMW,
		accessLog:        middleware.Logger,
	})
	if err != nil {
		t.Fatalf("newServeMux: %v", err)
	}

	tests := []struct {
		name       string
		credential string
		want       int
	}{
		{"no credential", "", http.StatusUnauthorized},
		{"valid bearer", "Bearer " + key, http.StatusOK},
		{"wrong bearer", "Bearer nope", http.StatusUnauthorized},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks", nil)
			if tc.credential != "" {
				req.Header.Set("Authorization", tc.credential)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}
