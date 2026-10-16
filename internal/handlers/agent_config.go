package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/authkey"
	"github.com/techbuzzz/agent-shaker/internal/configgen"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/publicurl"
)

// AgentConfigHandler serves the generated MCP configuration for one agent.
//
// It is a separate handler from AgentHandler because it needs three collaborators
// AgentHandler does not have — the key store, the people directory and the
// configured public origin — and because its two endpoints are a different
// concern from CRUD: one of them mints a credential, which is not something the
// agent CRUD surface should be able to do as a side effect of an update.
type AgentConfigHandler struct {
	agents   *queries.AgentsStore
	keys     *queries.APIKeysStore
	people   *queries.PeopleStore
	projects *queries.ProjectsStore

	// baseURL is the operator's BASE_URL, read once at startup. Empty means
	// "derive the origin from each request", which is correct behind a proxy
	// that sets X-Forwarded-* and wrong for a deployment exposed directly —
	// which is why it is an explicit setting rather than a guess.
	baseURL string
}

func NewAgentConfigHandler(agents *queries.AgentsStore, keys *queries.APIKeysStore, people *queries.PeopleStore, projects *queries.ProjectsStore, baseURL string) *AgentConfigHandler {
	return &AgentConfigHandler{agents: agents, keys: keys, people: people, projects: projects, baseURL: baseURL}
}

// agentServicePrefix namespaces the service principal a generated key hangs on.
//
// A prefixed name rather than the bare agent name, because the people directory
// also holds humans and an agent called "Victor" must not resolve to the person
// who happens to share the name — that would give an agent's credential the
// principal_kind of a human, and approval authority is decided by that kind.
const agentServicePrefix = "agent: "

// AgentConfig renders the connection files for one agent.
//
// GET /api/agents/{id}/config?format=claude|codex|cursor|generic
//
// No secret is ever in the response. Every credential is the
// ${env:AGENT_SHAKER_API_KEY} placeholder, which is what makes a downloaded
// file safe to commit; the browser substitutes the plaintext it is already
// holding when the operator copies it out of the connection modal.
//
// The format defaults to the one matching the agent's tool_type. An agent whose
// tool type this server cannot generate for still gets a file — the generic
// `.mcp.json` is what most MCP hosts read — rather than a refusal, because a
// working config for the wrong client beats a 400 that explains nothing about
// what to do instead.
func (h *AgentConfigHandler) AgentConfig(w http.ResponseWriter, r *http.Request) {
	if !h.agents.Available() {
		handleNoStore(w, r)
		return
	}

	// A named format is validated before anything is loaded. It costs nothing to
	// check, it answers the client without a database round trip, and it means a
	// typo in `?format=` is reported as the typo rather than as whatever the
	// agent lookup would have said first.
	requested := strings.TrimSpace(r.URL.Query().Get("format"))
	if requested != "" {
		if _, err := models.ParseConfigFormat(requested); err != nil {
			// The message enumerates what this server does support, so a client
			// can offer the choices rather than showing a bare validation error.
			httpx.WriteError(w, r, fmt.Errorf("%w: %w", httpx.ErrBadRequest, err))
			return
		}
	}

	agent, ok := h.agentOf(w, r)
	if !ok {
		return
	}

	in, ok := h.inputFor(w, r, agent)
	if !ok {
		return
	}

	format, err := in.Format(requested)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: %w", httpx.ErrBadRequest, err))
		return
	}

	files, err := configgen.Render(in, format)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("render config: %w", err))
		return
	}

	// no-store even though there is no secret here: the body embeds project and
	// agent identifiers, and a cached copy of "how is this deployment wired"
	// outliving the deployment is not something to leave lying around.
	w.Header().Set("Cache-Control", "no-store")

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"agent":     agent,
		"format":    format,
		"files":     files,
		"env_var":   configgen.APIKeyEnvVar,
		"endpoints": map[string]string{"mcp": in.MCPURL()},
	})
}

// ConnectAgent mints a credential bound to one agent and renders its config.
//
// POST /api/agents/{id}/connection {"format": "claude"}
//
// The response is the only place the plaintext exists besides the browser ref
// that holds it; the server keeps the hash, exactly as POST /api/keys does.
//
// Not idempotent, on purpose. One POST is one credential. A client that retries
// after a timeout gets a second key rather than the first one back, because
// making this idempotent would mean either caching a plaintext the server does
// not keep, or storing enough state to reproduce one. Both are worse than the
// operator seeing two keys in the inventory and revoking the one they did not
// use. The UI says so before it sends the request.
func (h *AgentConfigHandler) ConnectAgent(w http.ResponseWriter, r *http.Request) {
	if !h.agents.Available() || !h.keys.Available() {
		handleNoStore(w, r)
		return
	}

	var req models.AgentConnectionRequest
	// An absent or empty body is fine: the only field is optional, and
	// requiring `{}` would make curl invocations awkward for no benefit. An
	// empty body decodes as io.EOF, which is not a malformed request.
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
			return
		}
	}

	// Validated before the agent is loaded, and before anything is minted.
	// Order matters here in a way it does not on the read endpoint: a request
	// that cannot be honoured must fail before it creates a credential, or the
	// operator retries with a corrected format and is left holding two live
	// keys for a connection that only ever worked once.
	requested := strings.TrimSpace(req.Format)
	if requested != "" {
		if _, err := models.ParseConfigFormat(requested); err != nil {
			httpx.WriteError(w, r, fmt.Errorf("%w: %w", httpx.ErrBadRequest, err))
			return
		}
	}

	agent, ok := h.agentOf(w, r)
	if !ok {
		return
	}

	in, ok := h.inputFor(w, r, agent)
	if !ok {
		return
	}

	format, err := in.Format(requested)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: %w", httpx.ErrBadRequest, err))
		return
	}
	if !agent.ToolType.SupportsConfigGeneration() && requested == "" {
		// Falling back to generic is only right when the caller did not ask for
		// something specific. An agent whose tool has no dedicated layout yet
		// still gets a usable file — the generic `.mcp.json` is what most hosts
		// read — and the log line is how an operator finds out which agent is
		// waiting for Phase 3.
		slog.InfoContext(r.Context(), "no dedicated config format for tool type, rendering generic",
			"agent_id", agent.ID.String(), "tool_type", agent.ToolType)
	}

	principal, err := h.servicePrincipalFor(r, agent)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	secret, digest, display, err := authkey.Generate()
	if err != nil {
		slog.ErrorContext(r.Context(), "refusing to issue agent key", "error", err)
		httpx.WriteError(w, r, fmt.Errorf("could not generate key material"))
		return
	}

	key := models.APIKey{
		ID:            uuid.New(),
		KeyHash:       digest,
		KeyPrefix:     display,
		PrincipalID:   principal.ID,
		PrincipalType: principal.Kind,
		ProjectID:     &agent.ProjectID,
		AgentID:       &agent.ID,
		// read + write, and never admin. The admin scope mints and revokes
		// credentials; an agent holding it could mint a key for any other
		// agent and impersonate it, which is the whole thing the binding is
		// meant to prevent.
		Scopes:    []models.Scope{models.ScopeRead, models.ScopeWrite},
		CreatedAt: time.Now().UTC(),
	}
	if err := h.keys.CreateAPIKey(r.Context(), &key); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("create agent key: %w", err))
		return
	}

	files, err := configgen.Render(in, format)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("render config: %w", err))
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	// The prefix and the ids are enough to correlate. The secret never reaches
	// the log — a connection is exactly the event someone would want a
	// complete audit line for, and exactly the event that must not produce a
	// replayable credential in a log aggregator.
	slog.InfoContext(r.Context(), "agent key issued",
		"key_id", key.ID.String(),
		"key_prefix", key.KeyPrefix,
		"agent_id", agent.ID.String(),
		"project_id", agent.ProjectID.String(),
		"format", format,
	)

	httpx.WriteJSON(w, http.StatusCreated, models.AgentConnectionResponse{
		Key:    models.IssuedAPIKey{APIKey: key, Secret: secret},
		Agent:  *agent,
		Format: format,
		Files:  files,
	})
}

// agentOf resolves the {id} path value and loads the row, writing the error
// envelope itself. The bool is false when the handler has already answered.
func (h *AgentConfigHandler) agentOf(w http.ResponseWriter, r *http.Request) (*models.Agent, bool) {
	id, err := uuid.Parse(muxVars(r)["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid agent id: %v", httpx.ErrBadRequest, err))
		return nil, false
	}
	agent, err := h.agents.GetAgent(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w: %w", httpx.ErrNotFound, err))
			return nil, false
		}
		httpx.WriteError(w, r, err)
		return nil, false
	}
	return agent, true
}

// inputFor builds the generator input, loading the project name so the generated
// document reads as "Project **MyApp**" rather than a UUID.
func (h *AgentConfigHandler) inputFor(w http.ResponseWriter, r *http.Request, agent *models.Agent) (configgen.Input, bool) {
	// The project id is a UUID that came out of a row, so it needs no
	// validation — but it is still spliced into a URL handed to somebody else's
	// process, and a value that escaped its type would do that quietly.
	//
	// An empty origin is not an error: it means no trustworthy host could be
	// determined, and a config with a relative /mcp is still usable by a client
	// on the same origin. One advertising a guessed host is not.
	origin := publicurl.Resolve(h.baseURL, r)

	return configgen.Input{
		Origin:      origin,
		ProjectID:   agent.ProjectID.String(),
		ProjectName: h.projectName(r, agent.ProjectID),
		AgentID:     agent.ID.String(),
		AgentName:   agent.Name,
		ToolType:    agent.ToolType,
	}, true
}

// projectName looks up the display name, returning "" when it cannot be read.
//
// Best effort on purpose. The generated document is still correct without the
// name — it shows the UUID — and failing the whole endpoint because a
// presentation detail is missing would be a bad trade.
func (h *AgentConfigHandler) projectName(r *http.Request, id uuid.UUID) string {
	if h.projects == nil || !h.projects.Available() {
		return ""
	}
	project, err := h.projects.GetProject(r.Context(), id)
	if err != nil {
		return ""
	}
	return project.Name
}

// servicePrincipalFor finds or creates the service identity a key hangs on.
//
// Find-or-create rather than create, because minting a second key for the same
// agent must not litter the people directory with a new row every time — the
// directory is a thing operators read.
//
// There is deliberately no unique constraint on (kind, display_name). A
// concurrent pair of requests can both miss the lookup and both insert, and both
// service rows are valid principals; api_keys.agent_id, not the principal, is
// what actually binds a credential. Adding the constraint later would need a
// de-duplication migration against real data, so it is not smuggled in here.
func (h *AgentConfigHandler) servicePrincipalFor(r *http.Request, agent *models.Agent) (*models.Person, error) {
	if h.people == nil || !h.people.Available() {
		return nil, fmt.Errorf("people store: %w", httpx.ErrUnavailable)
	}

	name := agentServicePrefix + agent.Name
	person, err := h.people.GetServiceByDisplayName(r.Context(), name)
	if err == nil {
		return person, nil
	}
	if !errors.Is(err, models.ErrPersonNotFound) {
		return nil, fmt.Errorf("look up agent principal: %w", err)
	}

	created := &models.Person{
		ID:          uuid.New(),
		Kind:        models.PersonKindService,
		DisplayName: name,
		CreatedAt:   time.Now().UTC(),
	}
	if err := h.people.CreatePerson(r.Context(), created); err != nil {
		return nil, fmt.Errorf("create agent principal: %w", err)
	}
	return created, nil
}
