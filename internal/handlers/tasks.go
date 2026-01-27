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
	query := `
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	// Add optional filters
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr != "" {
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			http.Error(w, "Invalid project_id format", http.StatusBadRequest)
			return
		}
		query += " AND project_id = $1"
		args = append(args, projectID)
		argCount++
	}

	agentIDStr := r.URL.Query().Get("agent_id")
	if agentIDStr != "" {
		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			http.Error(w, "Invalid agent_id format", http.StatusBadRequest)
			return
		}
		if argCount == 1 {
			query += " AND assigned_to = $1"
		} else {
			query += " AND assigned_to = $2"
		}
		args = append(args, agentID)
		argCount++
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		switch argCount {
		case 1:
			query += " AND status = $1"
		case 2:
			query += " AND status = $2"
		default:
			query += " AND status = $3"
		}
		args = append(args, status)
		argCount++
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
		var assignedToStr sql.NullString
		var outputStr sql.NullString

		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.CreatedBy, &assignedToStr, &outputStr, &t.CreatedAt, &t.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan task", http.StatusInternalServerError)
			return
		}

		// Handle nullable assigned_to field
		if assignedToStr.Valid {
			if assignedToUUID, err := uuid.Parse(assignedToStr.String); err == nil {
				t.AssignedTo = &assignedToUUID
			}
		} else {
			t.AssignedTo = nil
		}

		// Handle nullable output field
		if outputStr.Valid {
			t.Output = outputStr.String
		} else {
			t.Output = ""
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
	var assignedToStr sql.NullString
	var outputStr sql.NullString

	err = h.db.QueryRow(`
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.CreatedBy, &assignedToStr, &outputStr, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	// Handle nullable assigned_to field
	if assignedToStr.Valid {
		if assignedToUUID, err := uuid.Parse(assignedToStr.String); err == nil {
			task.AssignedTo = &assignedToUUID
		}
	} else {
		task.AssignedTo = nil
	}

	// Handle nullable output field
	if outputStr.Valid {
		task.Output = outputStr.String
	} else {
		task.Output = ""
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
	var assignedToStr sql.NullString
	var outputStr sql.NullString

	err = h.db.QueryRow(`
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.CreatedBy, &assignedToStr, &outputStr, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	// Handle nullable assigned_to field
	if assignedToStr.Valid {
		if assignedToUUID, err := uuid.Parse(assignedToStr.String); err == nil {
			task.AssignedTo = &assignedToUUID
		}
	} else {
		task.AssignedTo = nil
	}

	// Handle nullable output field
	if outputStr.Valid {
		task.Output = outputStr.String
	} else {
		task.Output = ""
	}

	// Broadcast task update
	h.hub.BroadcastToProject(task.ProjectID, "task_update", task)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	// Validate status
	validStatuses := []string{"pending", "in_progress", "completed", "failed", "blocked"}
	valid := false
	for _, s := range validStatuses {
		if req.Status == s {
			valid = true
			break
		}
	}
	if !valid {
		http.Error(w, "invalid status value", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
		UPDATE tasks
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, req.Status, time.Now(), id)
	if err != nil {
		http.Error(w, "Failed to update task status", http.StatusInternalServerError)
		return
	}

	// Get updated task
	var task models.Task
	var assignedToStr sql.NullString
	var outputStr sql.NullString

	err = h.db.QueryRow(`
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.CreatedBy, &assignedToStr, &outputStr, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	// Handle nullable assigned_to field
	if assignedToStr.Valid {
		if assignedToUUID, err := uuid.Parse(assignedToStr.String); err == nil {
			task.AssignedTo = &assignedToUUID
		}
	} else {
		task.AssignedTo = nil
	}

	// Handle nullable output field
	if outputStr.Valid {
		task.Output = outputStr.String
	} else {
		task.Output = ""
	}

	// Broadcast task update
	h.hub.BroadcastToProject(task.ProjectID, "task_update", task)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Get task to retrieve project_id for WebSocket broadcast
	var task models.Task
	err = tx.QueryRow("SELECT id, project_id FROM tasks WHERE id = $1", id).Scan(&task.ID, &task.ProjectID)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	// Delete related contexts first
	_, err = tx.Exec("DELETE FROM contexts WHERE task_id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete related contexts", http.StatusInternalServerError)
		return
	}

	// Delete the task
	result, err := tx.Exec("DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Broadcast task deletion
	h.hub.BroadcastToProject(task.ProjectID, "task_deleted", map[string]interface{}{
		"task_id":    id,
		"project_id": task.ProjectID,
		"deleted_at": time.Now(),
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) ReassignTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid task ID format", http.StatusBadRequest)
		return
	}

	var req models.ReassignTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify the new agent exists
	var agentExists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM agents WHERE id = $1)", req.AssignedTo).Scan(&agentExists)
	if err != nil {
		http.Error(w, "Failed to verify agent", http.StatusInternalServerError)
		return
	}
	if !agentExists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	_, err = h.db.Exec(`
		UPDATE tasks
		SET assigned_to = $1, updated_at = $2
		WHERE id = $3
	`, req.AssignedTo, time.Now(), id)
	if err != nil {
		http.Error(w, "Failed to reassign task", http.StatusInternalServerError)
		return
	}

	// Get updated task
	var task models.Task
	var assignedToStr sql.NullString
	var outputStr sql.NullString

	err = h.db.QueryRow(`
		SELECT id, project_id, title, description, status, priority, created_by, assigned_to, output, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`, id).Scan(&task.ID, &task.ProjectID, &task.Title, &task.Description, &task.Status, &task.Priority, &task.CreatedBy, &assignedToStr, &outputStr, &task.CreatedAt, &task.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve task", http.StatusInternalServerError)
		return
	}

	// Handle nullable assigned_to field
	if assignedToStr.Valid {
		if assignedToUUID, err := uuid.Parse(assignedToStr.String); err == nil {
			task.AssignedTo = &assignedToUUID
		}
	} else {
		task.AssignedTo = nil
	}

	// Handle nullable output field
	if outputStr.Valid {
		task.Output = outputStr.String
	} else {
		task.Output = ""
	}

	// Broadcast task reassignment
	h.hub.BroadcastToProject(task.ProjectID, "task_reassigned", task)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
