package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		trusted    bool
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "direct request, no proxy trust",
			remoteAddr: "203.0.113.9:5555",
			want:       "203.0.113.9",
		},
		{
			// The default must not be swayed by a client-supplied header: if
			// the service is reachable directly, that header is fiction.
			name:       "forged XFF ignored when proxy is not trusted",
			remoteAddr: "203.0.113.9:5555",
			xff:        "1.2.3.4",
			want:       "203.0.113.9",
		},
		{
			name:       "trusted proxy, single hop",
			trusted:    true,
			remoteAddr: "172.18.0.5:41234",
			xff:        "203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			// THE attack. A proxy appends, so a client that sends its own
			// XFF produces "<forged>, <real>". Reading the leftmost entry
			// would let the caller pick its own bucket and rotate it to defeat
			// rate limiting entirely.
			name:       "forged prefix does not choose the bucket",
			trusted:    true,
			remoteAddr: "172.18.0.5:41234",
			xff:        "1.2.3.4, 203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "longer forged chain still resolves to the real peer",
			trusted:    true,
			remoteAddr: "172.18.0.5:41234",
			xff:        "1.1.1.1, 2.2.2.2, 203.0.113.9",
			want:       "203.0.113.9",
		},
		{
			name:       "whitespace around entries tolerated",
			trusted:    true,
			remoteAddr: "172.18.0.5:41234",
			xff:        "  1.2.3.4 ,  203.0.113.9  ",
			want:       "203.0.113.9",
		},
		{
			// Stripping the port matters: a client could otherwise vary its
			// source port to mint buckets even with an honest-looking entry.
			name:       "port is stripped from the forwarded entry",
			trusted:    true,
			remoteAddr: "172.18.0.5:41234",
			xff:        "203.0.113.9:44321",
			want:       "203.0.113.9",
		},
		{
			name:       "falls back to peer when header absent",
			trusted:    true,
			remoteAddr: "203.0.113.9:5555",
			want:       "203.0.113.9",
		},
		{
			name:       "falls back to peer when header is only separators",
			trusted:    true,
			remoteAddr: "203.0.113.9:5555",
			xff:        " , ",
			want:       "203.0.113.9",
		},
		{
			name:       "ipv6 peer",
			remoteAddr: "[2001:db8::1]:443",
			want:       "2001:db8::1",
		},
		{
			name:       "bare host with no port",
			remoteAddr: "203.0.113.9",
			want:       "203.0.113.9",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}

			if got := clientIP(req, tc.trusted); got != tc.want {
				t.Errorf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRateLimitForgedXFFCannotMintBuckets is the behaviour that matters at the
// limiter's level rather than the parser's: many clients forging different
// XFF prefixes must still share the one bucket the proxy created for them.
// With leftmost parsing this test fails, because each forged value lands in a
// fresh empty bucket and nothing is ever throttled.
func TestRateLimitForgedXFFCannotMintBuckets(t *testing.T) {
	t.Parallel()

	const limit, burst = 1, 2
	mw, shutdown := RateLimit(RateLimitConfig{Limit: limit, Burst: burst, TrustedProxy: true})
	defer shutdown(t.Context())

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	// Drain the shared bucket.
	for i := 0; i < burst; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		req.RemoteAddr = "172.18.0.5:1"
		req.Header.Set("X-Forwarded-For", "203.0.113.9")
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
	}

	// Ten calls with ten different forged prefixes, each followed by the real
	// peer the proxy appended. That is the exact shape Caddy produces for a
	// client that sends its own X-Forwarded-For, so all ten must be throttled:
	// they are one client as far as the limiter is concerned.
	//
	// Note the header below is what the *proxy* forwards, not what the caller
	// sent. Modelling the raw request instead would test nothing — with no
	// appended entry there is no way to tell a forged value from a real one.
	const realPeer = "203.0.113.9"
	var wg sync.WaitGroup
	results := make([]int, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			req.RemoteAddr = "172.18.0.5:1"
			req.Header.Set("X-Forwarded-For", forgedAddr(i)+", "+realPeer)
			rec := httptest.NewRecorder()
			mw(ok).ServeHTTP(rec, req)
			results[i] = rec.Code
		}(i)
	}
	wg.Wait()

	for i, code := range results {
		if code != http.StatusTooManyRequests {
			t.Errorf("forged prefix %s: status = %d, want 429 (a forged header must not mint a fresh bucket)",
				forgedAddr(i), code)
		}
	}
}

// TestRateLimitSeparatesDistinctClients is the other half: two genuinely
// different clients must not throttle each other. Without TRUSTED_PROXY the
// buckets collapse and this fails behind a proxy.
func TestRateLimitSeparatesDistinctClients(t *testing.T) {
	t.Parallel()

	mw, shutdown := RateLimit(RateLimitConfig{Limit: 1, Burst: 1, TrustedProxy: true})
	defer shutdown(t.Context())

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	call := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		req.RemoteAddr = "172.18.0.5:1"
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		return rec.Code
	}

	if code := call("198.51.100.1"); code != http.StatusOK {
		t.Fatalf("first client: status = %d, want 200", code)
	}
	if code := call("198.51.100.1"); code != http.StatusTooManyRequests {
		t.Errorf("first client, second request: status = %d, want 429", code)
	}
	// A different real client must have its own bucket.
	if code := call("198.51.100.2"); code != http.StatusOK {
		t.Errorf("second client: status = %d, want 200 (a distinct client must not be throttled by the first)", code)
	}
}

// TestRateLimitSkipPaths keeps probes out of the limiter: an orchestrator has
// no credential and a 429 there becomes a restart loop.
func TestRateLimitSkipPaths(t *testing.T) {
	t.Parallel()

	mw, shutdown := RateLimit(RateLimitConfig{
		Limit: 1, Burst: 1,
		Skip:         SkipPaths("/ws", "/healthz", "/readyz", "/metrics"),
		TrustedProxy: true,
	})
	defer shutdown(t.Context())

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "172.18.0.5:1"
			req.Header.Set("X-Forwarded-For", "198.51.100.7")
			rec := httptest.NewRecorder()
			mw(ok).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s request %d: status = %d, want 200 (probes are exempt)", path, i, rec.Code)
			}
		}
	}
}

// TestRateLimitRefills guards against a limiter that latches off forever.
func TestRateLimitRefills(t *testing.T) {
	t.Parallel()

	mw, shutdown := RateLimit(RateLimitConfig{Limit: 20, Burst: 1, TrustedProxy: true})
	defer shutdown(t.Context())

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	call := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		req.RemoteAddr = "172.18.0.5:1"
		req.Header.Set("X-Forwarded-For", "198.51.100.8")
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		return rec.Code
	}

	_ = call()
	if code := call(); code != http.StatusTooManyRequests {
		t.Fatalf("second immediate request: status = %d, want 429", code)
	}
	time.Sleep(120 * time.Millisecond)
	if code := call(); code != http.StatusOK {
		t.Errorf("after refill interval: status = %d, want 200", code)
	}
}

func forgedAddr(i int) string {
	return "10.0.0." + string(rune('0'+i))
}
