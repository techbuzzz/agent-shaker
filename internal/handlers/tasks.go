package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type TaskHandler struct {
	store *queries.TasksStore
	hub   *websocket.Hub
}

func NewTaskHandler(store *queries.TasksStore, hub *websocket.Hub) *TaskHandler {
	return &TaskHandler{store: store, hub: hub}
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
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
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
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
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
	if err := h.store.ReassignTask(r.Context(), id, req.AssignedTo); err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	task, err := h.store.GetTask(r.Context(), id)
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
