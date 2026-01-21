package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type TaskHandler struct {
	db  *database.DB
	hub *websocket.Hub
}

func NewTaskHandler(db *database.DB, hub *websocket.Hub) *TaskHandler {
	return &TaskHandler{db: db, hub: hub}
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateCreateTaskRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default priority if not provided
	if req.Priority == "" {
		req.Priority = "medium"
	}

	task := models.Task{
		ID:          uuid.New(),
		ProjectID:   req.ProjectID,
		Title:       req.Title,
		Description: req.Description,
		Status:      "pending",
		Priority:    req.Priority,
		CreatedBy:   req.CreatedBy,
		AssignedTo:  req.AssignedTo,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := h.db.Exec(`
		INSERT INTO tasks (id, project_id, title, description, status, priority, created_by, assigned_to, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, task.ID, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.CreatedBy, task.AssignedTo, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	// Broadcast task creation
	h.hub.BroadcastToProject(task.ProjectID, "task_update", task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, "Invalid project_id format", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE project_id = $1
	`
	args := []interface{}{projectID}

	// Add filters
	status := r.URL.Query().Get("status")
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	assignedToStr := r.URL.Query().Get("assigned_to")
	if assignedToStr != "" {
		assignedTo, err := uuid.Parse(assignedToStr)
		if err == nil {
			if status != "" {
				query += " AND assigned_to = $3"
			} else {
				query += " AND assigned_to = $2"
			}
			args = append(args, assignedTo)
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.CreatedBy, &t.AssignedTo, &t.Output, &t.CreatedAt, &t.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan task", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}

	// Return empty array instead of null
	if tasks == nil {
		tasks = []models.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	var task models.Task
	err = h.db.QueryRow(`
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.CreatedBy, &task.AssignedTo, &task.Output, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	var req models.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateUpdateTaskRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
		UPDATE tasks
		SET status = $1, output = $2, updated_at = $3
		WHERE id = $4
	`, req.Status, req.Output, time.Now(), id)
	if err != nil {
		http.Error(w, "Failed to update task", http.StatusInternalServerError)
		return
	}

	// Get updated task
	var task models.Task
	err = h.db.QueryRow(`
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.CreatedBy, &task.AssignedTo, &task.Output, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	// Broadcast task update
	h.hub.BroadcastToProject(task.ProjectID, "task_update", task)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
