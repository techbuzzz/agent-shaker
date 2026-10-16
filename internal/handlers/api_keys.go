package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/authkey"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// APIKeysHandler exposes the key-management API.
//
// This is the administrative surface of the service: it mints credentials and
// withdraws them. It is gated by models.ScopeAdmin at the route table (see
// RequireScopeOnPaths in cmd/server/routes.go), so a read-only or write-only
// key can never reach these methods, and the check is not repeated here — a
// second copy of an authorisation rule is a second thing to forget.
type APIKeysHandler struct {
	store *queries.APIKeysStore
	// people is consulted on the mint path so the key's principal_kind comes
	// from the authoritative row instead of the request body.
	people *queries.PeopleStore
	// agents is consulted when a key is bound to one agent: the agent's own
	// project is authoritative, and a key pointing at a row that does not exist
	// would resolve to an identity no tool could act as. May be nil in tests.
	agents *queries.AgentsStore
	// resolver is used only to drop the revoked key from the in-process cache.
	// Without this a revoke would keep working for up to DefaultKeyTTL, which
	// is not what an operator who just clicked "revoke" expects. It may be nil
	// in tests and in a deployment that has not wired the resolver.
	resolver middleware.KeyResolver
}

func NewAPIKeysHandler(store *queries.APIKeysStore, people *queries.PeopleStore, agents *queries.AgentsStore, resolver middleware.KeyResolver) *APIKeysHandler {
	return &APIKeysHandler{store: store, people: people, agents: agents, resolver: resolver}
}

// ListKeys returns the key inventory, newest first.
//
// GET /api/keys?principal_id=<uuid>&agent_id=<uuid>&include_revoked=true
//
// Revoked keys are hidden by default: the default view is the one an operator
// looks at to answer "what is still live", and burying a revoked row in it
// makes that question harder. include_revoked exists for the audit-shaped
// question instead.
//
// agent_id narrows to the credentials bound to one agent, which is how an
// operator rotates a single agent's key without reading the whole inventory.
func (h *APIKeysHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		httpx.WriteError(w, r, fmt.Errorf("key store: %w", httpx.ErrUnavailable))
		return
	}

	var principalID *uuid.UUID
	if raw := r.URL.Query().Get("principal_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("principal_id: %w: %w", httpx.ErrBadRequest, err))
			return
		}
		principalID = &id
	}

	var agentID *uuid.UUID
	if raw := r.URL.Query().Get("agent_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("agent_id: %w: %w", httpx.ErrBadRequest, err))
			return
		}
		agentID = &id
	}

	// Parsed strictly: a malformed flag silently reading as false would hide
	// exactly the rows the caller asked to see.
	includeRevoked := false
	if raw := r.URL.Query().Get("include_revoked"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("include_revoked: %w: %w", httpx.ErrBadRequest, err))
			return
		}
		includeRevoked = parsed
	}

	keys, err := h.store.ListAPIKeys(r.Context(), principalID, agentID, includeRevoked)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("list api keys: %w", err))
		return
	}

	// Defence in depth. models.APIKey already tags KeyHash `json:"-"`, so the
	// digests cannot leave through the serialiser; this strips them anyway so
	// that widening that tag later cannot turn the list endpoint into a
	// credential oracle. A digest is enough to test a guessed key against.
	for i := range keys {
		keys[i].KeyHash = ""
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

// CreateKey mints a credential and returns the plaintext exactly once.
//
// POST /api/keys {"principal_id": "...", "project_id": "...", "scopes": ["read"]}
//
// The plaintext exists in this response and nowhere else — not in the row, not
// in a log, not retrievable by a later GET. That is the property that makes
// the no-store header below load-bearing rather than decorative.
func (h *APIKeysHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		httpx.WriteError(w, r, fmt.Errorf("key store: %w", httpx.ErrUnavailable))
		return
	}

	var req models.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	principalID, err := uuid.Parse(req.PrincipalID)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("principal_id must be a uuid: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	// The kind is copied from the people row rather than accepted from the
	// client, so a caller cannot mint a key that claims to belong to a human
	// while pointing at a service account — which would be a way to launder
	// approval authority (see models.PersonKind.CanApprove).
	principal, err := h.principalOf(r, principalID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	var projectID *uuid.UUID
	if raw := req.ProjectID; raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("project_id must be a uuid: %w: %w", httpx.ErrBadRequest, err))
			return
		}
		projectID = &id
	}

	// An agent-bound key carries its project implicitly: the agent row already
	// knows which project it belongs to, and letting the two disagree would
	// produce a credential scoped to project A that resolves to an agent in
	// project B. project_id is therefore taken from the agent, not the body,
	// whenever agent_id is present.
	var agentID *uuid.UUID
	if raw := strings.TrimSpace(req.AgentID); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("agent_id must be a uuid: %w: %w", httpx.ErrBadRequest, err))
			return
		}
		agent, err := h.agents.GetAgent(r.Context(), id)
		if err != nil {
			if isNotFound(err) {
				httpx.WriteError(w, r, fmt.Errorf("agent_id: no such agent: %w", httpx.ErrNotFound))
				return
			}
			httpx.WriteError(w, r, fmt.Errorf("load agent: %w", err))
			return
		}
		agentID = &id
		projectID = &agent.ProjectID
	}

	scopes, err := models.NormalizeScopes(req.Scopes)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("scopes: %w: %w", httpx.ErrBadRequest, err))
		return
	}
	// An empty scope set means "unrestricted" (models.Allows), so silently
	// accepting a body that omits scopes would mint a wildcard credential from
	// a request that never asked for one. A privileged key must be requested
	// explicitly.
	if len(scopes) == 0 {
		httpx.WriteError(w, r, fmt.Errorf(
			"scopes is required: at least one of %s (read implies nothing, write implies read): %w",
			models.JoinValues(models.AllScopes), httpx.ErrBadRequest))
		return
	}

	secret, digest, display, err := authkey.Generate()
	if err != nil {
		// A broken CSPRNG is not retryable. Issuing from a predictable source
		// would be worse than failing the request, so this surfaces as a 500.
		slog.ErrorContext(r.Context(), "refusing to issue api key", "error", err)
		httpx.WriteError(w, r, fmt.Errorf("could not generate key material"))
		return
	}

	key := models.APIKey{
		ID:            uuid.New(),
		KeyHash:       digest,
		KeyPrefix:     display,
		PrincipalID:   principalID,
		PrincipalType: principal.Kind,
		ProjectID:     projectID,
		AgentID:       agentID,
		Scopes:        scopes,
		CreatedAt:     time.Now().UTC(),
	}
	if err := h.store.CreateAPIKey(r.Context(), &key); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("create api key: %w", err))
		return
	}

	// The response carries the only copy of the secret that will ever exist.
	// Without no-store a proxy or a browser cache may retain a live credential
	// long after the tab is closed.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	// Never log the secret. The prefix and the id are enough to correlate.
	slog.InfoContext(r.Context(), "api key issued",
		"key_id", key.ID.String(),
		"key_prefix", key.KeyPrefix,
		"principal_id", principalID.String(),
		"principal_kind", principal.Kind.String(),
		"scopes", scopes,
		"agent_id", agentID,
	)

	httpx.WriteJSON(w, http.StatusCreated, models.IssuedAPIKey{APIKey: key, Secret: secret})
}

// RevokeKey withdraws a credential.
//
// DELETE /api/keys/{id}
//
// Idempotent by design: revoking an already-revoked key succeeds and returns
// the same row. An operator clicking twice is not an error, and a 409 here
// would teach them to retry blindly.
func (h *APIKeysHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		httpx.WriteError(w, r, fmt.Errorf("key store: %w", httpx.ErrUnavailable))
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("id must be a uuid: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	revoked, err := h.store.RevokeAPIKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			httpx.WriteError(w, r, fmt.Errorf("api key: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("revoke api key: %w", err))
		return
	}

	// Drop the cached decision so the revocation bites on the very next request
	// in this process rather than in up to DefaultKeyTTL seconds. The digest
	// is read from the pre-revoke row, which is why the store returns it.
	if h.resolver != nil {
		h.resolver.InvalidateKey(revoked.KeyHash)
	}

	revoked.KeyHash = ""
	slog.InfoContext(r.Context(), "api key revoked",
		"key_id", revoked.ID.String(),
		"key_prefix", revoked.KeyPrefix,
	)

	httpx.WriteJSON(w, http.StatusOK, revoked)
}

// principalOf loads the people row a key is being minted for.
//
// It takes *http.Request rather than the context alone because it has to shape
// the error envelope, and it uses the People store through the handler so that
// a missing principal is a 404 that names the field rather than a foreign key
// violation surfaced as a 500.
func (h *APIKeysHandler) principalOf(r *http.Request, id uuid.UUID) (models.Person, error) {
	if h.people == nil || !h.people.Available() {
		return models.Person{}, fmt.Errorf("people store: %w", httpx.ErrUnavailable)
	}
	person, err := h.people.GetPerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrPersonNotFound) {
			return models.Person{}, fmt.Errorf("principal_id: no such person: %w", httpx.ErrNotFound)
		}
		return models.Person{}, fmt.Errorf("load principal: %w", err)
	}
	return *person, nil
}
