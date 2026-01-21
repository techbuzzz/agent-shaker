package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/db"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type Handler struct {
	db  *db.Database
	hub *websocket.Hub
}

// NewRouter creates a new HTTP router with all endpoints
func NewRouter(database *db.Database, hub *websocket.Hub) *mux.Router {
	h := &Handler{db: database, hub: hub}
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	
	// Project endpoints
	api.HandleFunc("/projects", h.CreateProject).Methods("POST")
	api.HandleFunc("/projects", h.ListProjects).Methods("GET")
	api.HandleFunc("/projects/{id}", h.GetProject).Methods("GET")

	// Agent endpoints
	api.HandleFunc("/agents", h.CreateAgent).Methods("POST")
	api.HandleFunc("/projects/{project_id}/agents", h.ListAgents).Methods("GET")
	api.HandleFunc("/agents/{id}", h.GetAgent).Methods("GET")

	// Task endpoints
	api.HandleFunc("/tasks", h.CreateTask).Methods("POST")
	api.HandleFunc("/tasks/{id}", h.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}/status", h.UpdateTaskStatus).Methods("PUT")
	api.HandleFunc("/agents/{agent_id}/tasks", h.ListTasksByAgent).Methods("GET")
	api.HandleFunc("/projects/{project_id}/tasks", h.ListTasksByProject).Methods("GET")

	// Documentation endpoints
	api.HandleFunc("/documentation", h.CreateDocumentation).Methods("POST")
	api.HandleFunc("/tasks/{task_id}/documentation", h.GetDocumentation).Methods("GET")

	// WebSocket endpoint
	r.HandleFunc("/ws", hub.HandleWebSocket)

	// Serve static HTML demo (only specific file for security)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve the root path, prevent directory traversal
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "websocket-demo.html")
	})

	// Health check
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")

	return r
}

// Project handlers
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Project name is required")
		return
	}

	project := &models.Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.CreateProject(project); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create project")
		return
	}

	respondJSON(w, http.StatusCreated, project)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	project, err := h.db.GetProject(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Project not found")
		return
	}

	respondJSON(w, http.StatusOK, project)
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.db.ListProjects()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list projects")
		return
	}

	respondJSON(w, http.StatusOK, projects)
}

// Agent handlers
func (h *Handler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string            `json:"project_id"`
		Name      string            `json:"name"`
		Role      models.AgentRole  `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ProjectID == "" || req.Name == "" || req.Role == "" {
		respondError(w, http.StatusBadRequest, "Project ID, name, and role are required")
		return
	}

	if req.Role != models.RoleBackend && req.Role != models.RoleFrontend {
		respondError(w, http.StatusBadRequest, "Role must be 'backend' or 'frontend'")
		return
	}

	agent := &models.Agent{
		ID:        uuid.New().String(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Role:      req.Role,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.CreateAgent(agent); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create agent")
		return
	}

	respondJSON(w, http.StatusCreated, agent)
}

func (h *Handler) GetAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	agent, err := h.db.GetAgent(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Agent not found")
		return
	}

	respondJSON(w, http.StatusOK, agent)
}

func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["project_id"]

	agents, err := h.db.ListAgentsByProject(projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list agents")
		return
	}

	respondJSON(w, http.StatusOK, agents)
}

// Task handlers
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID   string `json:"project_id"`
		AgentID     string `json:"agent_id,omitempty"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    int    `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ProjectID == "" || req.Title == "" {
		respondError(w, http.StatusBadRequest, "Project ID and title are required")
		return
	}

	task := &models.Task{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		AgentID:     req.AgentID,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.StatusPending,
		Priority:    req.Priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.db.CreateTask(task); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	respondJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	task, err := h.db.GetTask(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Task not found")
		return
	}

	respondJSON(w, http.StatusOK, task)
}

func (h *Handler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Status  models.TaskStatus `json:"status"`
		Message string            `json:"message,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := h.db.GetTask(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Task not found")
		return
	}

	if err := h.db.UpdateTaskStatus(id, req.Status); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update task status")
		return
	}

	// Broadcast update via WebSocket
	update := &models.TaskUpdate{
		TaskID:    id,
		AgentID:   task.AgentID,
		Status:    req.Status,
		Message:   req.Message,
		Timestamp: time.Now(),
	}
	h.hub.BroadcastTaskUpdate(update)

	task.Status = req.Status
	task.UpdatedAt = time.Now()
	respondJSON(w, http.StatusOK, task)
}

func (h *Handler) ListTasksByAgent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID := vars["agent_id"]

	tasks, err := h.db.ListTasksByAgent(agentID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list tasks")
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

func (h *Handler) ListTasksByProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["project_id"]

	tasks, err := h.db.ListTasksByProject(projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list tasks")
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

// Documentation handlers
func (h *Handler) CreateDocumentation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID    string `json:"task_id"`
		Content   string `json:"content"`
		CreatedBy string `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TaskID == "" || req.Content == "" || req.CreatedBy == "" {
		respondError(w, http.StatusBadRequest, "Task ID, content, and created_by are required")
		return
	}

	doc := &models.Documentation{
		ID:        uuid.New().String(),
		TaskID:    req.TaskID,
		Content:   req.Content,
		CreatedBy: req.CreatedBy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.CreateDocumentation(doc); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create documentation")
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

func (h *Handler) GetDocumentation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["task_id"]

	docs, err := h.db.GetDocumentationByTask(taskID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get documentation")
		return
	}

	respondJSON(w, http.StatusOK, docs)
}

// Health check
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "mcp-task-tracker",
	})
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
