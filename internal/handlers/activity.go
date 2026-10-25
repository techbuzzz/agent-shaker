package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

// ActivityHandler serves the append-only journal.
//
// It is a read-only surface on purpose. The journal is written by the thing
// that happened, and there is no endpoint here that can change a row — the
// trail is only worth something if nobody can quietly rewrite it.
type ActivityHandler struct {
	store *queries.AuditStore
}

// NewActivityHandler returns a handler over the audit store.
func NewActivityHandler(store *queries.AuditStore) *ActivityHandler {
	return &ActivityHandler{store: store}
}

// ListActivity returns the newest journal rows for a project.
//
// GET /api/activity?project_id=…&entity_type=…&entity_id=…&action=…&limit=…
//
// The project is required. A journal spans every project in the deployment,
// and a reader with no project scope would be reading all of them.
func (h *ActivityHandler) ListActivity(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(r.URL.Query().Get("project_id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: project_id is required", httpx.ErrBadRequest))
		return
	}

	filter := queries.AuditFilter{
		ProjectID:  projectID,
		EntityType: r.URL.Query().Get("entity_type"),
		Action:     r.URL.Query().Get("action"),
	}

	if raw := r.URL.Query().Get("entity_id"); raw != "" {
		entityID, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("%w: entity_id is not a uuid", httpx.ErrBadRequest))
			return
		}
		filter.EntityID = &entityID
	}

	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			httpx.WriteError(w, r, fmt.Errorf("%w: limit must be a positive number", httpx.ErrBadRequest))
			return
		}
		filter.Limit = limit
	}

	events, err := h.store.ListAuditEvents(r.Context(), filter)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"events": events})
}

// ChainHead returns the published head of a project's chain.
//
// GET /api/activity/anchor?project_id=…
//
// The head is the anchor: the value an operator pins outside the database, so
// that a trail rewritten inside the database cannot be rewritten with it. A
// project with no journalled events has no head, and that is reported as an
// empty one rather than as an error.
func (h *ActivityHandler) ChainHead(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectID, err := uuid.Parse(r.URL.Query().Get("project_id"))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: project_id is required", httpx.ErrBadRequest))
		return
	}

	head, err := h.store.AuditChainHead(r.Context(), projectID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if head == nil {
		head = &models.AuditChainHead{ProjectID: projectID}
	}

	httpx.WriteJSON(w, http.StatusOK, head)
}
