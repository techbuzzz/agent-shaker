package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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

type ContextHandler struct {
	store *queries.ContextsStore
	hub   *websocket.Hub
}

func NewContextHandler(store *queries.ContextsStore, hub *websocket.Hub) *ContextHandler {
	return &ContextHandler{store: store, hub: hub}
}

func (h *ContextHandler) CreateContext(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var req models.CreateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateContextRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}

	c := models.Context{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		AgentID:   req.AgentID,
		TaskID:    req.TaskID,
		Title:     req.Title,
		Content:   req.Content,
		Tags:      pq.StringArray(req.Tags),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.store.CreateContext(r.Context(), &c); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(c.ProjectID, "context_added", c)
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *ContextHandler) ListContexts(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		httpx.WriteError(w, r, fmt.Errorf("%w: project_id query parameter is required", httpx.ErrBadRequest))
		return
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
		return
	}
	var taskID *uuid.UUID
	if t := r.URL.Query().Get("task_id"); t != "" {
		parsed, err := uuid.Parse(t)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("invalid task_id: %w", err))
			return
		}
		taskID = &parsed
	}
	var tags []string
	if tagsParam := r.URL.Query().Get("tags"); tagsParam != "" {
		tags = strings.Split(tagsParam, ",")
	}
	contexts, err := h.store.ListContexts(r.Context(), projectID, taskID, tags)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if contexts == nil {
		contexts = []models.Context{}
	}
	httpx.WriteJSON(w, http.StatusOK, contexts)
}

// SearchContexts runs a full-text search over one project's contexts.
//
// The route is a separate handler rather than a mode of ListContexts because
// the two have different failure modes: ListContexts can safely return a whole
// project, while search is the operation that lets a caller ask for a subset,
// so its limit is enforced here as well as in the store. Keeping them apart
// also means a future search filter cannot silently change what "list" means.
func (h *ContextHandler) SearchContexts(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		httpx.WriteError(w, r, fmt.Errorf("%w: project_id query parameter is required", httpx.ErrBadRequest))
		return
	}
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
		return
	}

	// A missing q is not an error: it means "no results", and answering with
	// every context in the project would be the opposite of what the caller
	// asked for.
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("%w: limit must be an integer", httpx.ErrBadRequest))
			return
		}
		if limit < 0 {
			httpx.WriteError(w, r, fmt.Errorf("%w: limit must not be negative", httpx.ErrBadRequest))
			return
		}
	}

	contexts, err := h.store.SearchContexts(r.Context(), projectID, query, limit)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if contexts == nil {
		contexts = []models.Context{}
	}
	httpx.WriteJSON(w, http.StatusOK, contexts)
}

func (h *ContextHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid context id: %w", err))
		return
	}
	c, err := h.store.GetContext(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("context not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, c)
}

func (h *ContextHandler) UpdateContext(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid context id: %w", err))
		return
	}
	var req models.UpdateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateUpdateContextRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}
	current, err := h.store.GetContext(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("context not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	current.TaskID = req.TaskID
	current.Title = req.Title
	current.Content = req.Content
	current.Tags = pq.StringArray(req.Tags)
	current.UpdatedAt = time.Now()
	if err := h.store.UpdateContext(r.Context(), current); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	updated, err := h.store.GetContext(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(updated.ProjectID, "context_updated", updated)
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func (h *ContextHandler) DeleteContext(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid context id: %w", err))
		return
	}
	projectID, err := h.store.GetContextProjectID(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("context not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.store.DeleteContext(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(projectID, "context_deleted", map[string]any{
		"context_id": id,
		"project_id": projectID,
		"deleted_at": time.Now(),
	})
	w.WriteHeader(http.StatusNoContent)
}
