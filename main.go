package main

import (
	"log"
	"net/http"
	"os"

	"github.com/techbuzzz/agent-shaker/internal/api"
	"github.com/techbuzzz/agent-shaker/internal/db"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

func main() {
	// Initialize database
	database, err := db.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer database.Close()

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Setup API routes
	router := api.NewRouter(database, hub)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("MCP Task Tracker server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
