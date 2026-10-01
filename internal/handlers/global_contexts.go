package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// GlobalContextHandler manages CRUD for the global_contexts table. Scope
// rules: scope='global'  ⇒ no project_id; scope='project' ⇒ project_id
// required. The DB enforces the same rule via CHECK constraint; the
// handler enforces it earlier so the API surfaces a 400, not a 500.
type GlobalContextHandler struct {
	store *queries.GlobalContextsStore
	hub   *websocket.Hub
}

func NewGlobalContextHandler(store *queries.GlobalContextsStore, hub *websocket.Hub) *GlobalContextHandler {
	return &GlobalContextHandler{store: store, hub: hub}
}

// errGlobalContextConflict is returned when a (scope, title) row already
// exists. Mirrors the DB unique-index violation but presented as a 409.
type errGlobalContextConflict struct{ Title string }

func (e errGlobalContextConflict) Error() string {
	return fmt.Sprintf("a global context with title %q already exists", e.Title)
}

func (h *GlobalContextHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGlobalContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateGlobalContextRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}

	g := models.GlobalContext{
		ID:        uuid.New(),
		Scope:     req.Scope,
		ProjectID: req.ProjectID,
		AgentID:   req.AgentID,
		Title:     strings.TrimSpace(req.Title),
		Content:   req.Content,
		Tags:      pq.StringArray(req.Tags),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.store.Create(r.Context(), &g); err != nil {
		// Postgres unique_violation = SQLSTATE 23505.
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			httpx.WriteError(w, r, errGlobalContextConflict{Title: g.Title})
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	if g.Scope == models.ScopeProject && g.ProjectID != nil {
		h.hub.BroadcastToProject(*g.ProjectID, "global_context_added", g)
	}
	httpx.WriteJSON(w, http.StatusCreated, g)
}

func (h *GlobalContextHandler) List(w http.ResponseWriter, r *http.Request) {
	scope := models.GlobalContextScope(r.URL.Query().Get("scope"))
	if scope != "" && !models.ValidGlobalContextScopes[scope] {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid scope %q", httpx.ErrBadRequest, scope))
		return
	}
	var pid *uuid.UUID
	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		parsed, err := uuid.Parse(pidStr)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
			return
		}
		pid = &parsed
	}
	tagPrefix := r.URL.Query().Get("tag_prefix")
	out, err := h.store.List(r.Context(), scope, pid, tagPrefix)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if out == nil {
		out = []models.GlobalContext{}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *GlobalContextHandler) Get(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid global_context id: %w", err))
		return
	}
	g, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, g)
}

func (h *GlobalContextHandler) Update(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid global_context id: %w", err))
		return
	}
	var req models.UpdateGlobalContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	current, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	current.Content = req.Content
	current.Tags = pq.StringArray(req.Tags)
	current.UpdatedAt = time.Now()
	if err := h.store.Update(r.Context(), current); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if current.Scope == models.ScopeProject && current.ProjectID != nil {
		h.hub.BroadcastToProject(*current.ProjectID, "global_context_updated", current)
	}
	httpx.WriteJSON(w, http.StatusOK, current)
}

func (h *GlobalContextHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid global_context id: %w", err))
		return
	}
	g, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if g.Scope == models.ScopeProject && g.ProjectID != nil {
		h.hub.BroadcastToProject(*g.ProjectID, "global_context_deleted", map[string]uuid.UUID{"id": g.ID})
	}
	w.WriteHeader(http.StatusNoContent)
}

// Compile-time guards.
var _ error = errGlobalContextConflict{}
var _ = errors.New
