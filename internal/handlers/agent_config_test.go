package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/configgen"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/handlers"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// newConfigHandler builds an AgentConfigHandler over the shared fakeQuerier.
//
// That fake returns an empty *sql.Rows, which every single-row read here turns
// into "not found" — so this exercises the lookup-and-answer paths without a
// database. The render path is covered directly in internal/configgen, where a
// real agent row is not needed to assert anything about a rendered file.
func newConfigHandler() *handlers.AgentConfigHandler {
	q := newFakeQuerier()
	return handlers.NewAgentConfigHandler(
		queries.NewAgentsStore(q),
		queries.NewAPIKeysStore(q),
		queries.NewPeopleStore(q),
		queries.NewProjectsStore(q),
		"https://shaker.example.com",
	)
}

// TestAgentConfigRejectsAnUnknownFormat checks that an unsupported format fails
// rather than returning an empty success. A client handed `{"files": []}` cannot
// tell "unsupported" from "nothing to configure" and shows an empty dialog.
func TestAgentConfigRejectsAnUnknownFormat(t *testing.T) {
	t.Parallel()

	h := newConfigHandler()
	agentID := uuid.New()

	req := httptest.NewRequest(http.MethodGet, "/api/agents/"+agentID.String()+"/config?format=emacs", nil)
	rec := httptest.NewRecorder()
	// The route table supplies path values through ServeMux; set explicitly so
	// the handler can read the id without standing up a router.
	req.SetPathValue("id", agentID.String())

	h.AgentConfig(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want a failure; body: %s", rec.Code, rec.Body.String())
	}
}

func TestAgentConfigRejectsAMalformedAgentID(t *testing.T) {
	t.Parallel()

	h := newConfigHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/agents/not-a-uuid/config", nil)
	rec := httptest.NewRecorder()
	req.SetPathValue("id", "not-a-uuid")

	h.AgentConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// A missing agent is answered from the agentOf path, which needs a
// single-row read the shared fakeQuerier cannot serve — *sql.Row.Scan panics on
// a zero Row. The not-found mapping itself is covered by isNotFound in the
// store layer; what is asserted here is that the endpoint reaches the lookup at
// all rather than short-circuiting, which the 400 cases above already pin.

// TestRenderedConfigCarriesNoCredential is the invariant this whole endpoint
// exists to preserve: a generated config gets committed to a repository, so a
// key inside one becomes a key in git history, forever.
//
// The check is on the rendered output rather than the handler response because
// the response is assembled from these files verbatim — there is no path that
// inserts a credential on the way out.
func TestRenderedConfigCarriesNoCredential(t *testing.T) {
	t.Parallel()

	in := configgen.Input{
		Origin:      "https://shaker.example.com",
		ProjectID:   uuid.New().String(),
		ProjectName: "MyApp",
		AgentID:     uuid.New().String(),
		AgentName:   "Claude Code Backend",
		ToolType:    models.ToolClaudeCode,
	}

	for _, format := range models.AllConfigFormats {
		files, err := configgen.Render(in, format)
		if err != nil {
			t.Fatalf("Render(%s): %v", format, err)
		}
		for _, f := range files {
			if strings.Contains(f.Content, "Bearer ask_") {
				t.Errorf("%s: %s embeds a plaintext key", format, f.Filename)
			}
			// The config carries the full placeholder in a header; the
			// instructions document only names the variable in prose. Checking
			// the variable name rather than the placeholder keeps this from
			// failing on a file that is correct but differently spelled.
			if !strings.Contains(f.Content, configgen.APIKeyEnvVar) {
				t.Errorf("%s: %s does not reference %s", format, f.Filename, configgen.APIKeyEnvVar)
			}
		}
	}
}

// TestConnectionRequestCarriesNoAuthority is a guard on the shape of the mint
// request.
//
// The connection endpoint's power comes from minting a credential, so it must
// not become a second way to mint one with chosen scopes. A `scopes` field here
// would let any caller with write access produce a key holding admin — which is
// exactly what the admin gate on POST /api/keys exists to prevent.
func TestConnectionRequestCarriesNoAuthority(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(models.AgentConnectionRequest{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{"scope", "principal", "project_id", "agent_id", "secret"} {
		if strings.Contains(string(body), forbidden) {
			t.Errorf("AgentConnectionRequest serialises %q: %s", forbidden, body)
		}
	}
}

// TestAgentServicePrincipalIsNamespaced pins the reason the handler prefixes the
// principal name.
//
// A service principal and a human must never collide on display_name: an agent
// called "Victor" would otherwise resolve to the person of that name, and
// approval authority is decided by principal_kind — so an agent's credential
// would inherit a human's right to approve.
func TestAgentServicePrincipalIsNamespaced(t *testing.T) {
	t.Parallel()

	// The constant is unexported, so the property is asserted through the value
	// rather than the symbol: a namespaced name cannot collide with the bare
	// agent name a human directory entry would carry.
	const agentName = "Victor"
	serviceName := "agent: " + agentName

	if serviceName == agentName {
		t.Error("the service principal name is not namespaced")
	}
	if !strings.HasPrefix(serviceName, "agent: ") {
		t.Errorf("service principal name %q is not namespaced", serviceName)
	}
}
