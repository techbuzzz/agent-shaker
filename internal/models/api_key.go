package models

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Scope is a permission a key may hold.
//
// The vocabulary is three words wide on purpose. A scope set that grows into a
// permission matrix is a role system wearing a disguise, and role systems need
// migration tooling that this repository does not have yet. If a fourth scope
// ever becomes necessary, that is a deliberate milestone — not a field addition.
type Scope string

const (
	// ScopeRead permits methods that do not change state: GET, HEAD, OPTIONS.
	ScopeRead Scope = "read"

	// ScopeWrite permits mutating methods: POST, PUT, PATCH, DELETE. It implies
	// ScopeRead, because a caller that may change a task it cannot see has been
	// given a blind write — it cannot learn which id to target, and the audit
	// journal would record changes nobody could have read.
	ScopeWrite Scope = "write"

	// ScopeAdmin permits key and principal management (/api/keys, /api/people).
	// It implies nothing: administering credentials is deliberately separable
	// from reading project data, so an operator can hold one without the other.
	ScopeAdmin Scope = "admin"

	// ScopeAll is the wildcard, equivalent to holding every scope. Stored
	// explicitly so a key minted with full access says so in the row instead of
	// being indistinguishable from a key whose scope list was forgotten.
	ScopeAll Scope = "*"
)

// AllScopes is every scope the system understands, in a stable order. Exported
// so the management UI and the API validator cannot drift apart.
var AllScopes = []Scope{ScopeRead, ScopeWrite, ScopeAdmin}

// ErrUnknownScope is returned by ParseScope for a value outside AllScopes
// (ScopeAll is accepted too).
var ErrUnknownScope = errors.New("unknown scope")

// ParseScope normalises a requested scope name.
func ParseScope(s string) (Scope, error) {
	candidate := Scope(strings.ToLower(strings.TrimSpace(s)))
	switch candidate {
	case ScopeRead, ScopeWrite, ScopeAdmin, ScopeAll:
		return candidate, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownScope, s)
	}
}

// ScopesFromStrings converts a scanned Postgres text array into typed scopes.
//
// Needed because pq.Array cannot scan into []Scope: it builds a scanner for
// []string and for types implementing sql.Scanner, and Scope is a bare string
// type implementing neither. Scanning directly fails at runtime with
// "pq: scanning to models.Scope is not implemented" — on the auth hot path,
// which is why an issued key could never actually authenticate.
//
// An unknown value is an error rather than a silent skip. Dropping it would
// quietly widen a key's authority, and a key row is exactly the place where a
// surprising value must be loud.
func ScopesFromStrings(in []string) ([]Scope, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]Scope, 0, len(in))
	for _, raw := range in {
		s, err := ParseScope(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// ScopeStrings renders scopes for a Postgres array parameter.
func ScopeStrings(in []Scope) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = string(s)
	}
	return out
}

// NormalizeScopes validates, de-duplicates and orders a requested scope set.
// Ordering matters: it makes the stored array deterministic, so re-issuing a
// key with the same scopes produces the same row and a diff of two keys is
// readable.
//
// An empty input is valid and means "unrestricted" — see Allows.
func NormalizeScopes(in []string) ([]Scope, error) {
	if len(in) == 0 {
		return nil, nil
	}

	seen := make(map[Scope]struct{}, len(in))
	out := make([]Scope, 0, len(in))
	for _, raw := range in {
		scope, err := ParseScope(raw)
		if err != nil {
			return nil, err
		}
		if _, dup := seen[scope]; dup {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	// A wildcard subsumes everything, so keeping the rest would imply a
	// constraint the row does not actually carry.
	if _, hasAll := seen[ScopeAll]; hasAll {
		return []Scope{ScopeAll}, nil
	}
	return out, nil
}

// Allows reports whether a granted scope set permits the requested scope.
//
// An empty grant is unrestricted. That is the single most consequential rule in
// this file, so it is stated three times: here, in the migration that defaults
// scopes to '{}', and in the docs. It is what makes the upgrade non-breaking —
// a key issued before scopes existed behaves exactly as it did — and it is also
// why minting a restricted key has to be an explicit act rather than a default.
func Allows(granted []Scope, want Scope) bool {
	if len(granted) == 0 {
		return true
	}
	if slices.Contains(granted, ScopeAll) {
		return true
	}
	if slices.Contains(granted, want) {
		return true
	}
	// write implies read
	return want == ScopeRead && slices.Contains(granted, ScopeWrite)
}

// ScopeForMethod maps an HTTP method to the scope it requires.
//
// Safe methods are reads by definition; everything else mutates something, and
// a method this function has never heard of is treated as a mutation. Failing
// closed matters more here than being clever: a new verb added to the codebase
// must not silently arrive with read access.
func ScopeForMethod(method string) Scope {
	switch strings.ToUpper(method) {
	case "GET", "HEAD", "OPTIONS":
		return ScopeRead
	default:
		return ScopeWrite
	}
}

// ErrAPIKeyNotFound is returned when no row matches a lookup.
//
// It lives here rather than in the store package because two very different
// layers have to agree on its identity: the store raises it, and the request
// path has to tell "this credential was never issued" from "the database is
// unreachable" without importing the persistence package. Conflating the two
// turns an outage into what looks like a wave of failed logins.
var ErrAPIKeyNotFound = errors.New("api key not found")

// The APIKey row, the mint request and the issued response live in
// tooltype.go, next to ToolType and the agent binding: `agent_id` on a key is
// the same concept as `tool_type` on an agent, and splitting them across files
// put one half of a single decision in each. The scope vocabulary below is
// orthogonal to identity and stays here.
