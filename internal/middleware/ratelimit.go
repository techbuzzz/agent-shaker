package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitConfig configures a per-IP token-bucket rate limiter.
type RateLimitConfig struct {
	// Limit is the sustained refill rate in tokens per second.
	Limit rate.Limit
	// Burst is the bucket size.
	Burst int
	// Skip returns true when a request should bypass the limiter entirely
	// (e.g. /ws, /healthz, /readyz, /metrics).
	Skip func(*http.Request) bool
	// MaxEntries is the upper bound on tracked IPs; the eviction loop will
	// drop the oldest entries when this is exceeded. Default 100_000.
	MaxEntries int
	// IdleEvictAfter is how long an IP may be unused before its entry is
	// dropped. Default 30 minutes.
	IdleEvictAfter time.Duration
	// TrustedProxy, when true, reads the (single) X-Forwarded-For entry as
	// the source IP. Off by default because trusting client-supplied headers
	// is a known security anti-pattern.
	TrustedProxy bool
}

// ipEntry is one row of the per-IP limiter store.
//
// lastSeen is guarded by mu. sync.Map makes the *map* safe, but the value is a
// shared *ipEntry, and every request for the same IP mutates lastSeen
// concurrently — so the field needs its own lock. It is a time.Time rather
// than an int64 so the monotonic reading survives, which is what keeps the
// eviction comparison correct across a wall-clock adjustment.
//
// The race was not theoretical: `go test -race` reported concurrent unsynchronised
// writes to this field from two in-flight requests, and a torn time.Time read
// can make now.Sub(lastSeen) jump by centuries, evicting IPs that are actively
// being served.
type ipEntry struct {
	lim *rate.Limiter

	mu       sync.Mutex
	lastSeen time.Time
}

// touch records activity for this IP.
func (e *ipEntry) touch(t time.Time) {
	e.mu.Lock()
	e.lastSeen = t
	e.mu.Unlock()
}

// seen returns the last recorded activity. A copy, not the field: time.Time
// carries an internal pointer to its monotonic source, and handing that out
// without the lock would reintroduce the race one level up.
func (e *ipEntry) seen() time.Time {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.lastSeen
}

// RateLimit returns a Middleware that enforces a per-IP token bucket plus a
// shutdown function that stops the eviction loop. Always call the shutdown
// function during server drain to avoid a goroutine leak.
func RateLimit(cfg RateLimitConfig) (Middleware, func(context.Context)) {
	if cfg.Limit <= 0 {
		cfg.Limit = rate.Limit(100)
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 200
	}
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 100_000
	}
	if cfg.IdleEvictAfter <= 0 {
		cfg.IdleEvictAfter = 30 * time.Minute
	}

	store := sync.Map{}

	evictCtx, evictStop := context.WithCancel(context.Background())
	go func() {
		tick := time.NewTicker(10 * time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-evictCtx.Done():
				return
			case now := <-tick.C:
				evict(&store, now, cfg.IdleEvictAfter, cfg.MaxEntries)
			}
		}
	}()

	shutdown := func(ctx context.Context) {
		evictStop()
		// Drain store so map doesn't keep references alive after shutdown.
		store.Range(func(k, _ any) bool {
			store.Delete(k)
			return true
		})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}
			ip := clientIP(r, cfg.TrustedProxy)
			if ip == "" {
				next.ServeHTTP(w, r)
				return
			}

			now := time.Now()
			// Load first, then LoadOrStore: constructing the candidate entry
			// allocates, and doing that on every request in the hot path is
			// waste when the IP is almost always already present.
			v, ok := store.Load(ip)
			if !ok {
				v, _ = store.LoadOrStore(ip, &ipEntry{lim: rate.NewLimiter(cfg.Limit, cfg.Burst), lastSeen: now})
			}
			// The store is private to this closure and only ever written with
			// *ipEntry, so this cannot fail today. The comma-ok is here because
			// an unchecked assertion is a panic in the request path, and the
			// cost of being wrong here is a crash rather than a bad metric.
			entry, ok := v.(*ipEntry)
			if !ok {
				slog.Error("rate limit store holds an unexpected value; skipping limiter",
					"client_ip", ip, "path", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}
			entry.touch(now)
			if !entry.lim.Allow() {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"too many requests"}}`))
				slog.DebugContext(r.Context(), "rate limited", "client_ip", ip, "path", r.URL.Path)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, shutdown
}

func evict(store *sync.Map, now time.Time, idleAfter time.Duration, maxEntries int) {
	// Pass 1: drop idle entries.
	store.Range(func(k, v any) bool {
		e, ok := v.(*ipEntry)
		if !ok {
			store.Delete(k)
			return true
		}
		if now.Sub(e.seen()) > idleAfter {
			store.Delete(k)
		}
		return true
	})

	// Pass 2: cap total size by evicting oldest first (best effort).
	count := 0
	store.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count <= maxEntries {
		return
	}
	type cand struct {
		key string
		ts  time.Time
	}
	var oldest []cand
	store.Range(func(k, v any) bool {
		key, ok := k.(string)
		if !ok {
			return true
		}
		e, ok := v.(*ipEntry)
		if !ok {
			return true
		}
		oldest = append(oldest, cand{key: key, ts: e.seen()})
		return true
	})
	// Partial sort is fine; we only need to remove (count - maxEntries).
	slices.SortFunc(oldest, func(a, b cand) int {
		if a.ts.Before(b.ts) {
			return -1
		}
		return 1
	})
	for i := 0; i < count-maxEntries && i < len(oldest); i++ {
		store.Delete(oldest[i].key)
	}
}

// clientIP returns the bucket key for a request.
//
// trustedProxy controls whether X-Forwarded-For is consulted at all. It is off
// by default and must stay off whenever the service is reachable other than
// through a proxy you control, because a client can then choose its own bucket
// by rotating the header and the limiter stops meaning anything.
//
// ## Why the RIGHTMOST entry, not the leftmost
//
// Reverse proxies APPEND to X-Forwarded-For. Given a client that sends
//
//	X-Forwarded-For: 1.2.3.4
//
// Caddy rewrites it to
//
//	X-Forwarded-For: 1.2.3.4, <the address Caddy actually saw>
//
// The leftmost entry is therefore whatever the client felt like sending, and
// trusting it hands the limiter's identity straight to the caller: rotate a
// forged value and every request lands in a fresh, empty bucket. That is not a
// subtle weakness, it disables rate limiting entirely with one header.
//
// The rightmost entry is the one the trusted proxy itself appended, i.e. the
// peer it really observed. For a single trusted proxy in front — the topology
// this project ships — that is exactly the client.
//
// With a longer proxy chain the rightmost entry is the proxy nearest the
// service rather than the client. That is why TrustedProxy is an explicit
// operator decision and not something inferred: the operator knows their chain.
func clientIP(r *http.Request, trustedProxy bool) string {
	if trustedProxy {
		if ip := rightmostForwardedFor(r.Header.Get("X-Forwarded-For")); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rightmostForwardedFor returns the last entry of an X-Forwarded-For list, or
// "" if the header is absent or contains nothing usable.
func rightmostForwardedFor(v string) string {
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ",")
	last := strings.TrimSpace(parts[len(parts)-1])
	// Strip an optional port so the bucket key is the address alone; a client
	// cannot change its source port anyway, and keeping it would let one client
	// mint unlimited buckets by varying the port.
	if host, _, err := net.SplitHostPort(last); err == nil {
		return host
	}
	return last
}
