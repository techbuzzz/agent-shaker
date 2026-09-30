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

// ipEntry is one row of the per-IP limiter store. It is package-level so the
// eviction helper can see it; the store is still scoped to a single
// RateLimit() invocation via a sync.Map captured by closure.
type ipEntry struct {
	lim      *rate.Limiter
	lastSeen time.Time
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
			v, _ := store.LoadOrStore(ip, &ipEntry{lim: rate.NewLimiter(cfg.Limit, cfg.Burst), lastSeen: now})
			entry := v.(*ipEntry)
			entry.lastSeen = now
			if !entry.lim.Allow() {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"too many requests"}}`))
				slog.Debug("rate limited", "client_ip", ip, "path", r.URL.Path)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, shutdown
}

func evict(store *sync.Map, now time.Time, idleAfter time.Duration, maxEntries int) {
	// Pass 1: drop idle entries.
	store.Range(func(k, v any) bool {
		e := v.(*ipEntry)
		if now.Sub(e.lastSeen) > idleAfter {
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
		oldest = append(oldest, cand{key: k.(string), ts: v.(*ipEntry).lastSeen})
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

func clientIP(r *http.Request, trustedProxy bool) string {
	if trustedProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
			if first != "" {
				return first
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
