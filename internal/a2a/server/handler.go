package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

// A2AHandler handles A2A protocol HTTP requests
type A2AHandler struct {
	taskManager *task.Manager
}

// NewA2AHandler creates a new A2A handler
func NewA2AHandler(tm *task.Manager) *A2AHandler {
	return &A2AHandler{taskManager: tm}
}

// SendMessage handles POST /a2a/v1/message
func (h *A2AHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Message.Content == "" {
		h.writeError(w, "Message content is required", http.StatusBadRequest)
		return
	}

	// Set default format if not provided
	if req.Message.Format == "" {
		req.Message.Format = "text"
	}

	// Create the task
	t, err := h.taskManager.CreateTask(r.Context(), &req)
	if err != nil {
		h.writeError(w, "Failed to create task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.SendMessageResponse{
		TaskID:    t.ID,
		Status:    string(t.Status),
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}

	h.writeJSON(w, resp, http.StatusAccepted)
}

// GetTask handles GET /a2a/v1/tasks/{taskId}
func (h *A2AHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	taskID := vars["taskId"]
	if taskID == "" {
		h.writeError(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	t, err := h.taskManager.GetTask(r.Context(), taskID)
	if err != nil {
		h.writeError(w, "Task not found", http.StatusNotFound)
		return
	}

	h.writeJSON(w, t, http.StatusOK)
}

// ListTasks handles GET /a2a/v1/tasks
func (h *A2AHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	filter := &task.Filter{
		Status: r.URL.Query().Get("status"),
		Limit:  100, // Default limit
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	tasks, err := h.taskManager.ListTasks(r.Context(), filter)
	if err != nil {
		h.writeError(w, "Failed to list tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure we return an empty array instead of null
	if tasks == nil {
		tasks = []models.Task{}
	}

	resp := models.TaskListResponse{
		Tasks:      tasks,
		TotalCount: len(tasks),
	}

	h.writeJSON(w, resp, http.StatusOK)
}

// CancelTask handles DELETE /a2a/v1/tasks/{taskId}
func (h *A2AHandler) CancelTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	taskID := vars["taskId"]
	if taskID == "" {
		h.writeError(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	if err := h.taskManager.CancelTask(r.Context(), taskID); err != nil {
		h.writeError(w, "Failed to cancel task: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response with the given status code
func (h *A2AHandler) writeJSON(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response
func (h *A2AHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)

	errorResp := map[string]string{"error": message}
	json.NewEncoder(w).Encode(errorResp)
}

// RegisterA2ARoutes registers all A2A routes on the router
func RegisterA2ARoutes(r *mux.Router, handler *A2AHandler, streamingHandler *StreamingHandler, artifactHandler *ArtifactHandler, agentCardHandler *AgentCardHandler) {
	// Agent card endpoint (well-known)
	r.HandleFunc("/.well-known/agent-card.json", agentCardHandler.ServeHTTP).Methods("GET", "OPTIONS")

	// A2A API v1 routes
	a2a := r.PathPrefix("/a2a/v1").Subrouter()

	// Task endpoints
	a2a.HandleFunc("/message", handler.SendMessage).Methods("POST", "OPTIONS")
	a2a.HandleFunc("/tasks", handler.ListTasks).Methods("GET", "OPTIONS")
	a2a.HandleFunc("/tasks/{taskId}", handler.GetTask).Methods("GET", "OPTIONS")
	a2a.HandleFunc("/tasks/{taskId}", handler.CancelTask).Methods("DELETE", "OPTIONS")

	// Streaming endpoint
	if streamingHandler != nil {
		a2a.HandleFunc("/message:stream", streamingHandler.StreamMessage).Methods("POST", "OPTIONS")
	}

	// Artifact endpoints
	if artifactHandler != nil {
		a2a.HandleFunc("/artifacts", artifactHandler.ListArtifacts).Methods("GET", "OPTIONS")
		a2a.HandleFunc("/artifacts/{artifactId}", artifactHandler.GetArtifact).Methods("GET", "OPTIONS")
	}
}
