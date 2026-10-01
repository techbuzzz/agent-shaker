package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/handlers"
	"github.com/techbuzzz/agent-shaker/internal/mcp"
	"github.com/techbuzzz/agent-shaker/internal/middleware"
	"github.com/techbuzzz/agent-shaker/internal/observability"
)

// routeDeps bundles everything the route table needs to wire the HTTP graph.
type routeDeps struct {
	db                   *database.DB
	hub                  hubLike
	projectHandler       *handlers.ProjectHandler
	agentHandler         *handlers.AgentHandler
	taskHandler          *handlers.TaskHandler
	contextHandler       *handlers.ContextHandler
	standupHandler       *handlers.StandupHandler
	wsHandler            *handlers.WebSocketHandler
	dashboardHandler     *handlers.DashboardHandler
	milestoneHandler     *handlers.MilestoneHandler
	projectRepoHandler   *handlers.ProjectRepoHandler
	globalContextHandler *handlers.GlobalContextHandler
	mcpHandler           *mcp.MCPHandler
	agentCardHandler     *server.AgentCardHandler
	a2aHandler           *server.A2AHandler
	streamingHandler     *server.StreamingHandler
	artifactHandler      *server.ArtifactHandler
	obs                  *observability.Metrics
	maxBodyBytes         int64
	corsOrigins          []string
	corsAllowCreds       bool
	isTLS                bool
	rateLimitShutdown    func(context.Context)
}

// hubLike is the minimal interface we need from the WebSocket hub. Defined
// here so tests don't have to construct a full *websocket.Hub.
type hubLike interface {
	Shutdown()
}

// newServeMux builds the stdlib http.ServeMux and composes middlewares per
// route. Outer middlewares (RequestID, Prometheus instrument, Recovery) wrap
// the entire mux; per-route middlewares (CORS, size limit, logger) are
// composed per handler.
func newServeMux(d routeDeps) (http.Handler, error) {
	mux := http.NewServeMux()

	// Observability + readiness (no CORS, no body limit).
	mux.Handle("GET /metrics", d.obs.Handler())
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if d.db == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","reason":"no database"}`))
			return
		}
		pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := d.db.PingContext(pingCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, `{"status":"unavailable","reason":"db ping: %s"}`, err.Error())
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	// Legacy /health kept for backwards compatibility.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// WebSocket: no middleware besides the global chain; the handler manages
	// its own origin check.
	mux.HandleFunc("GET /ws", d.wsHandler.HandleWebSocket)

	// MCP — registered under three paths so the same handler serves all of
	// them. CORS-wrapped because external clients (VS Code) hit these.
	mcpHandler := middleware.Apply(
		http.HandlerFunc(d.mcpHandler.HandleMCP),
		middleware.Logger,
		middleware.CORS(d.corsOrigins, d.corsAllowCreds, nil, nil),
	)
	mux.Handle("GET /{$}", mcpHandler)
	mux.Handle("POST /{$}", mcpHandler)
	mux.Handle("OPTIONS /{$}", mcpHandler)
	mux.Handle("GET /mcp", mcpHandler)
	mux.Handle("POST /mcp", mcpHandler)
	mux.Handle("OPTIONS /mcp", mcpHandler)
	mux.Handle("POST /mcp/message", mcpHandler)
	mux.Handle("OPTIONS /mcp/message", mcpHandler)

	// A2A — agent card and task endpoints; CORS-wrapped.
	// Sub-mux so each path uses its own {taskId} pattern and r.PathValue works
	// in the handlers.
	a2aSub := http.NewServeMux()
	a2aSub.HandleFunc("POST /message", d.a2aHandler.SendMessage)
	a2aSub.HandleFunc("OPTIONS /message", d.a2aHandler.SendMessage)
	a2aSub.HandleFunc("GET /tasks", d.a2aHandler.ListTasks)
	a2aSub.HandleFunc("OPTIONS /tasks", d.a2aHandler.ListTasks)
	a2aSub.HandleFunc("GET /tasks/{taskId}", d.a2aHandler.GetTask)
	a2aSub.HandleFunc("DELETE /tasks/{taskId}", d.a2aHandler.CancelTask)
	// OPTIONS is handled by the CORS middleware above; no explicit handler
	// here (otherwise Go 1.22+ ServeMux panics on duplicate patterns).
	if d.streamingHandler != nil {
		a2aSub.HandleFunc("POST /message:stream", d.streamingHandler.StreamMessage)
		a2aSub.HandleFunc("OPTIONS /message:stream", d.streamingHandler.StreamMessage)
	}
	if d.artifactHandler != nil {
		a2aSub.HandleFunc("GET /artifacts", d.artifactHandler.ListArtifacts)
		a2aSub.HandleFunc("OPTIONS /artifacts", d.artifactHandler.ListArtifacts)
		a2aSub.HandleFunc("GET /artifacts/{artifactId}", d.artifactHandler.GetArtifact)
		a2aSub.HandleFunc("OPTIONS /artifacts/{artifactId}", d.artifactHandler.GetArtifact)
	}
	a2aHandler := middleware.Apply(a2aSub,
		middleware.Logger,
		middleware.CORS(d.corsOrigins, d.corsAllowCreds, nil, nil),
	)
	mux.Handle("/a2a/v1/", a2aHandler)

	// Agent card lives at /.well-known/agent-card.json (outside /a2a/v1/).
	mux.Handle("GET /.well-known/agent-card.json", middleware.Apply(
		http.HandlerFunc(d.agentCardHandler.ServeHTTP),
		middleware.Logger,
		middleware.CORS(d.corsOrigins, d.corsAllowCreds, nil, nil),
	))
	mux.Handle("OPTIONS /.well-known/agent-card.json", middleware.Apply(
		http.HandlerFunc(d.agentCardHandler.ServeHTTP),
		middleware.Logger,
		middleware.CORS(d.corsOrigins, d.corsAllowCreds, nil, nil),
	))

	// REST API — CORS-wrapped + body-size limited.
	apiHandler := middleware.Apply(d.apiRouter(),
		middleware.Logger,
		middleware.CORS(d.corsOrigins, d.corsAllowCreds, nil, nil),
		middleware.RequestSizeLimit(d.maxBodyBytes),
	)
	mux.Handle("/api/", apiHandler)

	// SPA fallback. http.FileServer rejects paths that escape its root, so
	// the previous distDir + req.URL.Path concatenation (a path-traversal
	// sink) is gone. For any request that doesn't match an explicit route
	// above, we serve the static file or fall back to index.html for SPA
	// client-side routing.
	//
	// Note: in the containerised topology the frontend is a separate Nuxt SSR
	// service, so ./web/dist is usually absent and this branch is not taken.
	// The "not found" line is logged once at wiring time, not per request —
	// an unmatched-path handler fires on every 404 and would otherwise flood
	// the log.
	if info, err := os.Stat("./web/dist"); err == nil && info.IsDir() {
		slog.Info("serving frontend from ./web/dist")
		mux.Handle("/", d.spaHandler("./web/dist"))
	} else {
		slog.Info("no ./web/dist directory; this server is API-only (frontend is a separate service)")
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	// Outer chain applied last so it wraps every registered route. Order
	// matters: RequestID first (innermost attribute), then the OTel server
	// span (which reads the inbound traceparent), then Prometheus, then
	// recovery, then security headers.
	return middleware.Apply(mux,
		middleware.RequestID,
		otelhttp.NewMiddleware("agent-shaker.http"),
		d.obs.Instrument,
		middleware.Recovery,
		middleware.SecurityHeaders(d.isTLS, ""),
	), nil
}

// spaHandler returns a handler that serves files from distDir and falls back
// to distDir/index.html for client-side routes. Path-traversal protection is
// delegated to http.FileServer (rejects ".." segments internally).
func (d routeDeps) spaHandler(distDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(distDir))
	indexPath := filepath.Join(distDir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(r.URL.Path)
		full := filepath.Join(distDir, clean)
		if rel, err := filepath.Rel(distDir, full); err == nil && !strings.HasPrefix(rel, "..") {
			if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, indexPath)
	})
}

// apiRouter is a small sub-mux for /api routes so we can attach the API
// handlers with method-prefixed patterns.
func (d routeDeps) apiRouter() http.Handler {
	mux := http.NewServeMux()
	// Dashboard
	mux.HandleFunc("GET /api/dashboard", d.dashboardHandler.GetDashboardStats)
	// Projects
	mux.HandleFunc("GET /api/projects", d.projectHandler.ListProjects)
	mux.HandleFunc("POST /api/projects", d.projectHandler.CreateProject)
	mux.HandleFunc("GET /api/projects/{id}", d.projectHandler.GetProject)
	mux.HandleFunc("PUT /api/projects/{id}/status", d.projectHandler.UpdateProjectStatus)
	mux.HandleFunc("DELETE /api/projects/{id}", d.projectHandler.DeleteProject)
	// Agents
	mux.HandleFunc("GET /api/agents", d.agentHandler.ListAgents)
	mux.HandleFunc("POST /api/agents", d.agentHandler.CreateAgent)
	mux.HandleFunc("GET /api/agents/{id}", d.agentHandler.GetAgent)
	mux.HandleFunc("PUT /api/agents/{id}/status", d.agentHandler.UpdateAgentStatus)
	mux.HandleFunc("DELETE /api/agents/{id}", d.agentHandler.DeleteAgent)
	// Tasks
	mux.HandleFunc("GET /api/tasks", d.taskHandler.ListTasks)
	mux.HandleFunc("POST /api/tasks", d.taskHandler.CreateTask)
	mux.HandleFunc("GET /api/tasks/{id}", d.taskHandler.GetTask)
	mux.HandleFunc("PUT /api/tasks/{id}", d.taskHandler.UpdateTask)
	mux.HandleFunc("PUT /api/tasks/{id}/status", d.taskHandler.UpdateTaskStatus)
	mux.HandleFunc("PUT /api/tasks/{id}/reassign", d.taskHandler.ReassignTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", d.taskHandler.DeleteTask)
	// Contexts
	mux.HandleFunc("GET /api/contexts", d.contextHandler.ListContexts)
	mux.HandleFunc("POST /api/contexts", d.contextHandler.CreateContext)
	mux.HandleFunc("GET /api/contexts/{id}", d.contextHandler.GetContext)
	mux.HandleFunc("PUT /api/contexts/{id}", d.contextHandler.UpdateContext)
	mux.HandleFunc("DELETE /api/contexts/{id}", d.contextHandler.DeleteContext)
	// Standups
	mux.HandleFunc("GET /api/standups", d.standupHandler.ListStandups)
	mux.HandleFunc("POST /api/standups", d.standupHandler.CreateStandup)
	mux.HandleFunc("GET /api/standups/{id}", d.standupHandler.GetStandup)
	mux.HandleFunc("PUT /api/standups/{id}", d.standupHandler.UpdateStandup)
	mux.HandleFunc("DELETE /api/standups/{id}", d.standupHandler.DeleteStandup)
	// Heartbeats
	mux.HandleFunc("POST /api/heartbeats", d.standupHandler.RecordHeartbeat)
	mux.HandleFunc("GET /api/agents/{id}/heartbeats", d.standupHandler.GetAgentHeartbeats)
	// Milestones (Phase 1)
	if d.milestoneHandler != nil {
		mux.HandleFunc("GET /api/milestones", d.milestoneHandler.ListMilestones)
		mux.HandleFunc("POST /api/milestones", d.milestoneHandler.CreateMilestone)
		mux.HandleFunc("GET /api/milestones/{id}", d.milestoneHandler.GetMilestone)
		mux.HandleFunc("PUT /api/milestones/{id}/status", d.milestoneHandler.UpdateMilestoneStatus)
		mux.HandleFunc("DELETE /api/milestones/{id}", d.milestoneHandler.DeleteMilestone)
	}
	// Project repos (Phase 2)
	if d.projectRepoHandler != nil {
		mux.HandleFunc("GET /api/project_repos", d.projectRepoHandler.ListRepos)
		mux.HandleFunc("POST /api/project_repos", d.projectRepoHandler.CreateRepo)
		mux.HandleFunc("GET /api/project_repos/{id}", d.projectRepoHandler.GetRepo)
		mux.HandleFunc("PUT /api/project_repos/{id}", d.projectRepoHandler.UpdateRepo)
		mux.HandleFunc("DELETE /api/project_repos/{id}", d.projectRepoHandler.DeleteRepo)
	}
	// Global contexts (Phase 3)
	if d.globalContextHandler != nil {
		mux.HandleFunc("GET /api/global_contexts", d.globalContextHandler.List)
		mux.HandleFunc("POST /api/global_contexts", d.globalContextHandler.Create)
		mux.HandleFunc("GET /api/global_contexts/{id}", d.globalContextHandler.Get)
		mux.HandleFunc("PUT /api/global_contexts/{id}", d.globalContextHandler.Update)
		mux.HandleFunc("DELETE /api/global_contexts/{id}", d.globalContextHandler.Delete)
	}
	return mux
}

// parseMaxBodyBytes reads the MAX_BODY_BYTES env var (default 10 MiB) into
// an int64 suitable for middleware.RequestSizeLimit.
func parseMaxBodyBytes() int64 {
	v := os.Getenv("MAX_BODY_BYTES")
	if v == "" {
		return 10 << 20
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		slog.Warn("invalid MAX_BODY_BYTES, using default", "value", v, "default", 10<<20)
		return 10 << 20
	}
	return n
}
