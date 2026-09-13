package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	a2aserver "github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/handlers"
	"github.com/techbuzzz/agent-shaker/internal/mcp"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/observability"
	"github.com/techbuzzz/agent-shaker/internal/task"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

func main() {
	// Structured logging is the default for everything in this binary.
	slog.SetDefault(observability.NewLogger())

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mcp:secret@localhost:5433/mcp_tracker?sslmode=disable"
	}

	// Connect to database
	db, err := database.NewDB(databaseURL)
	if err != nil {
		slog.Warn("database connection failed, starting without database", "error", err)
		db = nil
	} else {
		defer db.Close()

		// Configure connection pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)

		slog.Info("database connected")

		// Run migrations
		if err := runMigrations(db); err != nil {
			slog.Warn("migrations failed, continuing without", "error", err)
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
	standupHandler := handlers.NewStandupHandler(db, hub)
	wsHandler := handlers.NewWebSocketHandler(hub)
	dashboardHandler := handlers.NewDashboardHandler(db)
	mcpHandler := mcp.NewMCPHandler(db, hub)

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
	api.HandleFunc("/tasks/{id}/reassign", taskHandler.ReassignTask).Methods("PUT")

	// Contexts
	api.HandleFunc("/contexts", contextHandler.CreateContext).Methods("POST")
	api.HandleFunc("/contexts", contextHandler.ListContexts).Methods("GET")
	api.HandleFunc("/contexts/{id}", contextHandler.GetContext).Methods("GET")
	api.HandleFunc("/contexts/{id}", contextHandler.UpdateContext).Methods("PUT")
	api.HandleFunc("/contexts/{id}", contextHandler.DeleteContext).Methods("DELETE")

	// Daily Standups
	api.HandleFunc("/standups", standupHandler.CreateStandup).Methods("POST")
	api.HandleFunc("/standups", standupHandler.ListStandups).Methods("GET")
	api.HandleFunc("/standups/{id}", standupHandler.GetStandup).Methods("GET")
	api.HandleFunc("/standups/{id}", standupHandler.UpdateStandup).Methods("PUT")
	api.HandleFunc("/standups/{id}", standupHandler.DeleteStandup).Methods("DELETE")

	// Agent Heartbeats
	api.HandleFunc("/heartbeats", standupHandler.RecordHeartbeat).Methods("POST")
	api.HandleFunc("/agents/{id}/heartbeats", standupHandler.GetAgentHeartbeats).Methods("GET")

	// A2A Protocol routes
	a2aserver.RegisterA2ARoutes(r, a2aHandler, streamingHandler, artifactHandler, agentCardHandler)

	// WebSocket
	r.HandleFunc("/ws", wsHandler.HandleWebSocket)

	// Serve static files from web/dist (if exists) - BEFORE catch-all routes.
	// http.FileServer sanitizes paths internally (rejects "..") so we don't need
	// to call os.Stat with a concatenated path; that pattern was vulnerable to
	// path traversal in earlier revisions.
	distDir := "./web/dist"
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		slog.Info("serving frontend from ./web/dist")
		fileServer := http.FileServer(http.Dir(distDir))
		indexPath := filepath.Join(distDir, "index.html")
		r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Resolve against distDir, then let FileServer handle path cleaning.
			clean := filepath.Clean(req.URL.Path)
			full := filepath.Join(distDir, clean)
			if rel, err := filepath.Rel(distDir, full); err == nil && !strings.HasPrefix(rel, "..") {
				if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
					fileServer.ServeHTTP(w, req)
					return
				}
			}
			http.ServeFile(w, req, indexPath)
		}))
	} else {
		slog.Info("frontend not found at ./web/dist - serving backend only")
	}

	// MCP Protocol endpoint (root level for VS Code) - AFTER static files
	r.HandleFunc("/", mcpHandler.HandleMCP).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/mcp", mcpHandler.HandleMCP).Methods("GET", "POST", "OPTIONS")
	r.HandleFunc("/mcp/message", mcpHandler.HandleMCP).Methods("POST", "OPTIONS")

	// Observability and health endpoints. These are intentionally outside the
	// CORS middleware so probes can hit them without an Origin header.
	obs := observability.New()
	r.Handle("/metrics", obs.Handler()).Methods("GET", "OPTIONS")
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET", "OPTIONS")
	r.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if db == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","reason":"no database"}`))
			return
		}
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(pingCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"unavailable","reason":"db ping: %s"}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}).Methods("GET", "OPTIONS")

	// Legacy /health endpoint kept for backwards compatibility; mirrors /healthz.
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}).Methods("GET", "OPTIONS")

	// Setup CORS for API routes only
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	// Create a custom handler that routes WebSocket without middleware.
	// Every request gets RequestID + Prometheus instrumentation + Recovery as
	// the outermost middleware; per-route middleware (CORS, size limit,
	// access log) is composed inside the closure below.
	chain := func(h http.Handler) http.Handler {
		return middleware.RequestID(obs.Instrument(middleware.Recovery(h)))
	}

	handler := chain(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// WebSocket requests bypass all middleware
		if req.URL.Path == "/ws" {
			wsHandler.HandleWebSocket(w, req)
			return
		}

		// A2A Protocol routes - handle with CORS
		if req.URL.Path == "/.well-known/agent-card.json" ||
			strings.HasPrefix(req.URL.Path, "/a2a/") {
			middleware.Logger(c.Handler(r)).ServeHTTP(w, req)
			return
		}

		// MCP Protocol requests (root, /mcp, /mcp/message) - handle with CORS
		if req.URL.Path == "/" || req.URL.Path == "/mcp" || len(req.URL.Path) >= 4 && req.URL.Path[:4] == "/mcp" {
			middleware.Logger(c.Handler(http.HandlerFunc(mcpHandler.HandleMCP))).ServeHTTP(w, req)
			return
		}

		// API routes get full middleware
		if len(req.URL.Path) >= 4 && req.URL.Path[:4] == "/api" {
			middleware.Logger(
				middleware.RequestSizeLimit(10*1024*1024)(
					c.Handler(api),
				),
			).ServeHTTP(w, req)
			return
		}

		// Legacy /health endpoint keeps CORS-wrapped access log
		if req.URL.Path == "/health" {
			middleware.Logger(c.Handler(r)).ServeHTTP(w, req)
			return
		}

		// Other routes get minimal middleware
		middleware.Logger(r).ServeHTTP(w, req)
	}))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("server starting", "port", port)
	slog.Info("Agent Shaker - Multi-Protocol AI Agent Platform")
	slog.Info("endpoints",
		"a2a_discovery", "http://localhost:"+port+"/.well-known/agent-card.json",
		"a2a_api", "http://localhost:"+port+"/a2a/v1",
		"mcp", "http://localhost:"+port+"/",
		"rest_api", "http://localhost:"+port+"/api",
		"websocket", "ws://localhost:"+port+"/ws",
		"health", "http://localhost:"+port+"/healthz",
		"ready", "http://localhost:"+port+"/readyz",
		"metrics", "http://localhost:"+port+"/metrics",
		"github", "https://github.com/techbuzzz/agent-shaker",
	)

	if err := runServer(":"+port, handler, hub); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// runServer starts an http.Server and blocks until a SIGINT/SIGTERM arrives,
// then drains in-flight requests (graceful shutdown) and finally closes the
// WebSocket hub so all client pumps exit cleanly.
func runServer(addr string, h http.Handler, hub *websocket.Hub) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Cancel root context on SIGINT/SIGTERM so background goroutines can drain.
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			return err
		}
		return nil
	case <-rootCtx.Done():
		slog.Info("shutdown signal received, draining HTTP connections")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
	}

	slog.Info("closing WebSocket hub")
	hub.Shutdown()
	slog.Info("server stopped")
	return nil
}

func runMigrations(db *database.DB) error {
	slog.Info("running database migrations")

	// Acquire advisory lock to prevent concurrent migrations
	// Use a fixed integer key for migrations lock (hash of "agent-shaker-migrations")
	const migrationLockKey = 918273645

	// Get a dedicated connection to ensure advisory lock is acquired and released
	// on the same session (PostgreSQL advisory locks are session-scoped)
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get dedicated connection: %w", err)
	}
	defer conn.Close()

	// Try to acquire advisory lock (non-blocking)
	var lockAcquired bool
	err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", migrationLockKey).Scan(&lockAcquired)
	if err != nil {
		return fmt.Errorf("failed to acquire migration lock: %w", err)
	}

	if !lockAcquired {
		slog.Info("another instance is running migrations, waiting")
		// Block until we can acquire the lock
		_, err = conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey)
		if err != nil {
			return fmt.Errorf("failed to wait for migration lock: %w", err)
		}
	}

	// Ensure we release the lock when done
	defer func() {
		_, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrationLockKey)
		if err != nil {
			slog.Warn("failed to release migration lock", "error", err)
		}
	}()

	slog.Info("migration lock acquired")

	// Create migrations tracking table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64)
		);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return err
	}

	// Read all migration files from migrations directory
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}

	// Sort entries to ensure deterministic execution order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// Get already applied migrations
	appliedMigrations := make(map[string]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return err
	}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return err
		}
		appliedMigrations[version] = true
	}
	rows.Close()

	// Apply pending migrations in order
	appliedCount := 0
	migrationPattern := regexp.MustCompile(`^\d`)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Skip bootstrap and helper files (only process files starting with a digit)
		if !migrationPattern.MatchString(entry.Name()) {
			slog.Info("skipping non-migration file", "name", entry.Name())
			continue
		}

		// Skip if already applied
		if appliedMigrations[entry.Name()] {
			continue
		}

		slog.Info("applying migration", "name", entry.Name())

		// Read migration file
		migrationSQL, err := os.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		// Begin transaction for this migration
		// This ensures atomicity: either the migration and its record both succeed, or both fail
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", entry.Name(), err)
		}

		// Execute migration DDL within the transaction
		if _, err := tx.Exec(string(migrationSQL)); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Warn("failed to rollback transaction after migration error", "error", rbErr)
			}
			slog.Error("failed to apply migration", "name", entry.Name(), "error", err)
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}

		// Record the migration as applied within same transaction
		// ON CONFLICT provides defense-in-depth: if somehow a migration was recorded
		// between our initial check and now, we detect it here and skip redundant work
		_, err = tx.Exec(
			`INSERT INTO schema_migrations (version, applied_at)
			 VALUES ($1, CURRENT_TIMESTAMP)
			 ON CONFLICT (version) DO NOTHING`,
			entry.Name(),
		)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Warn("failed to rollback transaction after insert error", "error", rbErr)
			}
			return fmt.Errorf("failed to record migration %s: %w", entry.Name(), err)
		}

		// Commit the transaction - migration DDL and tracking record are both applied atomically
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", entry.Name(), err)
		}

		appliedCount++
		slog.Info("applied migration", "name", entry.Name())
	}

	if appliedCount == 0 {
		slog.Info("no pending migrations")
	} else {
		slog.Info("migrations applied", "count", appliedCount)
	}

	return nil
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return port
}
