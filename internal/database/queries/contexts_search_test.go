package queries

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
)

// explodingQuerier fails the test if any query reaches it.
//
// The guarantee under test is that an empty query never becomes a database
// round trip, and the only way to observe that is to make any query an
// immediate failure. A fake that returns an empty result set would not prove
// anything: the code could have queried and still returned nothing.
type explodingQuerier struct {
	t *testing.T
}

func (q explodingQuerier) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	q.t.Helper()
	q.t.Fatal("ExecContext was called: the empty-query short-circuit did not hold")
	return nil, nil
}

func (q explodingQuerier) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	q.t.Helper()
	q.t.Fatal("QueryContext was called: the empty-query short-circuit did not hold")
	return nil, nil
}

func (q explodingQuerier) QueryRowContext(context.Context, string, ...any) *sql.Row {
	q.t.Helper()
	q.t.Fatal("QueryRowContext was called: the empty-query short-circuit did not hold")
	return nil
}

// TestSearchContextsEmptyQueryDoesNotQuery covers the property that matters most
// for an agent-facing tool: asking for nothing returns nothing.
//
// The alternative implementation — letting an empty query reach Postgres —
// returns every context in the project, which is precisely the read the search
// exists to avoid. An agent calling the tool without arguments would pull the
// whole project into its context window and pay for it twice.
func TestSearchContextsEmptyQueryDoesNotQuery(t *testing.T) {
	store := NewContextsStore(explodingQuerier{t: t})

	for _, query := range []string{"", "   ", "\t\n  "} {
		got, err := store.SearchContexts(context.Background(), uuid.New(), query, 10)
		if err != nil {
			t.Fatalf("SearchContexts(%q): %v", query, err)
		}
		if got == nil {
			t.Errorf("SearchContexts(%q) returned nil; callers range over the result and a nil slice is a trap", query)
		}
		if len(got) != 0 {
			t.Errorf("SearchContexts(%q) returned %d rows, want 0", query, len(got))
		}
	}
}

// TestSearchLimitConstants pins the bounds the store clamps to. The cap is what
// stops a single request from reproducing the problem search exists to solve,
// and a default above the cap would be a contradiction.
func TestSearchLimitConstants(t *testing.T) {
	if DefaultSearchLimit <= 0 {
		t.Errorf("DefaultSearchLimit = %d, want a positive default", DefaultSearchLimit)
	}
	if MaxSearchLimit < DefaultSearchLimit {
		t.Errorf("MaxSearchLimit = %d is below DefaultSearchLimit = %d",
			MaxSearchLimit, DefaultSearchLimit)
	}
}
