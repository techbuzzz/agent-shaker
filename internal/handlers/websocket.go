package handlers

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	ws "github.com/techbuzzz/agent-shaker/internal/websocket"
)

// WebSocketHandler delegates the WebSocket upgrade to the shared Hub so the
// origin allow-list, project routing, and goroutine lifecycle are owned in one
// place. This handler is kept only to keep the existing route registration
// signature in main.go.
type WebSocketHandler struct {
	hub *ws.Hub
}

func NewWebSocketHandler(hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Pre-validate project_id so we can fail-fast with a 400 before the upgrade.
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		slog.WarnContext(r.Context(), "websocket connection rejected: project_id is required")
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	if _, err := uuid.Parse(projectIDStr); err != nil {
		slog.WarnContext(r.Context(), "websocket connection rejected: invalid project_id", "project_id", projectIDStr, "error", err)
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	slog.DebugContext(r.Context(), "websocket delegating connection", "project_id", projectIDStr)
	h.hub.HandleWebSocket(w, r)
}
