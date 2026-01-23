package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	a2aserver "github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/handlers"
	"github.com/techbuzzz/agent-shaker/internal/mcp"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/task"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

func main() {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mcp:secret@localhost:5433/mcp_tracker?sslmode=disable"
	}

	// Connect to database
	db, err := database.NewDB(databaseURL)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		log.Println("Starting server without database for WebSocket testing...")
		db = nil
	} else {
		defer db.Close()

		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)

		log.Println("Connected to database")

		// Run migrations
		if err := runMigrations(db); err != nil {
			log.Printf("Failed to run migrations: %v", err)
			log.Println("Continuing without migrations...")
		}
	}

	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create handlers
	projectHandler := handlers.NewProjectHandler(db, hub)
	agentHandler := handlers.NewAgentHandler(db, hub)
	taskHandler := handlers.NewTaskHandler(db, hub)
	contextHandler := handlers.NewContextHandler(db, hub)
	wsHandler := handlers.NewWebSocketHandler(hub)
	dashboardHandler := handlers.NewDashboardHandler(db)
	mcpHandler := mcp.NewMCPHandler(db)

	// A2A Protocol Setup
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + getPort()
	}

	// Create A2A task store and manager
	tasksDir := os.Getenv("TASKS_DIR")
	if tasksDir == "" {
		tasksDir = "./data/tasks"
	}
	taskStore := task.NewMemoryStore(tasksDir)
	taskManager := task.NewManager(taskStore, nil, baseURL)

	// Create A2A context storage (bridges existing contexts to A2A artifacts)
	contextStorage := a2aserver.NewDatabaseContextStorage(db)

	// Create A2A handlers
	agentCardHandler := a2aserver.NewAgentCardHandler("1.0.0", baseURL)
	a2aHandler := a2aserver.NewA2AHandler(taskManager)
	streamingHandler := a2aserver.NewStreamingHandler(taskManager)
	artifactHandler := a2aserver.NewArtifactHandler(contextStorage, baseURL)

	// Setup router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Dashboard
	api.HandleFunc("/dashboard", dashboardHandler.GetDashboardStats).Methods("GET")

	// Projects
	api.HandleFunc("/projects", projectHandler.CreateProject).Methods("POST")
	api.HandleFunc("/projects", projectHandler.ListProjects).Methods("GET")
	api.HandleFunc("/projects/{id}", projectHandler.GetProject).Methods("GET")
	api.HandleFunc("/projects/{id}", projectHandler.DeleteProject).Methods("DELETE")
	api.HandleFunc("/projects/{id}/status", projectHandler.UpdateProjectStatus).Methods("PUT")

	// Agents
	api.HandleFunc("/agents", agentHandler.CreateAgent).Methods("POST")
	api.HandleFunc("/agents", agentHandler.ListAgents).Methods("GET")
	api.HandleFunc("/agents/{id}", agentHandler.GetAgent).Methods("GET")
	api.HandleFunc("/agents/{id}", agentHandler.DeleteAgent).Methods("DELETE")
	api.HandleFunc("/agents/{id}/status", agentHandler.UpdateAgentStatus).Methods("PUT")

	// Tasks
	api.HandleFunc("/tasks", taskHandler.CreateTask).Methods("POST")
	api.HandleFunc("/tasks", taskHandler.ListTasks).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{id}", taskHandler.UpdateTask).Methods("PUT")
	api.HandleFunc("/tasks/{id}", taskHandler.DeleteTask).Methods("DELETE")
	api.HandleFunc("/tasks/{id}/status", taskHandler.UpdateTaskStatus).Methods("PUT")

	// Contexts
	api.HandleFunc("/contexts", contextHandler.CreateContext).Methods("POST")
	api.HandleFunc("/contexts", contextHandler.ListContexts).Methods("GET")
	api.HandleFunc("/contexts/{id}", contextHandler.GetContext).Methods("GET")
	api.HandleFunc("/contexts/{id}", contextHandler.UpdateContext).Methods("PUT")
	api.HandleFunc("/contexts/{id}", contextHandler.DeleteContext).Methods("DELETE")

	// A2A Protocol routes
	a2aserver.RegisterA2ARoutes(r, a2aHandler, streamingHandler, artifactHandler, agentCardHandler)

	// WebSocket
	r.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// MCP Protocol endpoint (root level for VS Code)
	r.HandleFunc("/", mcpHandler.HandleMCP).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/mcp", mcpHandler.HandleMCP).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/mcp/message", mcpHandler.HandleMCP).Methods("POST", "OPTIONS")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers manually for health endpoint
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Setup CORS for API routes only
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Create a custom handler that routes WebSocket without middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// WebSocket requests bypass all middleware
		if req.URL.Path == "/ws" {
			wsHandler.HandleWebSocket(w, req)
			return
		}

		// A2A Protocol routes - handle with CORS
		if req.URL.Path == "/.well-known/agent-card.json" ||
			strings.HasPrefix(req.URL.Path, "/a2a/") {
			middleware.Recovery(
				middleware.Logger(
					c.Handler(r),
				),
			).ServeHTTP(w, req)
			return
		}

		// MCP Protocol requests (root, /mcp, /mcp/message) - handle with CORS
		if req.URL.Path == "/" || req.URL.Path == "/mcp" || len(req.URL.Path) >= 4 && req.URL.Path[:4] == "/mcp" {
			c.Handler(http.HandlerFunc(mcpHandler.HandleMCP)).ServeHTTP(w, req)
			return
		}

		// API routes get full middleware
		if len(req.URL.Path) >= 4 && req.URL.Path[:4] == "/api" {
			middleware.Recovery(
				middleware.Logger(
					middleware.RequestSizeLimit(10*1024*1024)(
						c.Handler(api),
					),
				),
			).ServeHTTP(w, req)
			return
		}

		// Health endpoint gets CORS
		if req.URL.Path == "/health" {
			middleware.Recovery(
				middleware.Logger(
					c.Handler(r),
				),
			).ServeHTTP(w, req)
			return
		}

		// Other routes get minimal middleware
		middleware.Recovery(
			middleware.Logger(
				r,
			),
		).ServeHTTP(w, req)
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Println("Agent Shaker - Multi-Protocol AI Agent Platform")
	log.Println("Endpoints:")
	log.Println("  A2A Discovery: http://localhost:" + port + "/.well-known/agent-card.json")
	log.Println("  A2A API:       http://localhost:" + port + "/a2a/v1")
	log.Println("  MCP:           http://localhost:" + port + "/ (Protocol endpoint)")
	log.Println("  REST API:      http://localhost:" + port + "/api")
	log.Println("  WebSocket:     ws://localhost:" + port + "/ws")
	log.Println("  Health:        http://localhost:" + port + "/health")
	log.Println("  GitHub:        https://github.com/techbuzzz/agent-shaker")

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runMigrations(db *database.DB) error {
	migrationSQL, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}

	_, err = db.Exec(string(migrationSQL))
	return err
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}
