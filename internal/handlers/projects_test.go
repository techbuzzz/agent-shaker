package handlers_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/handlers"
)

// fakeQuerier is an in-memory implementation of queries.Querier used by
// the handler unit tests. It records calls so tests can assert on call
// counts without requiring a running Postgres.
//
// Note: database/sql.Row is a concrete struct whose Scan panics on an
// empty Row. This fake therefore only supports operations that do not
// invoke Scan on a fresh Row. Tests that need to exercise Scan-based
// reads should use a real database (or sqlmock).
type fakeQuerier struct {
	mu sync.Mutex

	createErr error
	listErr   error

	createCalls int
	listCalls   int
}

func newFakeQuerier() *fakeQuerier { return &fakeQuerier{} }

func (f *fakeQuerier) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	return fakeResult{}, nil
}

func (f *fakeQuerier) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	// Empty result set; ListProjects handler normalises this to [].
	return &sql.Rows{}, nil
}

func (f *fakeQuerier) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	// Return an empty Row. Tests must not invoke Scan on this — see note above.
	return &sql.Row{}
}

// fakeResult is a no-op sql.Result used for INSERT success.
type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 1, nil }

// Compile-time assertion that the fake satisfies the consumer-side
// interface. If the interface grows, this file fails to compile first.
var _ queries.Querier = (*fakeQuerier)(nil)

func TestProjectHandler_CreateProject_HappyPath(t *testing.T) {
	q := newFakeQuerier()
	h := handlers.NewProjectHandler(queries.NewProjectsStore(q), nil)

	body := `{"name":"Acme","description":"first project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d want %d", rec.Code, http.StatusCreated)
	}
	if q.createCalls != 1 {
		t.Errorf("CreateProject should have been called once, got %d", q.createCalls)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q want application/json", ct)
	}
}

func TestProjectHandler_CreateProject_InvalidBody(t *testing.T) {
	q := newFakeQuerier()
	h := handlers.NewProjectHandler(queries.NewProjectsStore(q), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader("not-json"))
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if q.createCalls != 0 {
		t.Errorf("CreateProject must not be called for invalid body, got %d calls", q.createCalls)
	}
}

func TestProjectHandler_CreateProject_ValidationFailure(t *testing.T) {
	q := newFakeQuerier()
	h := handlers.NewProjectHandler(queries.NewProjectsStore(q), nil)

	// Empty name — validator should reject before the DB is touched.
	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if q.createCalls != 0 {
		t.Errorf("validation must short-circuit the DB call, got %d", q.createCalls)
	}
}

func TestProjectHandler_DeleteProject_InvalidID(t *testing.T) {
	q := newFakeQuerier()
	h := handlers.NewProjectHandler(queries.NewProjectsStore(q), nil)

	// muxVars() reads r.PathValue("id"); with no routing match the value
	// is "" so uuid.Parse fails — exactly the path we want to exercise.
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/", nil)
	rec := httptest.NewRecorder()

	h.DeleteProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
}

// errDB is a sentinel error used to assert error propagation.
var errDB = errors.New("db exploded")

func TestProjectHandler_CreateProject_DBError(t *testing.T) {
	q := newFakeQuerier()
	q.createErr = errDB
	h := handlers.NewProjectHandler(queries.NewProjectsStore(q), nil)

	body := `{"name":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status: got %d want 500", rec.Code)
	}
}

// _ keeps the uuid import live even when test cases don't reference it.
var _ = uuid.Nil

// TestProjectHandler_NoDatabaseAvailable verifies that every handler method
// short-circuits with a structured 503 when the store has no underlying
// Querier (server running without Postgres). Without this guard, each
// request would panic; the Recovery middleware would catch the panic and
// return 500, hiding the real cause from clients.
func TestProjectHandler_NoDatabaseAvailable(t *testing.T) {
	q := newFakeQuerier() // non-nil interface value, but no real connection
	h := handlers.NewProjectHandler(queries.NewProjectsStore(nil), nil)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list", http.MethodGet, "/api/projects", ""},
		{"create", http.MethodPost, "/api/projects", `{"name":"x"}`},
		{"delete", http.MethodDelete, "/api/projects/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			rec := httptest.NewRecorder()

			switch tt.name {
			case "list":
				h.ListProjects(rec, req)
			case "create":
				h.CreateProject(rec, req)
			case "delete":
				h.DeleteProject(rec, req)
			}

			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("status: got %d want %d", rec.Code, http.StatusServiceUnavailable)
			}
			if q.createCalls != 0 {
				t.Errorf("handler must not reach the DB when store is nil, got %d calls", q.createCalls)
			}
		})
	}
}
