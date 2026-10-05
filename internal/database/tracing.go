package database

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracerName identifies this instrumentation. It is deliberately not one of the
// gen_ai.* namespaces: every key in that space was still at Development
// stability, so names taken from it could change without notice.
const TracerName = "github.com/techbuzzz/agent-shaker/internal/database"

// queryVerbPattern finds the statement class anywhere in the query, not just at
// the start, so a common table expression is classified by what it ultimately
// does rather than by its opening WITH.
var queryVerbPattern = regexp.MustCompile(`(?is)\b(SELECT|INSERT|UPDATE|DELETE)\b`)

// Table extraction is ordered by intent rather than by first match. A query
// like "WITH recent AS (SELECT ... FROM tasks) INSERT INTO milestones ..."
// mentions three relations, and the useful one is the one being written: the
// read that feeds a write is a detail. Taking the first FROM would label an
// insert into milestones as a read of tasks, which sends an operator to the
// wrong table when a write is slow.
var queryTablePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)\bINTO\s+"?([a-z_][a-z0-9_]*)"?`),
	regexp.MustCompile(`(?is)\bUPDATE\s+"?([a-z_][a-z0-9_]*)"?`),
	regexp.MustCompile(`(?is)\bFROM\s+"?([a-z_][a-z0-9_]*)"?`),
}

// classifyQuery reduces a statement to a bounded (verb, table) pair.
//
// The verb is normalised to upper case and the table to lower case, so the
// result does not depend on how the statement happened to be formatted in the
// source. A span named db.Update in one query and db.update in the next splits
// a single logical operation across two time series. Both are empty when the
// statement cannot be parsed, which is preferable to guessing.
//
// The statement text and any bound parameter are deliberately not captured:
// both are unbounded-cardinality values, and parameters can carry customer
// content.
func classifyQuery(query string) (verb, table string) {
	if m := queryVerbPattern.FindStringSubmatch(query); m != nil {
		verb = strings.ToUpper(m[1])
	}
	for _, pattern := range queryTablePatterns {
		if m := pattern.FindStringSubmatch(query); m != nil {
			table = strings.ToLower(m[1])
			break
		}
	}
	return verb, table
}

// tracedQuerier decorates a Querier with one span per statement. Decorating the
// interface once at the wiring point instruments every query in the application
// without touching any of the call sites, and keeps the raw SQL out of the
// span attributes.
type tracedQuerier struct {
	inner  Querier
	tracer trace.Tracer
}

// Traced returns a Querier that records a span per statement. Pass the result
// wherever a Querier is expected; when tracing is not configured the global
// provider is a no-op and this costs one function call per statement.
func Traced(q Querier) Querier {
	return &tracedQuerier{inner: q, tracer: otel.Tracer(TracerName)}
}

// startQuerySpan opens the span and returns a finish function that records the
// outcome. Errors are both recorded on the span and set as its status, so a
// failed query is visible whether the backend is a trace viewer or a log.
func (t *tracedQuerier) startQuerySpan(ctx context.Context, query string) (context.Context, func(error)) {
	verb, table := classifyQuery(query)

	name := "db.query"
	if verb != "" {
		name = "db." + strings.ToLower(verb)
	}

	attrs := []attribute.KeyValue{attribute.String("db.system", "postgresql")}
	if table != "" {
		attrs = append(attrs, attribute.String("shaker.db.table", table))
	}
	if verb != "" {
		attrs = append(attrs, attribute.String("shaker.db.operation", verb))
	}

	ctx, span := t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))

	return ctx, func(err error) {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}
}

func (t *tracedQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	ctx, done := t.startQuerySpan(ctx, query)
	result, err := t.inner.ExecContext(ctx, query, args...)
	done(err)
	return result, err
}

func (t *tracedQuerier) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	// The span closes when the call returns. A fuller measurement would need
	// the span to outlive the call, but Querier hands back the concrete
	// *sql.Rows rather than an interface, so there is no place to hang a
	// close hook without changing the interface every store depends on. What is
	// measured is dispatch plus server response, not the drain of a large
	// result set. Reads that matter for latency use QueryRowContext anyway.
	ctx, done := t.startQuerySpan(ctx, query)
	rows, err := t.inner.QueryContext(ctx, query, args...)
	done(err)
	return rows, err
}

func (t *tracedQuerier) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	// database/sql reports a deferred query failure only from Row.Scan, which
	// happens outside this call, so the span is closed here. A statement that
	// fails on the round trip still shows the error; one that fails while
	// streaming its single row does not.
	ctx, done := t.startQuerySpan(ctx, query)
	row := t.inner.QueryRowContext(ctx, query, args...)
	done(nil)
	return row
}

func (t *tracedQuerier) PingContext(ctx context.Context) error {
	ctx, span := t.tracer.Start(ctx, "db.ping")
	err := t.inner.PingContext(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
	return err
}

func (t *tracedQuerier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	ctx, span := t.tracer.Start(ctx, "db.transaction")
	tx, err := t.inner.BeginTx(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
	return tx, err
}
