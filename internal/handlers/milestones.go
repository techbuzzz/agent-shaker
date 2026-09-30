package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// MilestoneHandler manages CRUD for milestones. Status transitions use
// `CountMilestoneOpenTasks` to enforce the "all linked tasks done before
// milestone can be marked done" rule.
type MilestoneHandler struct {
	store  *queries.MilestonesStore
	tasks  *queries.TasksStore
	agents *queries.AgentsStore
	hub    *websocket.Hub
	db     *database.DB
}

func NewMilestoneHandler(store *queries.MilestonesStore, tasks *queries.TasksStore, agents *queries.AgentsStore, hub *websocket.Hub, db *database.DB) *MilestoneHandler {
	return &MilestoneHandler{store: store, tasks: tasks, agents: agents, hub: hub, db: db}
}

// errMilestoneOpenTasks is returned when the caller tries to close a
// milestone that still has non-terminal tasks linked to it.
type errMilestoneOpenTasks struct{ Open int }

func (e errMilestoneOpenTasks) Error() string {
	return fmt.Sprintf("milestone has %d open task(s); finish or cancel them first", e.Open)
}

// CreateMilestone inserts a new milestone.
func (h *MilestoneHandler) CreateMilestone(w http.ResponseWriter, r *http.Request) {
	var req models.CreateMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateMilestoneRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}

	// Default status
	if req.Status == "" {
		req.Status = models.MilestonePlanned
	}
	if !models.ValidMilestoneStatuses[req.Status] {
		httpx.WriteError(w, r, fmt.Errorf("validation: invalid status %q", req.Status))
		return
	}

	m := models.Milestone{
		ID:          uuid.New(),
		ProjectID:   req.ProjectID,
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      req.Status,
		TargetDate:  req.TargetDate,
		CreatedBy:   req.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.store.Create(r.Context(), &m); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(m.ProjectID, "milestone_added", m)
	httpx.WriteJSON(w, http.StatusCreated, m)
}

// ListMilestones returns every milestone for a project.
func (h *MilestoneHandler) ListMilestones(w http.ResponseWriter, r *http.Request) {
	pidStr := r.URL.Query().Get("project_id")
	if pidStr == "" {
		httpx.WriteError(w, r, fmt.Errorf("project_id query parameter is required"))
		return
	}
	pid, err := uuid.Parse(pidStr)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
		return
	}
	out, err := h.store.ListByProject(r.Context(), pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if out == nil {
		out = []models.Milestone{}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

// GetMilestone returns one milestone by id.
func (h *MilestoneHandler) GetMilestone(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid milestone id: %w", err))
		return
	}
	m, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, m)
}

// UpdateMilestoneStatus transitions a milestone to a new status.
// Closing (→done) requires all linked tasks to be done|cancelled.
func (h *MilestoneHandler) UpdateMilestoneStatus(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid milestone id: %w", err))
		return
	}
	var req models.UpdateMilestoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if !models.ValidMilestoneStatuses[req.Status] {
		httpx.WriteError(w, r, fmt.Errorf("validation: invalid status %q", req.Status))
		return
	}

	current, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if req.Status == models.MilestoneDone {
		open, err := h.tasks.CountMilestoneOpenTasks(r.Context(), id)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if open > 0 {
			httpx.WriteError(w, r, errMilestoneOpenTasks{Open: open})
			return
		}
	}

	updated := *current
	updated.Status = req.Status
	updated.Description = req.Description
	updated.UpdatedAt = time.Now()
	if err := h.store.Update(r.Context(), &updated); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(updated.ProjectID, "milestone_updated", updated)
	httpx.WriteJSON(w, http.StatusOK, updated)
}

// DeleteMilestone removes a milestone. Tasks linked to it are unlinked via
// the ON DELETE SET NULL foreign key.
func (h *MilestoneHandler) DeleteMilestone(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid milestone id: %w", err))
		return
	}
	// Look up first so we know where to broadcast.
	m, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(m.ProjectID, "milestone_deleted", map[string]uuid.UUID{"id": m.ID})
	w.WriteHeader(http.StatusNoContent)
}

// Compile-time guard: errMilestoneOpenTasks is errors.Is-compatible.
var _ error = errMilestoneOpenTasks{}
var _ = errors.New // keep errors import live for future use
