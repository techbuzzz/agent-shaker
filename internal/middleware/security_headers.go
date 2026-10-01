package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders returns a Middleware that sets baseline security headers on
// every response.
//
// HSTS is only meaningful when the client-facing hop is encrypted, so it is
// decided **per request** from what the request itself proves:
//
//   - r.TLS != nil            — this process terminated TLS directly
//   - X-Forwarded-Proto: https — a reverse proxy in front did
//
// `assumeTLS` remains as an explicit override for a proxy that does not set
// X-Forwarded-Proto (a raw TCP passthrough, or a hand-rolled one). It is
// deliberately an OR, not a switch.
//
// ## Why this is per request rather than a startup flag
//
// It used to read a single `TLS_TERMINATED` env var resolved once at boot. That
// made HSTS depend on an operator remembering a second flag when enabling the
// TLS edge — and forgetting it failed *silently*. The service started healthy,
// served traffic, and simply never sent the header. That is the worst shape for
// a security control: no error, no restart, no signal.
//
// Deriving it from the request means the bundled Caddy edge gets HSTS
// automatically, and the failure mode "TLS is on but HSTS is off" is no longer
// reachable through configuration alone.
//
// ## Why a client cannot abuse the header
//
// X-Forwarded-Proto is attacker-controllable on a directly-exposed service, so
// a forged `https` on a plain request would add an HSTS header to a cleartext
// response. Per RFC 6797 a browser only honours HSTS received over a secure
// transport, so such a header is inert: it grants no protection and enables no
// downgrade. The forged value can only ever *add* a header that will be
// ignored, never remove one that should be sent.
//
// csp is optional; when empty the CSP header is omitted (callers should set the
// appropriate CSP per-route, e.g. the SPA gets a different one than the JSON
// API).
func SecurityHeaders(assumeTLS bool, csp string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if assumeTLS || requestIsHTTPS(r) {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			if csp != "" {
				h.Set("Content-Security-Policy", csp)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requestIsHTTPS reports whether the client-facing hop is encrypted, according
// to the request.
//
// X-Forwarded-Proto may be a comma-separated chain; only the first entry was
// added by the proxy closest to the client and is therefore the one that
// describes the hop the browser actually made. Later entries describe hops
// further back and, in a chain, may be attacker-supplied.
func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	v := r.Header.Get("X-Forwarded-Proto")
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.EqualFold(strings.TrimSpace(v), "https")
}

// DefaultSPA_CSP returns a Content Security Policy suitable for serving the
// bundled Nuxt SPA. Adjust via your build pipeline (hash-based script-src
// instead of 'self' if the SPA inlines scripts).
func DefaultSPA_CSP() string {
	return "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' ws: wss:"
}
