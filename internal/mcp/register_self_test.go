package mcp

import (
	"strings"
	"testing"
)

// TestResolveSelfRoleRefusesPM is the guard for a privilege escalation.
//
// isPMAgent reads the role of the agent on the connection, and register_self
// mints a row the caller is then free to connect as. Without this refusal, one
// tool call buys an agent milestone and global-context approval authority.
//
// The escalation route is real for any credential without an agent binding —
// register_self is the onboarding path for exactly that case — so the guard
// cannot live in the key binding and has to live here.
func TestResolveSelfRoleRefusesPM(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"pm", "PM", " pm ", "Pm"} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			role, err := resolveSelfRole(raw)
			if err == nil {
				t.Fatalf("resolveSelfRole(%q) = %q, want a refusal", raw, role)
			}
			// The message has to say who can grant it, or the agent retries
			// with another spelling of "pm" and then gives up on the feature.
			if !strings.Contains(err.Error(), "human") {
				t.Errorf("error %q does not say the role is granted by a human", err)
			}
			if !strings.Contains(err.Error(), "backend") {
				t.Errorf("error %q does not offer a role the agent may take", err)
			}
		})
	}
}

// TestResolveSelfRoleAllowsEverythingElse keeps the guard narrow. Only PM
// carries approval authority, so the others stay open: refusing them would break
// an agent registering honestly as a frontend one for no security gain.
func TestResolveSelfRoleAllowsEverythingElse(t *testing.T) {
	t.Parallel()

	tests := []struct{ in, want string }{
		{"", "backend"},
		{"backend", "backend"},
		{"frontend", "frontend"},
		{"devops", "devops"},
		{"qa", "qa"},
		{"  frontend  ", "frontend"},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, err := resolveSelfRole(tc.in)
			if err != nil {
				t.Fatalf("resolveSelfRole(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("resolveSelfRole(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRegisterSelfSchemaDoesNotAdvertisePM keeps the tool contract and the
// server's behaviour in agreement.
//
// The enum used to list "pm", so a client was told the server accepted it and
// only found out at runtime — after it had already chosen the call. A schema
// that over-promises costs more than one that under-promises, because the
// agent retries an advertised value rather than reporting the discrepancy.
func TestRegisterSelfSchemaDoesNotAdvertisePM(t *testing.T) {
	t.Parallel()

	h := NewMCPHandler(nil, nil)
	result, rpcErr := h.handleToolsList(MCPContext{})
	if rpcErr != nil {
		t.Fatalf("handleToolsList: %v", rpcErr)
	}
	list, ok := result.(ToolsListResult)
	if !ok {
		t.Fatalf("unexpected result type %T", result)
	}

	for _, tool := range list.Tools {
		if tool.Name != "register_self" {
			continue
		}

		role, ok := tool.InputSchema.Properties["role"].(map[string]interface{})
		if !ok {
			t.Fatal("register_self has no role property")
		}
		enum, _ := role["enum"].([]string)
		for _, v := range enum {
			if strings.EqualFold(v, "pm") {
				t.Errorf("schema advertises %q but the server refuses it: %v", v, enum)
			}
		}
		// The description has to carry the reason, because an agent that
		// genuinely believes it is a PM deserves more than a missing enum value.
		if desc, _ := role["description"].(string); !strings.Contains(desc, "NOT 'pm'") {
			t.Errorf("role description does not say pm is refused: %q", desc)
		}
		return
	}
	t.Fatal("register_self is not in the tool list")
}

// The ON CONFLICT fix is deliberately not unit-tested here.
//
// It needs a real database: Postgres rejects an `ON CONFLICT` whose target
// matches no index, and there is no sqlmock or testcontainers dependency in this
// module — a test that faked the transaction would pass against the broken
// statement too, which is the only thing worth testing. The assertion that
// matters is made against a running server instead: register_self succeeds, and
// registering the same name twice returns the same id.
func TestRegisterSelfUpsertIsVerifiedAgainstALiveDatabase(t *testing.T) {
	t.Skip("covered by the live-stack verification; see migrations/018_agents_unique_name.up.sql")
}
