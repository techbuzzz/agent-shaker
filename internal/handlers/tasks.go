package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/techbuzzz/agent-shaker/internal/coordination"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type TaskHandler struct {
	store *queries.TasksStore
	hub   *websocket.Hub
	// agents is used to check that an assignee belongs to the task's project
	// before writing it; see ReassignTask.
	agents *queries.AgentsStore
	// resolver promotes dependent tasks when a predecessor finishes. It is nil
	// when the server runs without a database.
	resolver *coordination.DependencyResolver
}

// NewTaskHandler builds the task handler. The resolver is optional: a server
// without a database has no dependencies to resolve, and a nil resolver must
// degrade to "just update the status" rather than panic.
func NewTaskHandler(store *queries.TasksStore, agents *queries.AgentsStore, hub *websocket.Hub, resolver *coordination.DependencyResolver) *TaskHandler {
	return &TaskHandler{store: store, agents: agents, hub: hub, resolver: resolver}
}

// resolveDependents runs the dependency resolver after a task reaches done.
//
// This is here, not only in the MCP complete_task tool, because a task can be
// finished from the dashboard or the A2A surface too. When only the tool did
// it, a task completed from the UI left every task that depended on it blocked
// forever — the one failure the whole dependency mechanism exists to prevent,
// and one with no symptom anywhere except a task that never appears in the
// ready list.
func (h *TaskHandler) resolveDependents(r *http.Request, taskID uuid.UUID) {
	if h.resolver == nil {
		return
	}
	promoted, err := h.resolver.ResolveAfterCompletion(r.Context(), taskID)
	if err != nil {
		// The task itself is already saved. Failing the whole request here would
		// tell the caller their update did not happen, which is worse than a
		// dependency that stays blocked until the next completion touches it.
		slog.Error("resolving dependents failed", "task_id", taskID, "error", err)
		return
	}
	for _, id := range promoted {
		if task, err := h.store.GetTask(r.Context(), id); err == nil {
			h.hub.BroadcastToProject(task.ProjectID, "task_ready", map[string]any{
				"task_id": task.ID, "project_id": task.ProjectID, "title": task.Title,
				"status": string(models.StatusTaskReady),
			})
		}
	}
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateTaskRequest(&req); err != nil {
		httpx.WriteValidationError(w, r, err)
		return
	}
	tags := pq.StringArray(req.Tags)
	task := models.Task{
		ID:          uuid.New(),
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.TaskStatus("pending"),
		Priority:    req.Priority,
		CreatedBy:   req.CreatedBy,
		AssignedTo:  req.AssignedTo,
		Output:      "",
		MilestoneID: req.MilestoneID,
		Tags:        tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := h.store.CreateTask(r.Context(), &task); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(task.ProjectID, "task_added", task)
	httpx.WriteJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var projectID *uuid.UUID
	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
			return
		}
		projectID = &pid
	}
	tasks, err := h.store.ListTasks(r.Context(), projectID, r.URL.Query().Get("status"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}
	httpx.WriteJSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid task id: %w", err))
		return
	}
	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid task id: %w", err))
		return
	}
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateTaskRequest(&req); err != nil {
		httpx.WriteValidationError(w, r, err)
		return
	}
	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	task.Title = req.Title
	task.Description = req.Description
	task.Priority = req.Priority
	task.AssignedTo = req.AssignedTo
	task.UpdatedAt = time.Now()
	if err := h.store.UpdateTask(r.Context(), task); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid task id: %w", err))
		return
	}
	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := h.store.UpdateTaskStatus(r.Context(), id, string(req.Status)); err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	// Reaching done is the transition the dependency resolver keys on.
	if req.Status == models.StatusTaskDone {
		h.resolveDependents(r, id)
	}
	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) ReassignTask(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid task id: %w", err))
		return
	}
	var req models.ReassignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}

	// The assignee must be an agent of THIS project.
	//
	// Reassign used to write the id straight through. The foreign key on
	// tasks.assigned_to only proves the agent exists, not that it lives in the
	// same project — so a task could end up owned by an agent belonging to
	// another project. Nothing complains at that moment; the damage shows up
	// later and far away: raising a blocker for that task 404s ("agent not
	// found"), the dashboard's agent filter hides it, and the agent's own
	// heartbeat publishes into a project it does not belong to.
	//
	// 404 rather than 400, because from this task's point of view the agent is
	// not reachable — the same language the other project-scoped reads use.
	task, err := h.store.GetTask(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	assignee, err := h.agents.GetAgent(r.Context(), req.AssignedTo)
	if err != nil || assignee.ProjectID != task.ProjectID {
		httpx.WriteError(w, r, fmt.Errorf("%w: agent %s is not available in project %s",
			httpx.ErrNotFound, req.AssignedTo, task.ProjectID))
		return
	}

	if err := h.store.ReassignTask(r.Context(), id, req.AssignedTo); err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	task, err = h.store.GetTask(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid task id: %w", err))
		return
	}
	if err := h.store.DeleteTask(r.Context(), id); err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
