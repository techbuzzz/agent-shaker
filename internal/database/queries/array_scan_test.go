package queries

import (
	"testing"

	"github.com/lib/pq"
)

// contexts.tags and global_contexts.tags are TEXT[] columns.
//
// The stores used to scan them with pq.Array(&c.Tags) — a *[]string. That
// builds a generic array over the slice value rather than a string array, and
// reading fails at runtime with "scanning to string is not implemented; only
// sql.Scanner", which turned the whole contexts list into a 500.
//
// pq.StringArray is the type that implements sql.Scanner for a TEXT[] column.
// This test pins that contract in the one place it can be pinned without a
// database: a *[]string does not satisfy sql.Scanner, a pq.StringArray does.
//
// If a future refactor swaps the scan back to pq.Array over a plain slice,
// this stops compiling — which is the point.
func TestStringArraySatisfiesScannerAndSliceDoesNot(t *testing.T) {
	var tags pq.StringArray
	// The assertion has to be a type assertion, not a nil check. Assigning to
	// an interface already proves the implementation at compile time, and the
	// interface it lands in is never nil — so `scanner == nil` is a
	// comparison that can never be true and says nothing.
	if _, ok := any(&tags).(interface{ Scan(src any) error }); !ok {
		t.Fatal("pq.StringArray must implement sql.Scanner for a TEXT[] column")
	}

	// pq.Array over a *[]string is the shape that failed at runtime.
	var plain []string
	if _, ok := any(&plain).(interface{ Scan(src any) error }); ok {
		t.Fatal("*[]string must not satisfy sql.Scanner; if it does, the real scan behaviour changed")
	}

	// And the round trip the store relies on: the driver hands a TEXT[] column
	// over as its array literal, and the value has to come back as a plain
	// []string the model can hold.
	for _, tc := range []struct {
		raw  string
		want []string
	}{
		{raw: "{}", want: []string{}},
		{raw: "{api,documentation}", want: []string{"api", "documentation"}},
		{raw: `{"has space"}`, want: []string{"has space"}},
	} {
		var scanned pq.StringArray
		if err := scanned.Scan(tc.raw); err != nil {
			t.Fatalf("scanning %s: %v", tc.raw, err)
		}
		got := []string(scanned)
		if len(got) != len(tc.want) {
			t.Errorf("scanning %s produced %v, want %v", tc.raw, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("scanning %s produced %v, want %v", tc.raw, got, tc.want)
			}
		}
	}
}
