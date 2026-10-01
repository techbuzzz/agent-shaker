package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// serve runs the middleware over a request and returns the response headers.
func serve(t *testing.T, assumeTLS bool, req *http.Request) http.Header {
	t.Helper()
	rec := httptest.NewRecorder()
	SecurityHeaders(assumeTLS, "")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
	).ServeHTTP(rec, req)
	return rec.Result().Header
}

func TestSecurityHeadersHSTS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		assumeTLS bool
		tls       bool
		xfp       string
		wantHSTS  bool
	}{
		{
			// The regression this change exists for: TLS terminated by the edge,
			// operator did NOT set TLS_TERMINATED. Previously HSTS was silently
			// omitted; now the request proves the hop is encrypted.
			name:     "proxied https is detected without any env var",
			xfp:      "https",
			wantHSTS: true,
		},
		{
			name:     "direct tls is detected",
			tls:      true,
			wantHSTS: true,
		},
		{
			name:     "plain http gets no hsts",
			wantHSTS: false,
		},
		{
			// The explicit override still works, for a proxy that does not set
			// the header at all.
			name:      "assumeTLS forces hsts on",
			assumeTLS: true,
			wantHSTS:  true,
		},
		{
			name:     "forwarded proto is case insensitive",
			xfp:      "HTTPS",
			wantHSTS: true,
		},
		{
			name:     "surrounding whitespace tolerated",
			xfp:      "  https ",
			wantHSTS: true,
		},
		{
			// Only the first entry describes the client-facing hop. A later
			// "http" must not suppress HSTS for a genuinely encrypted request.
			name:     "chain uses the entry closest to the client",
			xfp:      "https, http",
			wantHSTS: true,
		},
		{
			// A forged trailing value must not talk us out of HSTS.
			name:     "chain does not let a later entry downgrade",
			xfp:      "http, https",
			wantHSTS: false,
		},
		{
			// Direct TLS plus a header claiming plain HTTP: the actual socket
			// wins, because that is the hop that carries the bytes.
			name:     "direct tls wins over a contradictory header",
			tls:      true,
			xfp:      "http",
			wantHSTS: true,
		},
		{
			name:     "unrelated scheme is not https",
			xfp:      "gopher",
			wantHSTS: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if tc.xfp != "" {
				req.Header.Set("X-Forwarded-Proto", tc.xfp)
			}

			h := serve(t, tc.assumeTLS, req)
			_, got := h["Strict-Transport-Security"]
			if got != tc.wantHSTS {
				t.Errorf("HSTS present = %v, want %v", got, tc.wantHSTS)
			}
		})
	}
}

// TestSecurityHeadersAlwaysEmitted guards the headers that are not conditional:
// dropping one of these silently weakens every response.
func TestSecurityHeadersAlwaysEmitted(t *testing.T) {
	t.Parallel()

	h := serve(t, false, httptest.NewRequest(http.MethodGet, "/", nil))
	for _, name := range []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
	} {
		if h.Get(name) == "" {
			t.Errorf("%s is missing", name)
		}
	}
}

// TestSecurityHeadersCSPOmittedWhenEmpty keeps the documented contract: an
// empty csp means "caller decides", not "send an empty policy".
func TestSecurityHeadersCSPOmittedWhenEmpty(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	SecurityHeaders(false, "")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}),
	).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Result().Header.Get("Content-Security-Policy"); got != "" {
		t.Errorf("CSP = %q, want empty", got)
	}
}
