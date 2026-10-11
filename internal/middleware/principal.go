package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// Principal is the identity behind a request.
//
// It exists because "is this key valid?" is the wrong question for everything
// that follows: attribution needs a subject, permissions need a grant, and the
// approval flow (M4) needs to know it is talking to a human. A Principal is the
// answer to all three, resolved once per request at the edge.
type Principal struct {
	// Type is the kind of identity. Always set — the anonymous and legacy
	// principals carry explicit types rather than leaving it empty, because an
	// unset kind would silently fail an approval-authority check later.
	Type models.PersonKind

	// ID references people.id. The two synthetic principals below use reserved
	// UUIDs that no inserted row can collide with: a real person row always
	// carries a random v4 id, so those two values are unreachable in practice.
	ID uuid.UUID

	// DisplayName is safe to log and to render. It is never a credential.
	DisplayName string

	// Scopes is the granted set. Empty means unrestricted; see models.Allows for
	// why that default is the compatible one.
	Scopes []models.Scope

	// ProjectID narrows a key to one project, or is nil for a key that is not
	// project-scoped. Carried on the principal so a handler that needs it does
	// not have to re-read the key row.
	ProjectID *uuid.UUID

	// Synthetic marks a principal that is not backed by a people row: either
	// authentication is switched off, or the caller presented one of the flat
	// API_KEYS secrets. Audit events (M3) record it, so "a human did this" and
	// "somebody with a shared secret did this" stay distinguishable.
	Synthetic bool
}

// Synthetic principal identities.
//
// Reserved, deterministic, and outside the space gen_random_uuid() produces.
var (
	// AnonymousPrincipalID is attached when authentication is disabled.
	AnonymousPrincipalID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// LegacyPrincipalID is attached to any request authenticated by one of the
	// flat API_KEYS secrets.
	LegacyPrincipalID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

// unrestricted is the scope set of a synthetic principal: the wildcard, so the
// scope rules are genuinely evaluated rather than bypassed by a special case in
// the check itself.
var unrestricted = []models.Scope{models.ScopeAll}

// AnonymousPrincipal is the identity of a request that arrived while
// authentication was switched off.
//
// It is deliberately not "no identity": handlers downstream can then treat
// "no principal in context" as the anomaly it actually is — a route that was
// wired without auth — instead of silently behaving like an anonymous user.
func AnonymousPrincipal() Principal {
	return Principal{
		Type:        models.PersonKindService,
		ID:          AnonymousPrincipalID,
		DisplayName: "anonymous (authentication disabled)",
		Scopes:      unrestricted,
		Synthetic:   true,
	}
}

// LegacyPrincipal is the identity behind a flat API_KEYS secret.
//
// This is the compatibility bridge: a deployment that upgrades keeps working
// with the same API_KEYS value it had before, and the request carries a real
// principal from the first request on the new version rather than an empty
// hole that has to be patched later.
func LegacyPrincipal() Principal {
	return Principal{
		Type:        models.PersonKindService,
		ID:          LegacyPrincipalID,
		DisplayName: "api_keys (legacy shared secret)",
		Scopes:      unrestricted,
		Synthetic:   true,
	}
}

// Allows reports whether the principal may exercise the given scope.
func (p Principal) Allows(scope models.Scope) bool { return models.Allows(p.Scopes, scope) }

// IsAnonymous reports whether the principal is the placeholder attached when
// authentication is disabled.
func (p Principal) IsAnonymous() bool { return p.ID == AnonymousPrincipalID }

type principalContextKey struct{}

// WithPrincipal returns a context carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

// PrincipalFromContext returns the principal attached by RequireAPIKey.
//
// The bool is false when authentication is disabled *and* a route was mounted
// outside the auth chain, which is a wiring mistake rather than a state a
// handler should have to reason about.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(Principal)
	return p, ok
}

// RequireScope rejects any request whose principal lacks the scope.
//
// Mount it *inside* RequireAPIKey in the chain — i.e. later in the Apply list —
// so the principal is already in the context when this runs. The ordering is
// not obvious from the type signatures, so it is stated here rather than left
// to be discovered as a permanent 401.
func RequireScope(scope models.Scope) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorizeScope(w, r, scope) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireScopeOnPaths applies a scope to a subset of routes, identified by
// path prefix, and leaves the rest of the tree alone.
//
// It exists so the privileged surface can be named in one place. Key and
// principal management is the administrative API of this service; if the check
// were spread across route registrations, deleting one route would silently
// delete its authorisation with it.
func RequireScopeOnPaths(scope models.Scope, prefixes ...string) Middleware {
	need := make([]string, 0, len(prefixes))
	for _, p := range prefixes {
		if trimmed := strings.TrimSuffix(strings.TrimSpace(p), "/"); trimmed != "" {
			need = append(need, trimmed)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !matchesAnyPrefix(r.URL.Path, need) {
				next.ServeHTTP(w, r)
				return
			}
			if !authorizeScope(w, r, scope) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireMethodScope derives the required scope from the request method: safe
// methods need read, everything else needs write.
//
// One middleware instead of an annotation per route, and it fails closed — a
// verb this function has never seen is a mutation, so a new method cannot
// arrive with read access by omission.
func RequireMethodScope() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorizeScope(w, r, models.ScopeForMethod(r.Method)) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authorizeScope performs the check and writes the rejection. It returns true
// when the request may proceed.
func authorizeScope(w http.ResponseWriter, r *http.Request, scope models.Scope) bool {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		// No principal at all means this route was mounted outside the auth
		// chain. Refusing is the only safe reading: the alternative is a handler
		// that treats "unknown caller" as "allowed", which is exactly the shape
		// of an authentication bypass.
		slog.ErrorContext(r.Context(), "route reached without a principal; check the middleware chain",
			"path", r.URL.Path,
			"method", r.Method,
			"required_scope", string(scope),
		)
		rejectForbidden(w, r, scope, Principal{})
		return false
	}

	if p.Allows(scope) {
		return true
	}

	// The rejection names the missing scope and the principal, both of which the
	// caller already knows, and never the granted set: telling a caller exactly
	// what it lacks is fine, but echoing the whole grant turns a 403 into a
	// enumeration oracle for a shared identity.
	slog.WarnContext(r.Context(), "scope check failed",
		"path", r.URL.Path,
		"method", r.Method,
		"required_scope", string(scope),
		"principal_type", p.Type.String(),
		"principal_id", p.ID.String(),
		"client_ip", clientIP(r, false),
	)
	rejectForbidden(w, r, scope, p)
	return false
}

func matchesAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

// rejectForbidden writes a 403 in the same envelope as every other error.
//
// The envelope is hand-written rather than delegated to internal/httpx because
// httpx imports this package for RequestIDFromContext — reusing it here would
// be an import cycle. The shape is therefore pinned by
// TestForbiddenResponseShape, which fails if the two envelopes diverge.
func rejectForbidden(w http.ResponseWriter, r *http.Request, scope models.Scope, p Principal) {
	who := "the caller"
	if p.ID != uuid.Nil {
		who = p.DisplayName
	}

	writeAuthError(w, r, http.StatusForbidden, "forbidden",
		"this credential is not allowed to "+scopeLabel(scope), nil)

	slog.DebugContext(r.Context(), "forbidden request", "subject", who)
}

// scopeLabel turns a scope into a phrase for a human-readable message.
func scopeLabel(scope models.Scope) string {
	switch scope {
	case models.ScopeRead:
		return "read"
	case models.ScopeWrite:
		return "write"
	case models.ScopeAdmin:
		return "manage people and keys"
	default:
		return string(scope)
	}
}

// writeAuthError renders the shared error envelope for the auth middleware.
//
// Factored out of rejectUnauthorized so the 401 and the 403 cannot drift: two
// hand-written envelopes in one file is exactly how a client ends up with two
// parsers.
func writeAuthError(w http.ResponseWriter, r *http.Request, status int, code, message string, headers map[string]string) {
	for k, v := range headers {
		w.Header().Set(k, v)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":       code,
			"message":    message,
			"request_id": RequestIDFromContext(r.Context()),
		},
	})
}
