package mcp

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

var boundAgent = uuid.MustParse("22222222-2222-2222-2222-222222222222")

// principalBoundTo builds the principal an authenticated request carries.
//
// A nil agent models every credential that predates the binding: the legacy
// flat API_KEYS secret, the anonymous principal in auth-disabled development,
// and any key issued before migration 017.
func principalBoundTo(agent *uuid.UUID) middleware.Principal {
	return middleware.Principal{
		Type:        models.PersonKindService,
		ID:          uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		DisplayName: "test principal",
		Scopes:      []models.Scope{models.ScopeRead, models.ScopeWrite},
		AgentID:     agent,
	}
}

// TestResolveIdentityPrecedence is the whole point of migration 017.
//
// Before it, extractContext believed whatever agent_id arrived in the query
// string, and the key behind that request was scoped only to a project — so any
// holder of a project key could claim any agent in it, heartbeat on their
// behalf, claim their tasks and complete work nobody assigned them. Every row
// here is a case that either used to be a hole or would become one if this
// function were written the obvious way.
func TestResolveIdentityPrecedence(t *testing.T) {
	t.Parallel()

	otherAgent := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name      string
		bound     *uuid.UUID
		requested string
		want      string
		wantErr   bool
	}{
		{
			// A URL that simply omits the id is not a disagreement. Refusing it
			// would break the generated configs for no security gain: the key
			// already says which agent this is.
			name:  "bound key with no agent_id in the url",
			bound: &boundAgent,
			want:  boundAgent.String(),
		},
		{
			name:      "bound key with the matching agent_id",
			bound:     &boundAgent,
			requested: boundAgent.String(),
			want:      boundAgent.String(),
		},
		{
			// UUIDs are case-insensitive in the wire format Postgres emits, and
			// a client copying one out of a URL may well lowercase it. Refusing
			// the same agent for a capital letter would be a bug that reads as a
			// security control.
			name:      "bound key with the same id in different case",
			bound:     &boundAgent,
			requested: strings.ToUpper(boundAgent.String()),
			want:      boundAgent.String(),
		},
		{
			// The impersonation attempt.
			name:      "bound key with somebody else's agent_id",
			bound:     &boundAgent,
			requested: otherAgent.String(),
			wantErr:   true,
		},
		{
			name:      "bound key with a non-uuid agent_id",
			bound:     &boundAgent,
			requested: "not-a-uuid",
			wantErr:   true,
		},
		{
			// The compatibility guarantee. Every key issued before migration 017
			// has a NULL agent_id, as do the legacy flat API_KEYS secrets and the
			// auth-disabled development mode. They carry their identity in the URL
			// and nothing else, and refusing them would break every deployment
			// that works today.
			name:      "unbound key passes the requested id through",
			bound:     nil,
			requested: otherAgent.String(),
			want:      otherAgent.String(),
		},
		{
			name:  "unbound key with no requested id",
			bound: nil,
			want:  "",
		},
		{
			// Whitespace arrives from a hand-edited .mcp.json more often than
			// one would like.
			name:      "unbound key with surrounding whitespace",
			bound:     nil,
			requested: "  " + otherAgent.String() + "  ",
			want:      otherAgent.String(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveIdentity(tc.bound, tc.requested)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveIdentity(%v, %q) = %q, want an error", tc.bound, tc.requested, got)
				}
				if !errors.Is(err, ErrAgentIdentityMismatch) {
					t.Errorf("error %v does not wrap ErrAgentIdentityMismatch", err)
				}
				// The message has to be actionable: it is the only thing the
				// operator sees when their agent mysteriously cannot connect,
				// and "forbidden" alone would cost them an hour.
				if !strings.Contains(err.Error(), "drop the agent_id") {
					t.Errorf("error %q does not say how to fix it", err)
				}
				if got != "" {
					t.Errorf("resolveIdentity returned %q alongside the error, want empty", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveIdentity(%v, %q): %v", tc.bound, tc.requested, err)
			}
			if got != tc.want {
				t.Errorf("resolveIdentity(%v, %q) = %q, want %q", tc.bound, tc.requested, got, tc.want)
			}
		})
	}
}

// TestApplyPrincipalIdentityNarrowing checks the path the request actually
// takes: a principal with a binding narrows the context, and one without leaves
// it exactly as the URL set it.
func TestApplyPrincipalIdentityNarrowing(t *testing.T) {
	t.Parallel()

	t.Run("a bound principal overrides a matching url value", func(t *testing.T) {
		t.Parallel()
		ctx := MCPContext{AgentID: boundAgent.String()}
		if err := applyPrincipalIdentity(&ctx, principalBoundTo(&boundAgent)); err != nil {
			t.Fatalf("applyPrincipalIdentity: %v", err)
		}
		if ctx.AgentID != boundAgent.String() {
			t.Errorf("AgentID = %q, want %q", ctx.AgentID, boundAgent)
		}
	})

	t.Run("an unbound principal changes nothing", func(t *testing.T) {
		t.Parallel()
		requested := "44444444-4444-4444-4444-444444444444"
		ctx := MCPContext{AgentID: requested}
		// The legacy and anonymous principals both have a nil AgentID, which is
		// the state every pre-017 key resolves to.
		if err := applyPrincipalIdentity(&ctx, principalBoundTo(nil)); err != nil {
			t.Fatalf("applyPrincipalIdentity: %v", err)
		}
		if ctx.AgentID != requested {
			t.Errorf("AgentID = %q, want the URL value %q untouched", ctx.AgentID, requested)
		}
	})

	t.Run("a mismatch is refused and the context is left alone", func(t *testing.T) {
		t.Parallel()
		// A refused request must not fall through to the tools with the
		// requested identity intact: the handler answers and returns, and this
		// is the state that state would have been in.
		other := uuid.MustParse("33333333-3333-3333-3333-333333333333")
		ctx := MCPContext{AgentID: other.String()}
		if err := applyPrincipalIdentity(&ctx, principalBoundTo(&boundAgent)); !errors.Is(err, ErrAgentIdentityMismatch) {
			t.Fatalf("err = %v, want ErrAgentIdentityMismatch", err)
		}
		if ctx.AgentID != other.String() {
			t.Errorf("AgentID = %q, want the rejected value %q — a refused request must not be silently re-pointed", ctx.AgentID, other)
		}
	})
}
