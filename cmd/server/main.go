package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"

	a2aserver "github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/handlers"
	"github.com/techbuzzz/agent-shaker/internal/mcp"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/observability"
	"github.com/techbuzzz/agent-shaker/internal/task"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

func main() {
	// Root context for startup. The server lifecycle owns its own context.
	ctx := context.Background()

	// Tracing must be initialised BEFORE the logger so the otelslog bridge
	// is available when NewLogger is constructed (otherwise log lines would
	// lack trace_id correlation). InitTracing is a no-op when
	// OTEL_EXPORTER_OTLP_ENDPOINT is unset.
	tracingShutdown, err := observability.InitTracing(ctx, "agent-shaker", "0.1.0")
	if err != nil {
		slog.Error("tracing init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracingShutdown(shutdownCtx); err != nil {
			slog.Error("tracing shutdown error", "error", err)
		}
	}()

	// Structured logging is the default for everything in this binary.
	slog.SetDefault(observability.NewLogger())

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mcp:secret@localhost:5433/mcp_tracker?sslmode=disable"
	}

	// Production safety guard: refuse to boot when sslmode=disable would
	// silently send credentials and data over an unencrypted connection.
	if os.Getenv("ENV") == "production" && strings.Contains(databaseURL, "sslmode=disable") {
		slog.Error("refusing to start in production with sslmode=disable")
		os.Exit(2)
	}

	// Connect to database. Pool tuning happens inside NewDB; see
	// internal/database/database.go for defaults.
	db, err := database.NewDB(ctx, databaseURL)
	if err != nil {
		slog.Warn("database connection failed, starting without database", "error", err)
		db = nil
	} else {
		defer db.Close()

		slog.Info("database connected")

		// Run migrations on startup unless the operator has disabled it (CI
		// runs the dedicated cmd/migrate binary instead).
		if os.Getenv("RUN_MIGRATIONS_ON_START") != "false" {
			if err := runMigrations(db); err != nil {
				slog.Warn("migrations failed, continuing without", "error", err)
			}
		} else {
			slog.Info("RUN_MIGRATIONS_ON_START=false; skipping in-process migrations")
		}
	}

	// Create WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Create handlers. Handlers own a typed query store. When the database
	// is unavailable, pass a literal nil interface — a typed-nil *database.DB
	// would box into a non-nil Querier interface, defeating the handlers'
	// `store.Available()` short-circuit (Go's typed-nil-interface gotcha).
	var querier queries.Querier
	if db != nil {
		querier = db
	}
	projectStore := queries.NewProjectsStore(querier)
	agentStore := queries.NewAgentsStore(querier)
	tasksQueryStore := queries.NewTasksStore(querier)
	contextsQueryStore := queries.NewContextsStore(querier)
	standupsQueryStore := queries.NewStandupsStore(querier)
	dashboardQueryStore := queries.NewDashboardStore(querier)
	milestonesStore := queries.NewMilestonesStore(querier)
	projectReposStore := queries.NewProjectReposStore(querier)
	globalContextsStore := queries.NewGlobalContextsStore(querier)
	projectHandler := handlers.NewProjectHandler(projectStore, hub)
	agentHandler := handlers.NewAgentHandler(agentStore, hub)
	taskHandler := handlers.NewTaskHandler(tasksQueryStore, hub)
	contextHandler := handlers.NewContextHandler(contextsQueryStore, hub)
	standupHandler := handlers.NewStandupHandler(standupsQueryStore, hub)
	wsHandler := handlers.NewWebSocketHandler(hub)
	dashboardHandler := handlers.NewDashboardHandler(dashboardQueryStore)
	milestoneHandler := handlers.NewMilestoneHandler(milestonesStore, tasksQueryStore, agentStore, hub, db)
	projectRepoHandler := handlers.NewProjectRepoHandler(projectReposStore, hub)
	globalContextHandler := handlers.NewGlobalContextHandler(globalContextsStore, hub)
	mcpHandler := mcp.NewMCPHandler(db, hub)

	// A2A Protocol Setup
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:" + getPort()
	}

	// Create A2A task store. Default to MemoryStore for back-compat; opt into
	// the Postgres-backed store via TASK_STORE=postgres (requires DATABASE_URL
	// and that migrations 001..004 have been applied).
	var taskStore task.Store
	switch os.Getenv("TASK_STORE") {
	case "", "memory":
		tasksDir := os.Getenv("TASKS_DIR")
		if tasksDir == "" {
			tasksDir = "./data/tasks"
		}
		taskStore = task.NewMemoryStore(tasksDir)
		slog.Info("task store", "type", "memory", "dir", tasksDir)
	case "postgres":
		if db == nil {
			slog.Error("TASK_STORE=postgres requires a working database connection")
			os.Exit(1)
		}
		pgxPool, err := database.NewPool(ctx, os.Getenv("DATABASE_URL"))
		if err != nil {
			slog.Error("failed to open pgxpool for task store", "error", err)
			os.Exit(1)
		}
		taskStore = task.NewPostgresStore(pgxPool)
		slog.Info("task store", "type", "postgres")
	default:
		slog.Error("unknown TASK_STORE value; expected memory or postgres", "value", os.Getenv("TASK_STORE"))
		os.Exit(1)
	}
	taskManager := task.NewManager(taskStore, nil, baseURL)

	// Create A2A context storage (bridges existing contexts to A2A artifacts)
	contextStorage := a2aserver.NewDatabaseContextStorage(db)

	// Create A2A handlers
	agentCardHandler := a2aserver.NewAgentCardHandler("1.0.0", baseURL)
	a2aHandler := a2aserver.NewA2AHandler(taskManager)
	streamingHandler := a2aserver.NewStreamingHandler(taskManager)
	artifactHandler := a2aserver.NewArtifactHandler(contextStorage, baseURL)

	// Build the route table and middleware chain. See cmd/server/routes.go
	// for the actual route registrations.
	obs := observability.New()
	rateMW, rateShutdown := middleware.RateLimit(middleware.RateLimitConfig{
		Limit: middleware.RateFromEnv("RATE_LIMIT_RPS", 100),
		Burst: middleware.IntFromEnv("RATE_LIMIT_BURST", 200),
		Skip:  middleware.SkipPaths("/ws", "/healthz", "/readyz", "/metrics"),
	})
	defer rateShutdown(context.Background())

	corsOrigins := middleware.CORSOriginsFromEnv("CORS_ALLOWED_ORIGINS", "http://localhost", "http://127.0.0.1")
	corsAllowCreds := os.Getenv("AUTH_ENABLED") == "true"

	deps := routeDeps{
		db:                   db,
		hub:                  hub,
		projectHandler:       projectHandler,
		agentHandler:         agentHandler,
		taskHandler:          taskHandler,
		contextHandler:       contextHandler,
		standupHandler:       standupHandler,
		wsHandler:            wsHandler,
		dashboardHandler:     dashboardHandler,
		milestoneHandler:     milestoneHandler,
		projectRepoHandler:   projectRepoHandler,
		globalContextHandler: globalContextHandler,
		mcpHandler:           mcpHandler,
		agentCardHandler:     agentCardHandler,
		a2aHandler:           a2aHandler,
		streamingHandler:     streamingHandler,
		artifactHandler:      artifactHandler,
		obs:                  obs,
		maxBodyBytes:         parseMaxBodyBytes(),
		corsOrigins:          corsOrigins,
		corsAllowCreds:       corsAllowCreds,
		isTLS:                false,
		rateLimitShutdown:    rateShutdown,
	}

	rootHandler, err := newServeMux(deps)
	if err != nil {
		slog.Error("failed to build routes", "error", err)
		os.Exit(1)
	}
	// Wrap the router with the rate limiter so it sits between RequestID
	// (outermost) and the per-route CORS handlers.
	handler := middleware.Apply(rootHandler, rateMW)

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
