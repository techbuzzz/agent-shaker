package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/authkey"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// presentedKey is a real, well-formed credential.
//
// Not a placeholder string: ResolveKey rejects anything that fails
// authkey.IsWellFormed *before* it consults the store, so a fake that is handed
// "presented" would never be called and the failure would read as a lookup
// problem rather than as the test data being wrong.
var presentedKey = func() string {
	secret, _, _, err := authkey.Generate()
	if err != nil {
		panic("authkey.Generate: " + err.Error())
	}
	return secret
}()

// fakeKeyStore is the slice of the api_keys table the resolver reads.
//
// It also records how many lookups happened, which is what lets the caching
// tests distinguish a cached decision from a fresh one.
type fakeKeyStore struct {
	key   *models.APIKey
	err   error
	reads int

	touched []uuid.UUID
}

func (f *fakeKeyStore) GetAPIKeyByHash(_ context.Context, _ string) (*models.APIKey, error) {
	f.reads++
	if f.err != nil {
		return nil, f.err
	}
	return f.key, nil
}

func (f *fakeKeyStore) TouchAPIKeyLastUsed(_ context.Context, id uuid.UUID) error {
	f.touched = append(f.touched, id)
	return nil
}

func boundKey(agentID *uuid.UUID) *models.APIKey {
	return &models.APIKey{
		ID: uuid.New(),
		// The digest has to be the one for the key the test presents: the
		// resolver re-compares it in constant time as defence against a store
		// that ignored its WHERE clause, and a mismatch there is reported as
		// "key not recognised" rather than as a hash mismatch — which is
		// exactly the kind of failure that sends you looking at the wrong
		// layer.
		KeyHash:       authkey.Hash(presentedKey),
		KeyPrefix:     "ask_abcdef",
		PrincipalID:   uuid.New(),
		PrincipalType: models.PersonKindService,
		Scopes:        []models.Scope{models.ScopeRead, models.ScopeWrite},
		AgentID:       agentID,
	}
}

// TestResolveKeyCarriesTheAgentBinding checks that the binding on the row reaches
// the principal, because that copy is the only place a key row becomes a request
// identity. If it were lost here, every downstream check would silently see an
// unbound credential and trust the query string again.
func TestResolveKeyCarriesTheAgentBinding(t *testing.T) {
	t.Parallel()

	agent := uuid.New()
	resolver := NewKeyResolver(&fakeKeyStore{key: boundKey(&agent)}, ResolverOptions{})

	principal, err := resolver.ResolveKey(context.Background(), presentedKey)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}

	got, bound := principal.AgentIdentity()
	if !bound {
		t.Fatal("AgentIdentity() reports no binding, want the key's agent")
	}
	if got != agent {
		t.Errorf("AgentIdentity() = %s, want %s", got, agent)
	}
}

// TestResolveKeyWithoutABinding is the compatibility case, and it is the one
// that matters most: every credential issued before migration 017 has a NULL
// agent_id, as do the legacy API_KEYS secrets. They must resolve to a principal
// that reports no binding, not to one pointing at the zero UUID.
func TestResolveKeyWithoutABinding(t *testing.T) {
	t.Parallel()

	resolver := NewKeyResolver(&fakeKeyStore{key: boundKey(nil)}, ResolverOptions{})

	principal, err := resolver.ResolveKey(context.Background(), presentedKey)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}

	if got, bound := principal.AgentIdentity(); bound {
		t.Errorf("AgentIdentity() = %s, bound=%v; want no binding for an unbound key", got, bound)
	}
	if principal.AgentID != nil {
		t.Errorf("AgentID = %v, want nil so the MCP layer keeps its legacy behaviour", principal.AgentID)
	}
}

// TestResolveKeyCopiesTheBinding guards a subtler failure: the cached principal
// outlives the row it was built from, so a pointer aliased into that row could
// be mutated by a later lookup and change the identity of an in-flight request.
func TestResolveKeyCopiesTheBinding(t *testing.T) {
	t.Parallel()

	agent := uuid.New()
	// The row holds its own copy, deliberately. Passing &agent would make the
	// store and the assertion alias the same variable, so the "mutation" below
	// would rewrite the expected value as well and the test would pass against
	// a resolver that aliased pointers.
	rowAgent := agent
	store := &fakeKeyStore{key: boundKey(&rowAgent)}
	resolver := NewKeyResolver(store, ResolverOptions{})

	principal, err := resolver.ResolveKey(context.Background(), presentedKey)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}

	// Simulate the row being reused for a different agent, which is what an
	// aliased pointer would follow.
	*store.key.AgentID = uuid.New()

	got, _ := principal.AgentIdentity()
	if got != agent {
		t.Errorf("AgentIdentity() = %s after the row changed, want the original %s", got, agent)
	}
}

// TestCachedPrincipalKeepsItsBinding pins the interaction with the cache. The
// agent_id on a key is immutable — no endpoint re-binds one — so a cached
// principal is never stale in this field, and no extra invalidation is needed.
func TestCachedPrincipalKeepsItsBinding(t *testing.T) {
	t.Parallel()

	agent := uuid.New()
	store := &fakeKeyStore{key: boundKey(&agent)}
	resolver := NewKeyResolver(store, ResolverOptions{})

	first, err := resolver.ResolveKey(context.Background(), presentedKey)
	if err != nil {
		t.Fatalf("first ResolveKey: %v", err)
	}
	second, err := resolver.ResolveKey(context.Background(), presentedKey)
	if err != nil {
		t.Fatalf("second ResolveKey: %v", err)
	}

	if store.reads != 1 {
		t.Errorf("store reads = %d, want 1 — the second lookup should have been cached", store.reads)
	}
	if first.AgentID == nil || second.AgentID == nil {
		t.Fatal("the cached principal lost its binding")
	}
	if *first.AgentID != *second.AgentID {
		t.Errorf("cached principal agent %s differs from the original %s", *second.AgentID, *first.AgentID)
	}

	// And exactly one last-used stamp for two authentications: it is written per
	// cache lifetime, not per request.
	if len(store.touched) != 1 {
		t.Errorf("last-used stamps = %d, want 1", len(store.touched))
	}
}

// TestRevokingInvalidatesTheCachedBinding matters operationally. A revoked
// agent key that keeps resolving for up to the cache TTL is a credential that
// outlives the operator's decision to withdraw it, and it still carries an
// agent binding — so it can act as that agent for the remainder of the TTL.
func TestRevokingInvalidatesTheCachedBinding(t *testing.T) {
	t.Parallel()

	agent := uuid.New()
	store := &fakeKeyStore{key: boundKey(&agent)}
	resolver := NewKeyResolver(store, ResolverOptions{})

	if _, err := resolver.ResolveKey(context.Background(), presentedKey); err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}

	resolver.InvalidateKey(authkey.Hash(presentedKey))

	if _, err := resolver.ResolveKey(context.Background(), presentedKey); err != nil {
		t.Fatalf("ResolveKey after invalidation: %v", err)
	}
	if store.reads != 2 {
		t.Errorf("store reads = %d, want 2 — invalidation must force a fresh lookup", store.reads)
	}
}
