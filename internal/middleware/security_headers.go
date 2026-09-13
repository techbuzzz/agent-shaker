package middleware

import "net/http"

// SecurityHeaders returns a Middleware that sets baseline security headers
// on every response. HSTS is only set when isTLS is true because sending it
// over plain HTTP is meaningless and confusing.
//
// csp is optional; when empty the CSP header is omitted (callers should set
// the appropriate CSP per-route, e.g. the SPA gets a different one than the
// JSON API).
func SecurityHeaders(isTLS bool, csp string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if isTLS {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			if csp != "" {
				h.Set("Content-Security-Policy", csp)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// DefaultSPA_CSP returns a Content Security Policy suitable for serving the
// bundled Nuxt SPA. Adjust via your build pipeline (hash-based script-src
// instead of 'self' if the SPA inlines scripts).
func DefaultSPA_CSP() string {
	return "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' ws: wss:"
}
