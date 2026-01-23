package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

// StreamingHandler handles SSE streaming for A2A tasks
type StreamingHandler struct {
	taskManager *task.Manager
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(tm *task.Manager) *StreamingHandler {
	return &StreamingHandler{taskManager: tm}
}

// StreamMessage handles POST /a2a/v1/message:stream
func (h *StreamingHandler) StreamMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		h.writeCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Message.Content == "" {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	// Set default format if not provided
	if req.Message.Format == "" {
		req.Message.Format = "text"
	}

	// Create task
	t, err := h.taskManager.CreateTask(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create task: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	h.writeCORSHeaders(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial task created event
	h.sendSSE(w, flusher, "task_created", map[string]any{
		"task_id":    t.ID,
		"status":     string(t.Status),
		"created_at": t.CreatedAt.Format(time.RFC3339),
	})

	// Subscribe to task updates
	updates := h.taskManager.SubscribeToTask(t.ID)
	defer h.taskManager.UnsubscribeFromTask(t.ID, updates)

	// Stream updates
	ctx := r.Context()
	keepaliveTicker := time.NewTicker(15 * time.Second)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return

		case update, ok := <-updates:
			if !ok {
				// Channel closed
				return
			}

			h.sendSSE(w, flusher, update.Event, update.Data)

			if update.IsFinal {
				return
			}

		case <-keepaliveTicker.C:
			// Send keepalive to prevent connection timeout
			h.sendKeepalive(w, flusher)
		}
	}
}

// sendSSE sends a Server-Sent Event
func (h *StreamingHandler) sendSSE(w http.ResponseWriter, f http.Flusher, event string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	f.Flush()
}

// sendKeepalive sends a keepalive comment
func (h *StreamingHandler) sendKeepalive(w http.ResponseWriter, f http.Flusher) {
	fmt.Fprintf(w, ": keepalive %s\n\n", time.Now().Format(time.RFC3339))
	f.Flush()
}

// writeCORSHeaders writes CORS headers for the response
func (h *StreamingHandler) writeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
}

// PollTask is a utility method for polling task status (non-streaming clients)
func (h *StreamingHandler) PollTask(ctx context.Context, taskID string, interval time.Duration, timeout time.Duration) (*models.Task, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-timeoutCh:
			return nil, fmt.Errorf("timeout waiting for task %s to complete", taskID)

		case <-ticker.C:
			t, err := h.taskManager.GetTask(ctx, taskID)
			if err != nil {
				return nil, err
			}

			if t.Status == models.TaskStatusCompleted || t.Status == models.TaskStatusFailed {
				return t, nil
			}
		}
	}
}
