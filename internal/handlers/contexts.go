package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type ContextHandler struct {
	db  *database.DB
	hub *websocket.Hub
}

func NewContextHandler(db *database.DB, hub *websocket.Hub) *ContextHandler {
	return &ContextHandler{db: db, hub: hub}
}

func (h *ContextHandler) CreateContext(w http.ResponseWriter, r *http.Request) {
	var req models.CreateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateCreateContextRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := models.Context{
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

	_, err := h.db.Exec(`
		INSERT INTO contexts (id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, ctx.ID, ctx.ProjectID, ctx.AgentID, ctx.TaskID, ctx.Title, ctx.Content, ctx.Tags, ctx.CreatedAt, ctx.UpdatedAt)
	if err != nil {
		http.Error(w, "Failed to create context", http.StatusInternalServerError)
		return
	}

	// Broadcast context creation
	h.hub.BroadcastToProject(ctx.ProjectID, "context_added", ctx)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ctx)
}

func (h *ContextHandler) ListContexts(w http.ResponseWriter, r *http.Request) {
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
		SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at
		FROM contexts
		WHERE project_id = $1
	`
	args := []interface{}{projectID}

	// Add tag filter
	tagsParam := r.URL.Query().Get("tags")
	if tagsParam != "" {
		tags := strings.Split(tagsParam, ",")
		query += " AND tags && $2"
		args = append(args, pq.Array(tags))
	}

	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve contexts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var contexts []models.Context
	for rows.Next() {
		var c models.Context
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.AgentID, &c.TaskID, &c.Title, &c.Content, &c.Tags, &c.CreatedAt, &c.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan context", http.StatusInternalServerError)
			return
		}
		contexts = append(contexts, c)
	}

	// Return empty array instead of null
	if contexts == nil {
		contexts = []models.Context{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contexts)
}

func (h *ContextHandler) GetContext(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid context ID format", http.StatusBadRequest)
		return
	}

	var ctx models.Context
	err = h.db.QueryRow(`
		SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at
		FROM contexts
		WHERE id = $1
	`, id).Scan(&ctx.ID, &ctx.ProjectID, &ctx.AgentID, &ctx.TaskID, &ctx.Title, &ctx.Content, &ctx.Tags, &ctx.CreatedAt, &ctx.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Context not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve context", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ctx)
}

func (h *ContextHandler) UpdateContext(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid context ID format", http.StatusBadRequest)
		return
	}

	var req models.UpdateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateUpdateContextRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if context exists and get current data
	var currentCtx models.Context
	err = h.db.QueryRow(`
		SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at
		FROM contexts
		WHERE id = $1
	`, id).Scan(&currentCtx.ID, &currentCtx.ProjectID, &currentCtx.AgentID, &currentCtx.TaskID, &currentCtx.Title, &currentCtx.Content, &currentCtx.Tags, &currentCtx.CreatedAt, &currentCtx.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Context not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve context", http.StatusInternalServerError)
		return
	}

	// Update the context
	_, err = h.db.Exec(`
		UPDATE contexts
		SET task_id = $1, title = $2, content = $3, tags = $4, updated_at = $5
		WHERE id = $6
	`, req.TaskID, req.Title, req.Content, pq.Array(req.Tags), time.Now(), id)
	if err != nil {
		http.Error(w, "Failed to update context", http.StatusInternalServerError)
		return
	}

	// Get updated context
	var updatedCtx models.Context
	err = h.db.QueryRow(`
		SELECT id, project_id, agent_id, task_id, title, content, tags, created_at, updated_at
		FROM contexts
		WHERE id = $1
	`, id).Scan(&updatedCtx.ID, &updatedCtx.ProjectID, &updatedCtx.AgentID, &updatedCtx.TaskID, &updatedCtx.Title, &updatedCtx.Content, &updatedCtx.Tags, &updatedCtx.CreatedAt, &updatedCtx.UpdatedAt)
	if err != nil {
		http.Error(w, "Failed to retrieve updated context", http.StatusInternalServerError)
		return
	}

	// Broadcast context update
	h.hub.BroadcastToProject(updatedCtx.ProjectID, "context_updated", updatedCtx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedCtx)
}

func (h *ContextHandler) DeleteContext(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid context ID format", http.StatusBadRequest)
		return
	}

	// Check if context exists and get project_id for broadcasting
	var projectID uuid.UUID
	err = h.db.QueryRow(`
		SELECT project_id FROM contexts WHERE id = $1
	`, id).Scan(&projectID)
	if err == sql.ErrNoRows {
		http.Error(w, "Context not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve context", http.StatusInternalServerError)
		return
	}

	// Delete the context
	result, err := h.db.Exec(`DELETE FROM contexts WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "Failed to delete context", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "Failed to confirm deletion", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Context not found", http.StatusNotFound)
		return
	}

	// Broadcast context deletion
	h.hub.BroadcastToProject(projectID, "context_deleted", map[string]interface{}{
		"id": id,
	})

	w.WriteHeader(http.StatusNoContent)
}
