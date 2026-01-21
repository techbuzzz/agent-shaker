package handlers

import (
	"log"
		log.Printf("WebSocket connection established for project %s", projectID)
	client := &ws.Client{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	log.Printf("WebSocket connection attempt with project_id: %s", projectIDStr)
	if projectIDStr == "" {
		log.Printf("WebSocket connection failed: project_id is required")
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		log.Printf("WebSocket connection failed: Invalid project_id %s: %v", projectIDStr, err)
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	log.Printf("WebSocket upgrading connection for project %s", projectID)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("WebSocket connection established, closing immediately for testing")
	conn.Close()
}
