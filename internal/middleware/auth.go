package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// Header names a client may use to present a credential. Both are supported so
// the same key works from a browser (X-API-Key, trivially set by $fetch) and
// from a CLI or MCP client (Authorization: Bearer, the convention most tooling
// already implements).
const (
	HeaderAPIKey = "X-API-Key"
	HeaderAuthz  = "Authorization"
)

// QueryAPIKey is the query parameter accepted on the WebSocket route only.
//
// Browsers cannot attach headers to a WebSocket handshake, so a query
// parameter is the only way a page can authenticate one. It is deliberately
// NOT accepted on other routes: query strings land in access logs, proxy logs
// and browser history, which would leak the key. See AuthConfig.AllowQueryKey.
const QueryAPIKey = "api_key"

// ErrAuthMisconfigured is returned by RequireAPIKey when authentication is
// switched on but no keys are configured. Failing loudly at boot matters: the
// alternative — starting with an empty key set and quietly rejecting everyone,
// or worse, quietly accepting everyone — is the kind of footgun that turns
// into an incident.
var ErrAuthMisconfigured = errors.New("auth: AUTH_ENABLED is set but no API keys are configured")

// AuthConfig configures API-key authentication.
type AuthConfig struct {
	// Enabled turns enforcement on. When false the returned middleware is a
	// no-op, which keeps local development and the test suite free of
	// credentials.
	Enabled bool

	// Keys is the set of accepted credentials. Any one grants access; this is
	// a shared-secret scheme, not per-user identity, so there is no "who".
	Keys []string

	// AllowQueryKey permits ?api_key=... in addition to the headers. Set this
	// only for routes a browser must reach over WebSocket.
	AllowQueryKey bool

	// Skip bypasses authentication for the given paths. Health and metrics
	// probes are the usual entries: an orchestrator has no credential to
	// present, and locking it out of /healthz turns a config error into an
	// outage loop.
	Skip []string
}

// RequireAPIKey returns a Middleware that rejects any request without a valid
// credential, plus the error that prevented it from being constructed.
//
// The second return value is non-nil exactly when configuration is unusable
// (auth on, no keys). Callers must treat that as fatal at startup.
func RequireAPIKey(cfg AuthConfig) (Middleware, error) {
	if !cfg.Enabled {
		return func(next http.Handler) http.Handler { return next }, nil
	}

	// Trim and drop empties so a trailing comma in the env var ("a,b,") does
	// not create a key that matches the empty string.
	keys := make([]string, 0, len(cfg.Keys))
	for _, k := range cfg.Keys {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return nil, ErrAuthMisconfigured
	}

	skip := make(map[string]struct{}, len(cfg.Skip))
	for _, p := range cfg.Skip {
		skip[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, bypass := skip[r.URL.Path]; bypass {
				next.ServeHTTP(w, r)
				return
			}

			presented := extractCredential(r, cfg.AllowQueryKey)
			if presented == "" || !matchesAnyKey(presented, keys) {
				rejectUnauthorized(w, r, presented == "")
				return
			}

			next.ServeHTTP(w, r)
		})
	}, nil
}

// extractCredential returns the credential the request presented, or "".
func extractCredential(r *http.Request, allowQueryKey bool) string {
	if h := strings.TrimSpace(r.Header.Get(HeaderAPIKey)); h != "" {
		return h
	}
	if h := strings.TrimSpace(r.Header.Get(HeaderAuthz)); h != "" {
		// Accept both "Bearer <key>" and a bare token, since some MCP clients
		// send the key without the scheme.
		if len(h) >= 7 && strings.EqualFold(h[:7], "bearer ") {
			return strings.TrimSpace(h[7:])
		}
		return h
	}
	if allowQueryKey {
		if q := strings.TrimSpace(r.URL.Query().Get(QueryAPIKey)); q != "" {
			return q
		}
	}
	return ""
}

// matchesAnyKey compares the presented credential against every configured key
// and only then decides.
//
// The loop deliberately does NOT short-circuit on a match: comparing
// sequentially and returning early leaks, through response timing, how many
// leading characters were correct and which key index matched. Every
// comparison runs and the results are OR-ed, so the work is independent of both
// the guess and the key set.
func matchesAnyKey(presented string, keys []string) bool {
	var ok int
	for _, k := range keys {
		// ConstantTimeCompare returns 0 immediately for length mismatches, so
		// the length of the guess is still observable. That is acceptable here
		// (keys are high-entropy), and hashing first would only move the leak
		// rather than remove it.
		ok |= subtle.ConstantTimeCompare([]byte(presented), []byte(k))
	}
	return ok == 1
}

func rejectUnauthorized(w http.ResponseWriter, r *http.Request, missing bool) {
	// Per RFC 9110 the challenge is required on a 401.
	w.Header().Set("WWW-Authenticate", `Bearer realm="agent-shaker"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	code, msg := "unauthorized", "a valid API key is required"
	if missing {
		msg = "missing API key: send it in the X-API-Key header or as 'Authorization: Bearer <key>'"
	}
	// Deliberately no reflection of the presented value and no hint about which
	// keys exist.
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":       code,
			"message":    msg,
			"request_id": RequestIDFromContext(r.Context()),
		},
	})

	// Log the failure so operators can spot credential-stuffing, but never the
	// presented value.
	slog.WarnContext(r.Context(), "unauthorized request",
		"path", r.URL.Path,
		"method", r.Method,
		"client_ip", clientIP(r, false),
		"credential_present", !missing,
	)
}

// APIKeysFromEnv reads a comma-separated credential list from envName,
// trimming whitespace and dropping empties. Exported so the fail-fast check
// in main and the middleware construct agree on the parsing rules.
func APIKeysFromEnv(envName string) []string {
	raw := os.Getenv(envName)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// DescribeAuthConfig renders a log-safe summary of the auth configuration.
// Keys are reduced to a count and a per-key fingerprint prefix that is too
// short to be usable, so an operator can confirm which key is loaded without
// the line becoming a credential leak.
func DescribeAuthConfig(enabled bool, keys []string) string {
	if !enabled {
		return "disabled"
	}
	if len(keys) == 0 {
		return "enabled but misconfigured (no keys)"
	}
	fps := make([]string, 0, len(keys))
	for _, k := range keys {
		if len(k) <= 4 {
			fps = append(fps, "****")
			continue
		}
		fps = append(fps, fmt.Sprintf("%s…(%d)", k[:4], len(k)))
	}
	return fmt.Sprintf("enabled, %d key(s): %s", len(keys), strings.Join(fps, ", "))
}
