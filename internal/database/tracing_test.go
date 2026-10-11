package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestClassifyQuery pins the (verb, table) reduction. Both values become span
// name and attribute, so a wrong value does not merely mislabel a trace: an
// unparsed table would fall back to the empty string, and a regex that matched
// a column name would attribute a read to the wrong relation.
func TestClassifyQuery(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantVerb  string
		wantTable string
	}{
		{
			name:      "select from",
			query:     "SELECT id, title FROM tasks WHERE project_id = $1",
			wantVerb:  "SELECT",
			wantTable: "tasks",
		},
		{
			name:      "insert into",
			query:     "INSERT INTO contexts (project_id, title) VALUES ($1, $2)",
			wantVerb:  "INSERT",
			wantTable: "contexts",
		},
		{
			name:      "update",
			query:     "UPDATE tasks SET status = $1 WHERE id = $2",
			wantVerb:  "UPDATE",
			wantTable: "tasks",
		},
		{
			name:      "delete from",
			query:     "DELETE FROM daily_standups WHERE id = $1",
			wantVerb:  "DELETE",
			wantTable: "daily_standups",
		},
		{
			name: "common table expression is attributed to the written table",
			// A WITH query that ends in an INSERT mentions three relations.
			// The write target is the one an operator needs: labelling this
			// as a read of the CTE source would send them to the wrong table
			// when a write is slow.
			query: `WITH recent AS (
				SELECT id FROM tasks WHERE project_id = $1
			)
			INSERT INTO milestones (task_id) SELECT id FROM recent`,
			wantVerb:  "SELECT",
			wantTable: "milestones",
		},
		{
			name:      "quoted table name",
			query:     `SELECT * FROM "global_contexts" WHERE id = $1`,
			wantVerb:  "SELECT",
			wantTable: "global_contexts",
		},
		{
			name:      "lowercase statement",
			query:     "select id from agents where id = $1",
			wantVerb:  "SELECT",
			wantTable: "agents",
		},
		{
			name:      "unparseable statement yields no labels rather than a guess",
			query:     "SET timezone = 'UTC'",
			wantVerb:  "",
			wantTable: "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			verb, table := classifyQuery(tc.query)
			if verb != tc.wantVerb {
				t.Errorf("verb = %q, want %q", verb, tc.wantVerb)
			}
			if table != tc.wantTable {
				t.Errorf("table = %q, want %q", table, tc.wantTable)
			}
		})
	}
}

// fakeQuerier records the calls made against it and returns a canned outcome.
type fakeQuerier struct {
	execErr  error
	queryErr error
	pingErr  error
	beginErr error
	calls    int
}

func (f *fakeQuerier) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	f.calls++
	if f.execErr != nil {
		return nil, f.execErr
	}
	return driverResult{}, nil
}

func (f *fakeQuerier) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	f.calls++
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return nil, errors.New("fake: no rows available")
}

func (f *fakeQuerier) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }

func (f *fakeQuerier) PingContext(context.Context) error {
	f.calls++
	return f.pingErr
}

func (f *fakeQuerier) BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error) {
	f.calls++
	return nil, f.beginErr
}

type driverResult struct{}

func (driverResult) LastInsertId() (int64, error) { return 0, nil }
func (driverResult) RowsAffected() (int64, error) { return 1, nil }

// newRecordingTracer installs a TracerProvider backed by an in-memory exporter
// for the duration of the test and returns the recorded spans.
func newRecordingTracer(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})

	return recorder
}

// TestTracedRecordsSpanForSuccessfulStatement checks the happy path: a span
// named after the statement class, with no statement text in its attributes.
func TestTracedRecordsSpanForSuccessfulStatement(t *testing.T) {
	recorder := newRecordingTracer(t)
	fake := &fakeQuerier{}

	traced := Traced(fake)
	if _, err := traced.ExecContext(context.Background(),
		"UPDATE tasks SET status = $1 WHERE id = $2", "completed"); err != nil {
		t.Fatalf("ExecContext: %v", err)
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded %d spans, want 1", len(spans))
	}

	span := spans[0]
	if span.Name() != "db.update" {
		t.Errorf("span name = %q, want %q", span.Name(), "db.update")
	}

	attrs := map[string]string{}
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = kv.Value.String()
	}
	if attrs["shaker.db.table"] != "tasks" {
		t.Errorf("shaker.db.table = %q, want %q", attrs["shaker.db.table"], "tasks")
	}
	if attrs["shaker.db.operation"] != "UPDATE" {
		t.Errorf("shaker.db.operation = %q, want %q", attrs["shaker.db.operation"], "UPDATE")
	}
	if attrs["db.system"] != "postgresql" {
		t.Errorf("db.system = %q, want %q", attrs["db.system"], "postgresql")
	}

	// The whole point of classifying instead of recording: the statement text
	// and the bound parameter must never reach a span.
	for key := range attrs {
		if key == "db.statement" || key == "db.query.text" {
			t.Errorf("span carries statement text under %q", key)
		}
	}
	if _, leaked := attrs["completed"]; leaked {
		t.Error("span carries a bound parameter value")
	}
}

// TestTracedRecordsErrorOnSpan covers the failure path. An error that is only
// returned to the caller leaves no trace of itself in the telemetry, which is
// how a database outage ends up looking like a quiet period.
func TestTracedRecordsErrorOnSpan(t *testing.T) {
	recorder := newRecordingTracer(t)
	sentinel := errors.New("connection refused")
	traced := Traced(&fakeQuerier{execErr: sentinel})

	if _, err := traced.ExecContext(context.Background(), "SELECT 1"); !errors.Is(err, sentinel) {
		t.Fatalf("ExecContext error = %v, want %v", err, sentinel)
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("recorded %d spans, want 1", len(spans))
	}
	if spans[0].Status().Code.String() != "Error" {
		t.Errorf("span status = %v, want Error", spans[0].Status().Code)
	}
	if len(spans[0].Events()) == 0 {
		t.Error("span has no recorded error event")
	}
}

// TestTracedPropagatesParentContext checks the span joins the caller's trace
// instead of starting an unrelated one. Without this, every query would appear
// as its own root and the HTTP span would have no children.
func TestTracedPropagatesParentContext(t *testing.T) {
	recorder := newRecordingTracer(t)
	traced := Traced(&fakeQuerier{})

	parentCtx, parent := otel.Tracer("test").Start(context.Background(), "parent")
	_, err := traced.ExecContext(parentCtx, "DELETE FROM projects WHERE id = $1")
	if err != nil {
		t.Fatalf("ExecContext: %v", err)
	}
	parent.End()

	spans := recorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("recorded %d spans, want 2 (parent and child)", len(spans))
	}

	var child sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == "db.delete" {
			child = s
		}
	}
	if child == nil {
		t.Fatal("no span named db.delete was recorded")
	}
	t.Logf("parent: trace=%s span=%s", parent.SpanContext().TraceID(), parent.SpanContext().SpanID())
	t.Logf("child : trace=%s span=%s parent=%s", child.SpanContext().TraceID(),
		child.SpanContext().SpanID(), child.Parent().SpanID())
	if child.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Errorf("query span parent = %s, want the caller's span %s",
			child.Parent().SpanID(), parent.SpanContext().SpanID())
	}
	if child.SpanContext().TraceID() != parent.SpanContext().TraceID() {
		t.Error("query span is in a different trace from its caller")
	}
}

// TestTracedPingAndTransaction covers the two lifecycle calls that are easy to
// leave uninstrumented and are exactly the ones that reveal a saturated pool.
func TestTracedPingAndTransaction(t *testing.T) {
	recorder := newRecordingTracer(t)
	traced := Traced(&fakeQuerier{})

	if err := traced.PingContext(context.Background()); err != nil {
		t.Fatalf("PingContext: %v", err)
	}
	// BeginTx fails in the fake, which is fine: the point is that both calls
	// open a span and surface the error on it.
	_, _ = traced.BeginTx(context.Background(), nil)

	names := map[string]bool{}
	for _, s := range recorder.Ended() {
		names[s.Name()] = true
	}
	for _, want := range []string{"db.ping", "db.transaction"} {
		if !names[want] {
			t.Errorf("no span named %q was recorded; got %v", want, names)
		}
	}
}
