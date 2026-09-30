package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
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

	vars := a2aVars(r)
	taskID := vars["taskId"]
	if taskID == "" {
		h.writeError(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	t, err := h.taskManager.GetTask(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			h.writeError(w, "Task not found", http.StatusNotFound)
			return
		}
		httpx.WriteError(w, r, err)
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

	vars := a2aVars(r)
	taskID := vars["taskId"]
	if taskID == "" {
		h.writeError(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	if err := h.taskManager.CancelTask(r.Context(), taskID); err != nil {
		switch {
		case errors.Is(err, task.ErrTaskNotFound):
			h.writeError(w, "Task not found", http.StatusNotFound)
		case errors.Is(err, task.ErrTaskTerminal):
			h.writeError(w, "Task is already in a terminal state", http.StatusConflict)
		default:
			httpx.WriteError(w, r, err)
		}
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

// RegisterA2ARoutes registers all A2A routes on the supplied stdlib mux.
//
// This is the stdlib `net/http` enhanced ServeMux version (Go 1.22+). The
// legacy signature used gorilla/mux and is no longer supported.
func RegisterA2ARoutes(mux *http.ServeMux, handler *A2AHandler, streamingHandler *StreamingHandler, artifactHandler *ArtifactHandler, agentCardHandler *AgentCardHandler) {
	// Agent card endpoint (well-known).
	mux.HandleFunc("GET /.well-known/agent-card.json", agentCardHandler.ServeHTTP)
	mux.HandleFunc("OPTIONS /.well-known/agent-card.json", agentCardHandler.ServeHTTP)

	// Task endpoints.
	mux.HandleFunc("POST /a2a/v1/message", handler.SendMessage)
	mux.HandleFunc("OPTIONS /a2a/v1/message", handler.SendMessage)
	mux.HandleFunc("GET /a2a/v1/tasks", handler.ListTasks)
	mux.HandleFunc("OPTIONS /a2a/v1/tasks", handler.ListTasks)
	mux.HandleFunc("GET /a2a/v1/tasks/{taskId}", handler.GetTask)
	mux.HandleFunc("OPTIONS /a2a/v1/tasks/{taskId}", handler.GetTask)
	mux.HandleFunc("DELETE /a2a/v1/tasks/{taskId}", handler.CancelTask)
	mux.HandleFunc("OPTIONS /a2a/v1/tasks/{taskId}", handler.CancelTask)

	// Streaming endpoint.
	if streamingHandler != nil {
		mux.HandleFunc("POST /a2a/v1/message:stream", streamingHandler.StreamMessage)
		mux.HandleFunc("OPTIONS /a2a/v1/message:stream", streamingHandler.StreamMessage)
	}

	// Artifact endpoints.
	if artifactHandler != nil {
		mux.HandleFunc("GET /a2a/v1/artifacts", artifactHandler.ListArtifacts)
		mux.HandleFunc("OPTIONS /a2a/v1/artifacts", artifactHandler.ListArtifacts)
		mux.HandleFunc("GET /a2a/v1/artifacts/{artifactId}", artifactHandler.GetArtifact)
		mux.HandleFunc("OPTIONS /a2a/v1/artifacts/{artifactId}", artifactHandler.GetArtifact)
	}
}
