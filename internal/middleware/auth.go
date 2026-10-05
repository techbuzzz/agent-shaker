package middleware

import (
	"context"
	"crypto/subtle"
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
	// pass-through that attaches an anonymous principal, which keeps local
	// development and the test suite free of credentials without leaving
	// handlers without an identity.
	Enabled bool

	// Keys is the set of accepted credentials from the flat API_KEYS setting.
	// Any one grants access; this is a shared-secret scheme, not per-user
	// identity, so there is no "who" — the request carries the synthetic
	// LegacyPrincipal instead.
	//
	// Ignored for matching when Resolver is set; it stays in the struct because
	// the legacy secrets must keep working for the bootstrap paths that have
	// not been reissued yet.
	Keys []string

	// Resolver turns a presented credential into a real principal backed by the
	// api_keys table. When set it is the authority, and Keys is only a
	// fallback for the routes wired with a resolver-less middleware.
	Resolver KeyResolver

	// AllowQueryKey permits ?api_key=... in addition to the headers. Set this
	// only for routes a browser must reach over WebSocket.
	AllowQueryKey bool

	// Skip bypasses authentication for the given paths. Health and metrics
	// probes are the usual entries: an orchestrator has no credential to
	// present, and locking it out of /healthz turns a config error into an
	// outage loop.
	Skip []string
}

// RequireAPIKey returns a Middleware that resolves the caller to a principal and
// rejects any request without a usable credential, plus the error that
// prevented it from being constructed.
//
// The second return value is non-nil exactly when configuration is unusable
// (auth on, no keys and no resolver). Callers must treat that as fatal at
// startup.
func RequireAPIKey(cfg AuthConfig) (Middleware, error) {
	if !cfg.Enabled {
		// Not a bare pass-through: an authenticated surface hands every
		// downstream handler a principal, and a handler that finds none has no
		// way to tell "authentication is off" from "this route was wired
		// without auth". The anonymous principal makes the first explicit.
		anonymous := AnonymousPrincipal()
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), anonymous)))
			})
		}, nil
	}

	// Trim and drop empties so a trailing comma in the env var ("a,b,") does
	// not create a key that matches the empty string.
	keys := make([]string, 0, len(cfg.Keys))
	for _, k := range cfg.Keys {
		if k = strings.TrimSpace(k); k != "" {
			keys = append(keys, k)
		}
	}
	// A resolver makes an empty key set legitimate: the credentials live in the
	// database now, and the operator legitimately has none in the environment.
	// Without a resolver it is the misconfiguration the sentinel describes.
	if len(keys) == 0 && cfg.Resolver == nil {
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
			if presented == "" {
				rejectUnauthorized(w, r, true)
				return
			}

			principal, err := authenticate(r.Context(), presented, cfg, keys)
			switch {
			case err == nil:
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
			case errors.Is(err, ErrKeyStoreUnavailable):
				rejectUnavailable(w, r, err)
			default:
				rejectUnauthorized(w, r, false)
			}
		})
	}, nil
}

// authenticate maps a presented credential to a principal.
//
// The resolver is consulted first; the flat keys are the fallback, which is what
// keeps an existing deployment working through the upgrade: a value in API_KEYS
// is still accepted, it just carries the synthetic principal instead of a
// database row.
func authenticate(ctx context.Context, presented string, cfg AuthConfig, keys []string) (Principal, error) {
	if cfg.Resolver != nil {
		principal, err := cfg.Resolver.ResolveKey(ctx, presented)
		if err != nil {
			return Principal{}, err
		}
		return principal, nil
	}

	if !matchesAnyKey(presented, keys) {
		return Principal{}, ErrKeyUnknown
	}
	return LegacyPrincipal(), nil
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
	msg := "a valid API key is required"
	if missing {
		msg = "missing API key: send it in the X-API-Key header or as 'Authorization: Bearer <key>'"
	}

	// Deliberately no reflection of the presented value and no hint about which
	// keys exist. A revoked key and a key that was never issued produce the
	// same body, so this endpoint cannot be used to confirm that a given
	// credential once worked.
	writeAuthError(w, r, http.StatusUnauthorized, "unauthorized", msg, map[string]string{
		// Per RFC 9110 the challenge is required on a 401.
		"WWW-Authenticate": `Bearer realm="agent-shaker"`,
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

// rejectUnavailable answers 503 when the credential could not be checked
// because the backing store is unreachable.
//
// The distinction from 401 is the point of this function: a database outage
// that answers 401 reads to every operator exactly like a credential attack,
// and the first hour of the incident is spent looking for an intruder who does
// not exist.
func rejectUnavailable(w http.ResponseWriter, r *http.Request, cause error) {
	slog.ErrorContext(r.Context(), "credential check failed: key store unavailable",
		"path", r.URL.Path,
		"method", r.Method,
		"error", cause,
	)
	writeAuthError(w, r, http.StatusServiceUnavailable, "unavailable",
		"the credential store is temporarily unavailable", nil)
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
