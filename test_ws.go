package main

import (
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	log.Printf("WebSocket connection attempt with project_id: %s", projectIDStr)

	if projectIDStr == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		log.Printf("Invalid project_id: %v", err)
		http.Error(w, "Invalid project_id", http.StatusBadRequest)
		return
	}

	log.Printf("Upgrading connection for project %s", projectID)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	log.Printf("WebSocket connection established")
	conn.Close()
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	log.Println("Test WebSocket server starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
