package models

import "testing"

// TestScopesFromStrings guards the auth hot path.
//
// The store used to scan the scopes array straight into []Scope with pq.Array,
// which cannot work: pq.Array only builds a scanner for []string and for types
// implementing sql.Scanner, and Scope implements neither. Every read of an
// api_keys row therefore failed at runtime with "pq: scanning to
// models.Scope is not implemented" — including GetAPIKeyByHash, so an issued
// key could never authenticate. Unit tests missed it because the store had no
// database behind it; only a live run did.
func TestScopesFromStrings(t *testing.T) {
	got, err := ScopesFromStrings([]string{"read", "write", "admin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d scopes, want 3", len(got))
	}
	for i, want := range []Scope{ScopeRead, ScopeWrite, ScopeAdmin} {
		if got[i] != want {
			t.Errorf("scope %d = %q, want %q", i, got[i], want)
		}
	}

	// Empty must stay nil, because nil and empty both mean "unrestricted" to
	// Allows but serialise differently.
	got, err = ScopesFromStrings(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("nil input should yield nil, got %v", got)
	}

	// An unknown value must be an error. Silently dropping it would quietly
	// widen a key's authority, and a key row is the last place to be lenient.
	if _, err := ScopesFromStrings([]string{"read", "superuser"}); err == nil {
		t.Fatal("an unknown scope was accepted; that silently drops a grant or widens one")
	}
}

// TestScopeStringsRoundTrips proves the write path and the read path agree.
func TestScopeStringsRoundTrips(t *testing.T) {
	original := []Scope{ScopeAdmin, ScopeRead}
	back, err := ScopesFromStrings(ScopeStrings(original))
	if err != nil {
		t.Fatalf("round trip failed: %v", err)
	}
	if len(back) != len(original) {
		t.Fatalf("round trip changed length: %d -> %d", len(original), len(back))
	}
	for i := range original {
		if back[i] != original[i] {
			t.Errorf("round trip changed scope %d: %q -> %q", i, original[i], back[i])
		}
	}
}
