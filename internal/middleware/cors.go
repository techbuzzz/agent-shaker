package middleware

import (
	"net/http"
	"slices"
	"strings"
)

// CORS returns a Middleware that sets the standard CORS response headers for
// requests whose Origin is in the supplied allow-list. The wildcard "*" is
// accepted but rejected when allowCredentials is true (the browser-rejected
// combination is a config bug, not a runtime condition).
//
// preflightMethods and preflightHeaders are the values echoed back in
// Access-Control-Allow-Methods / -Headers. Sensible defaults are used when
// the caller passes empty slices.
func CORS(allowedOrigins []string, allowCredentials bool, preflightMethods, preflightHeaders []string) Middleware {
	if allowCredentials && slices.Contains(allowedOrigins, "*") {
		panic("middleware.CORS: AllowCredentials=true is incompatible with a wildcard origin; specify explicit origins")
	}

	if len(preflightMethods) == 0 {
		preflightMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(preflightHeaders) == 0 {
		preflightHeaders = []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"}
	}

	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	set := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		set[strings.ToLower(o)] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			originLower := strings.ToLower(origin)
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := set[originLower]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			} else {
				// Origin not allowed; do not advertise this endpoint to the browser.
				next.ServeHTTP(w, r)
				return
			}
			if allowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(preflightMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(preflightHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
