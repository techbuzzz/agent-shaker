package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/authkey"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// Default resolver tuning.
//
// The TTL is the single most security-relevant number in this file: it is how
// long a revoked credential keeps working after revocation, because a
// revocation that has to wait for a cache entry to expire is not a revocation
// for those 15 seconds. 15 s is short enough that an operator who revokes a
// leaked key has lost nothing they still need, and long enough to keep the
// hot path off the database.
const (
	DefaultKeyTTL         = 15 * time.Second
	DefaultNegativeKeyTTL = 5 * time.Second
	DefaultKeyCacheMax    = 4096
)

// KeyResolver turns a presented credential into a Principal.
//
// An interface rather than a concrete type so the auth middleware can be tested
// without a database, and so the flat API_KEYS path stays available when the
// server runs without one.
type KeyResolver interface {
	// ResolveKey maps a presented credential to its principal. The returned
	// error must wrap ErrKeyUnknown or ErrKeyRevoked for a credential decision;
	// anything else is treated as an infrastructure failure, and answering 401
	// to it would disguise an outage as an attack.
	ResolveKey(ctx context.Context, presented string) (Principal, error)

	// InvalidateKey drops a cached decision for the given digest. Called after
	// a revoke so the revocation is effective on the very next request in this
	// process rather than at the end of the TTL.
	InvalidateKey(digest string)
}

var (
	// ErrKeyUnknown is returned for a credential that was never issued.
	ErrKeyUnknown = errors.New("api key is not recognised")

	// ErrKeyRevoked is returned for a credential that was issued and withdrawn.
	ErrKeyRevoked = errors.New("api key has been revoked")

	// ErrKeyStoreUnavailable marks an infrastructure failure behind the
	// credential decision. It is distinct from a rejected credential on
	// purpose: the first is an outage the caller cannot fix, the second is a
	// verdict on the caller, and answering 401 to the first both lies and
	// points every operator at the wrong place.
	ErrKeyStoreUnavailable = errors.New("api key store is unavailable")
)

// KeyStore is the slice of the api_keys table the resolver depends on.
// *queries.APIKeysStore satisfies it; the interface keeps internal/middleware
// free of the persistence layer.
type KeyStore interface {
	GetAPIKeyByHash(ctx context.Context, digest string) (*models.APIKey, error)
	TouchAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error
}

// ResolverOptions tunes NewKeyResolver. The zero value is valid and yields the
// defaults above.
type ResolverOptions struct {
	TTL         time.Duration
	NegativeTTL time.Duration
	MaxEntries  int

	// Now is injectable so the cache can be tested without sleeping.
	Now func() time.Time

	// Logger receives cache-boundary and last-used-stamp failures. Defaults to
	// slog.Default.
	Logger *slog.Logger
}

// StoreKeyResolver resolves credentials against the database, with a
// short-lived cache in front.
type StoreKeyResolver struct {
	store      KeyStore
	ttl        time.Duration
	negativeTTL time.Duration
	now        func() time.Time
	logger     *slog.Logger
	cache      *resolverCache
}

// NewKeyResolver returns a resolver backed by store.
func NewKeyResolver(store KeyStore, opts ResolverOptions) *StoreKeyResolver {
	if opts.TTL <= 0 {
		opts.TTL = DefaultKeyTTL
	}
	if opts.NegativeTTL <= 0 {
		opts.NegativeTTL = DefaultNegativeKeyTTL
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = DefaultKeyCacheMax
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &StoreKeyResolver{
		store:       store,
		ttl:         opts.TTL,
		negativeTTL: opts.NegativeTTL,
		now:         now,
		logger:      logger,
		cache:       newResolverCache(opts.MaxEntries, now),
	}
}

// ResolveKey maps a presented credential to a principal.
func (r *StoreKeyResolver) ResolveKey(ctx context.Context, presented string) (Principal, error) {
	secret := strings.TrimSpace(presented)
	if secret == "" {
		return Principal{}, ErrKeyUnknown
	}

	// Cheap rejection before any work: a credential without our prefix was not
	// minted here, and the legacy flat keys never reach this resolver. Skipping
	// the database for those turns a flood of junk tokens into a CPU cost
	// rather than a query-per-request cost.
	if !authkey.IsWellFormed(secret) {
		return Principal{}, ErrKeyUnknown
	}

	digest := authkey.Hash(secret)
	if p, err, ok := r.cache.get(digest); ok {
		return p, err
	}

	key, err := r.store.GetAPIKeyByHash(ctx, digest)
	switch {
	case errors.Is(err, models.ErrAPIKeyNotFound):
		// Cached, because the cost this protects against is a flood of distinct
		// wrong keys, each of which would otherwise be its own index probe.
		r.cache.put(digest, Principal{}, ErrKeyUnknown, r.negativeTTL)
		return Principal{}, ErrKeyUnknown
	case err != nil:
		// Deliberately not cached: an outage must not be remembered as a
		// permanent verdict, and the next request should retry the database.
		// Marked so the caller answers 503 rather than 401 — an outage that
		// looks like a credential problem sends every operator hunting for an
		// attack that never happened.
		return Principal{}, fmt.Errorf("%w: resolve api key: %w", ErrKeyStoreUnavailable, err)
	}

	// Defence in depth. The authoritative check is the indexed lookup above;
	// this re-compares the digest in constant time so that a store which
	// ignored its WHERE clause could not turn into an authentication bypass.
	if subtle.ConstantTimeCompare([]byte(key.KeyHash), []byte(digest)) != 1 {
		r.logger.ErrorContext(ctx, "api key digest mismatch; store returned a different row",
			"key_id", key.ID.String(),
		)
		return Principal{}, ErrKeyUnknown
	}

	if !key.Active() {
		r.cache.put(digest, Principal{}, ErrKeyRevoked, r.ttl)
		return Principal{}, ErrKeyRevoked
	}

	principal := Principal{
		Type:        key.PrincipalType,
		ID:          key.PrincipalID,
		DisplayName: principalName(key),
		Scopes:      key.Scopes,
		ProjectID:   key.ProjectID,
	}
	r.cache.put(digest, principal, nil, r.ttl)

	// Last-used stamping happens once per cache lifetime, not once per request:
	// a write on the request path would make authentication the most
	// write-heavy operation in the process. The timestamp is allowed to be a
	// few seconds stale — it answers "when was this last used", not "is this
	// in use right now".
	if err := r.store.TouchAPIKeyLastUsed(ctx, key.ID); err != nil {
		r.logger.WarnContext(ctx, "could not stamp api key last_used_at",
			"key_id", key.ID.String(),
			"error", err,
		)
	}

	return principal, nil
}

// InvalidateKey drops the cached decision for a digest.
func (r *StoreKeyResolver) InvalidateKey(digest string) { r.cache.delete(digest) }

// principalName picks the best label available on a key row. The list query
// joins people for display; the lookup path deliberately does not, so a
// hot-path resolution falls back to the principal type.
func principalName(key *models.APIKey) string {
	if key.PrincipalName != "" {
		return key.PrincipalName
	}
	if key.PrincipalType == "" {
		return "unknown principal"
	}
	return string(key.PrincipalType) + " principal"
}

// resolverCache is a bounded, TTL'd map of digest to decision.
//
// It holds negative decisions too. That is the point: the expensive case under
// attack is a stream of distinct wrong keys, and caching "unknown" turns it
// from N database probes into at most N probes per negative-TTL window.
type resolverCache struct {
	mu      sync.RWMutex
	entries map[string]resolverEntry
	max     int
	now     func() time.Time
}

type resolverEntry struct {
	principal Principal
	err       error
	expiresAt time.Time
}

func newResolverCache(max int, now func() time.Time) *resolverCache {
	return &resolverCache{
		entries: make(map[string]resolverEntry, 64),
		max:     max,
		now:     now,
	}
}

// get returns a fresh decision. A stale entry is treated as a miss and is not
// removed here — get runs on the read path under a shared lock, and the
// subsequent put overwrites it. Sweeping happens on write, where the exclusive
// lock is already held.
func (c *resolverCache) get(digest string) (Principal, error, bool) {
	c.mu.RLock()
	entry, ok := c.entries[digest]
	c.mu.RUnlock()

	if !ok || !c.now().Before(entry.expiresAt) {
		return Principal{}, nil, false
	}
	return entry.principal, entry.err, true
}

func (c *resolverCache) put(digest string, principal Principal, err error, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.max {
		c.evictLocked()
	}
	c.entries[digest] = resolverEntry{
		principal: principal,
		err:       err,
		expiresAt: c.now().Add(ttl),
	}
}

func (c *resolverCache) delete(digest string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, digest)
}

// evictLocked makes room for one entry.
//
// Expired entries go first; if the cache is still full, entries are dropped in
// map order until there is headroom. That is arbitrary eviction, which is the
// right trade for a cache with no recency signal to work from: adding an LRU
// list would buy a marginally better hit rate and a lock held across every
// request.
func (c *resolverCache) evictLocked() {
	now := c.now()
	for k, v := range c.entries {
		if !now.Before(v.expiresAt) {
			delete(c.entries, k)
		}
	}
	// Clear a quarter of the capacity rather than a single slot, so a cache
	// running at its limit does not evict on every single insert.
	target := c.max - c.max/4
	for len(c.entries) > target {
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
}
