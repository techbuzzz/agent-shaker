package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// ProjectHandler manages CRUD for the projects resource.
type ProjectHandler struct {
	store *queries.ProjectsStore
	hub   *websocket.Hub
}

// NewProjectHandler returns a handler backed by the supplied store.
func NewProjectHandler(store *queries.ProjectsStore, hub *websocket.Hub) *ProjectHandler {
	return &ProjectHandler{store: store, hub: hub}
}

// CreateProject decodes, validates, and inserts a new project.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateProjectRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}

	project := models.Project{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.store.CreateProject(r.Context(), &project); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, project)
}

// ListProjects returns every project as a JSON array (empty array, never
// null).
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.store.ListProjects(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if projects == nil {
		projects = []models.Project{}
	}
	httpx.WriteJSON(w, http.StatusOK, projects)
}

// GetProject returns one project by id.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project id: %w", err))
		return
	}
	project, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("project not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, project)
}

// UpdateProjectStatus validates a status value and updates the row.
func (h *ProjectHandler) UpdateProjectStatus(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project id: %w", err))
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if !validProjectStatus(req.Status) {
		httpx.WriteError(w, r, fmt.Errorf("invalid status; expected active, completed or archived"))
		return
	}
	if err := h.store.UpdateProjectStatus(r.Context(), id, req.Status); err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("project not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	project, err := h.store.GetProject(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	// Broadcast the status change over the WebSocket fan-out so connected
	// dashboards see the update without a poll.
	h.hub.BroadcastToProject(id, "project_status_update", project)
	httpx.WriteJSON(w, http.StatusOK, project)
}

// DeleteProject removes a project and its related rows in one transaction.
// The transaction lives in this handler (not the query store) because it
// touches three tables (contexts, tasks, agents, projects) — keeping the
// orchestration here avoids leaking a half-deleted state.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project id: %w", err))
		return
	}

	// Delegate the multi-table delete to the projects query store. The
	// store takes a Querier; we hand it a *sql.Tx here so the whole delete
	// is atomic.
	tx, err := h.store.BeginTx(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := h.store.DeleteProjectCascade(r.Context(), tx, id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(id, "project_deleted", map[string]any{
		"project_id": id,
		"deleted_at": time.Now(),
	})
	w.WriteHeader(http.StatusNoContent)
}

// validProjectStatus returns true when s is one of the documented status
// values. Centralised so validation and the typed query enum stay aligned.
func validProjectStatus(s string) bool {
	switch strings.ToLower(s) {
	case "active", "completed", "archived":
		return true
	default:
		return false
	}
}

// isNotFound matches the sentinel strings produced by the query helpers
// when a row is missing. Kept local because the helpers do not yet return a
// typed sentinel (that follow-up is part of the broader M4 work).
func isNotFound(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "not found"))
}
