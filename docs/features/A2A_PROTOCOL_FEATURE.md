# Technical Specification: A2A Protocol Integration for Agent Shaker

**Project:** Agent Shaker - A2A Protocol Support  
**Backend:** Go  
**Version:** 1.0  
**Date:** 2026-01-23  
**Status:** Draft

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Project Overview](#project-overview)
3. [Technical Architecture](#technical-architecture)
4. [Epic: Add A2A Protocol Support to Agent Shaker](#epic-add-a2a-protocol-support-to-agent-shaker)
5. [Story 1: Agent Discovery - Agent Card Endpoint](#story-1-agent-discovery---agent-card-endpoint)
6. [Story 2: A2A Task Lifecycle Implementation](#story-2-a2a-task-lifecycle-implementation)
7. [Story 3: Streaming Support for Real-time Updates](#story-3-streaming-support-for-real-time-updates)
8. [Story 4: Context Sharing as A2A Artifacts](#story-4-context-sharing-as-a2a-artifacts)
9. [Story 5: A2A Client for External Agent Communication](#story-5-a2a-client-for-external-agent-communication)
10. [Story 6: Integration with Existing MCP Functionality](#story-6-integration-with-existing-mcp-functionality)
11. [Story 7: Documentation and Usage Examples](#story-7-documentation-and-usage-examples)
12. [Story 8: Testing and A2A Compatibility Validation](#story-8-testing-and-a2a-compatibility-validation)
13. [Implementation Timeline](#implementation-timeline)
14. [Definition of Done](#definition-of-done)

---

## Executive Summary

This technical specification outlines the implementation of Agent-to-Agent (A2A) Protocol support in the Agent Shaker platform. Agent Shaker is currently an MCP (Model Context Protocol) server written in Go, which manages AI agent contexts and provides real-time WebSocket communication capabilities.

The goal is to extend Agent Shaker to be fully A2A-compatible, enabling:
- **Agent Discovery** via standardized agent-card.json
- **Task Management** through A2A task lifecycle endpoints
- **Real-time Streaming** for task updates via Server-Sent Events (SSE)
- **Context/Artifact Sharing** between agents
- **Bidirectional Communication** with external A2A-compatible agents

---

## Project Overview

### Current State
- **Platform:** Agent Shaker MCP Server
- **Language:** Go (Golang)
- **Features:**
  - MCP-compliant server with tools and resources
  - Context-aware endpoints for managing agent contexts (markdown docs)
  - WebSocket hub for real-time notifications
  - REST API for context CRUD operations
  - CLI application for MCP server management

### Target State
- Full A2A Protocol compliance (server-side)
- Ability to act as an A2A agent that can be discovered and communicated with
- A2A client capability to delegate tasks to external A2A agents
- Seamless integration between MCP and A2A protocols

### Technology Stack
- **Language:** Go 1.21+
- **Web Framework:** Chi/Gorilla Mux (or existing HTTP router)
- **Real-time Communication:** Existing WebSocket Hub + SSE for A2A
- **Storage:** File-based storage for contexts/artifacts
- **Serialization:** JSON
- **Protocols:** MCP, A2A, HTTP/REST, WebSocket, SSE

---

## Technical Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Agent Shaker Platform                    │
│                                                             │
│  ┌──────────────┐       ┌──────────────┐                  │
│  │ MCP Server   │       │ A2A Server   │                  │
│  │ (Existing)   │       │ (New)        │                  │
│  └──────────────┘       └──────────────┘                  │
│         │                       │                          │
│         └───────┬───────────────┘                          │
│                 │                                          │
│         ┌───────▼────────┐                                │
│         │  Task Manager  │                                │
│         │  (Unified)     │                                │
│         └───────┬────────┘                                │
│                 │                                          │
│    ┌────────────┼────────────┐                            │
│    │            │            │                            │
│ ┌──▼───┐   ┌───▼────┐  ┌───▼──────┐                      │
│ │Context│   │WebSocket│ │A2A Client│                      │
│ │Storage│   │  Hub    │ │(Outbound)│                      │
│ └───────┘   └────────┘  └──────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

### Go Package Structure

```
agent-shaker/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── a2a/
│   │   ├── server/
│   │   │   ├── handler.go          # A2A HTTP handlers
│   │   │   ├── streaming.go        # SSE streaming logic
│   │   │   ├── agent_card.go       # Agent card generation
│   │   │   └── middleware.go       # A2A-specific middleware
│   │   ├── client/
│   │   │   ├── client.go           # A2A client implementation
│   │   │   ├── discovery.go        # Agent discovery logic
│   │   │   └── task.go             # Task submission/polling
│   │   ├── models/
│   │   │   ├── task.go             # A2A task models
│   │   │   ├── artifact.go         # A2A artifact models
│   │   │   └── agent_card.go       # Agent card models
│   │   └── mapper/
│   │       └── mcp_a2a.go          # MCP <-> A2A mappings
│   ├── mcp/
│   │   ├── server/                 # Existing MCP server
│   │   └── tools/                  # MCP tools
│   ├── task/
│   │   ├── manager.go              # Unified task manager
│   │   └── store.go                # Task persistence
│   ├── context/
│   │   └── storage.go              # Context/artifact storage
│   └── websocket/
│       └── hub.go                  # Existing WebSocket hub
├── api/
│   └── openapi.yaml                # A2A OpenAPI spec
├── docs/
│   └── A2A_INTEGRATION.md
└── tests/
    └── a2a/
        ├── integration_test.go
        └── compatibility_test.go
```

---

## Epic: Add A2A Protocol Support to Agent Shaker

### Epic Description
Implement full server-side A2A Protocol support in Agent Shaker to enable agent discovery, task management, streaming updates, and integration with external A2A agents. This will transform Agent Shaker from an MCP-only server into a dual-protocol platform capable of MCP and A2A communication.

### Business Value
- Interoperability with the growing A2A ecosystem
- Ability to delegate complex tasks to specialized external agents
- Enhanced real-time communication capabilities
- Position Agent Shaker as a versatile multi-protocol AI agent platform

### Acceptance Criteria
- [ ] Agent Shaker is discoverable via `/.well-known/agent-card.json`
- [ ] All A2A task lifecycle endpoints are implemented and functional
- [ ] Streaming support via SSE is operational
- [ ] Existing MCP contexts can be shared as A2A artifacts
- [ ] Agent Shaker can communicate with external A2A agents as a client
- [ ] MCP tools can delegate tasks to A2A agents
- [ ] Comprehensive documentation is available
- [ ] Integration tests validate A2A compatibility

---

## Story 1: Agent Discovery - Agent Card Endpoint

### Story Description
As a **client application**, I want to **discover Agent Shaker via the A2A agent-card endpoint**, so that I can **understand its capabilities and available endpoints**.

### Background
The A2A Protocol requires agents to publish a standardized `agent-card.json` at `/.well-known/agent-card.json`. This JSON document describes the agent's identity, capabilities, supported protocols, and API endpoints.

### Technical Requirements

#### 1.1 Endpoint Specification
- **URL:** `GET /.well-known/agent-card.json`
- **Method:** GET
- **Content-Type:** `application/json`
- **Response Code:** 200 OK

#### 1.2 Agent Card Schema (Go Struct)

```go
package models

type AgentCard struct {
    Name         string         `json:"name"`
    Description  string         `json:"description"`
    Version      string         `json:"version"`
    Capabilities []Capability   `json:"capabilities"`
    Endpoints    []Endpoint     `json:"endpoints"`
    Metadata     map[string]any `json:"metadata,omitempty"`
}

type Capability struct {
    Type        string `json:"type"`        // e.g., "task", "streaming", "artifacts"
    Description string `json:"description"`
}

type Endpoint struct {
    Path        string            `json:"path"`
    Method      string            `json:"method"`
    Description string            `json:"description"`
    Protocol    string            `json:"protocol"` // "A2A", "MCP"
    Params      map[string]string `json:"params,omitempty"`
}
```

#### 1.3 Example Response

```json
{
  "name": "Agent Shaker",
  "description": "MCP-compatible context management server with A2A support",
  "version": "1.0.0",
  "capabilities": [
    {
      "type": "task",
      "description": "Execute tasks and manage context-aware operations"
    },
    {
      "type": "streaming",
      "description": "Real-time task updates via SSE"
    },
    {
      "type": "artifacts",
      "description": "Share markdown contexts as artifacts"
    }
  ],
  "endpoints": [
    {
      "path": "/a2a/v1/message",
      "method": "POST",
      "description": "Send a task to the agent",
      "protocol": "A2A"
    },
    {
      "path": "/a2a/v1/message:stream",
      "method": "POST",
      "description": "Send a task and stream results",
      "protocol": "A2A"
    },
    {
      "path": "/a2a/v1/tasks",
      "method": "GET",
      "description": "List all tasks",
      "protocol": "A2A"
    },
    {
      "path": "/a2a/v1/tasks/{taskId}",
      "method": "GET",
      "description": "Get task details",
      "protocol": "A2A"
    }
  ],
  "metadata": {
    "supported_protocols": ["A2A", "MCP"],
    "websocket_available": true
  }
}
```

#### 1.4 Implementation Details (Go)

**File:** `internal/a2a/server/agent_card.go`

```go
package server

import (
    "encoding/json"
    "net/http"
    "agent-shaker/internal/a2a/models"
)

type AgentCardHandler struct {
    version string
}

func NewAgentCardHandler(version string) *AgentCardHandler {
    return &AgentCardHandler{version: version}
}

func (h *AgentCardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    card := h.generateAgentCard()

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(card)
}

func (h *AgentCardHandler) generateAgentCard() models.AgentCard {
    return models.AgentCard{
        Name:        "Agent Shaker",
        Description: "MCP-compatible context management server with A2A support",
        Version:     h.version,
        Capabilities: []models.Capability{
            {Type: "task", Description: "Execute tasks and manage context-aware operations"},
            {Type: "streaming", Description: "Real-time task updates via SSE"},
            {Type: "artifacts", Description: "Share markdown contexts as artifacts"},
        },
        Endpoints: []models.Endpoint{
            {Path: "/a2a/v1/message", Method: "POST", Description: "Send a task to the agent", Protocol: "A2A"},
            {Path: "/a2a/v1/message:stream", Method: "POST", Description: "Send a task and stream results", Protocol: "A2A"},
            {Path: "/a2a/v1/tasks", Method: "GET", Description: "List all tasks", Protocol: "A2A"},
            {Path: "/a2a/v1/tasks/{taskId}", Method: "GET", Description: "Get task details", Protocol: "A2A"},
        },
        Metadata: map[string]any{
            "supported_protocols": []string{"A2A", "MCP"},
            "websocket_available": true,
        },
    }
}
```

**Router Registration (Chi example):**

```go
r := chi.NewRouter()
r.Get("/.well-known/agent-card.json", NewAgentCardHandler("1.0.0").ServeHTTP)
```

### Acceptance Criteria
- [ ] `GET /.well-known/agent-card.json` returns 200 OK
- [ ] Response is valid JSON matching the A2A agent card schema
- [ ] All A2A endpoints are listed in the card
- [ ] Capabilities accurately reflect Agent Shaker's features
- [ ] Version number is dynamic and configurable

### Testing
- Unit test for agent card generation
- Integration test to verify endpoint accessibility
- JSON schema validation test

---

## Story 2: A2A Task Lifecycle Implementation

### Story Description
As an **external A2A client**, I want to **send tasks to Agent Shaker, retrieve task status, and list all tasks**, so that I can **interact with Agent Shaker asynchronously**.

### Background
The A2A Protocol defines three core endpoints for task management:
1. **POST /a2a/v1/message** - Send a new task
2. **GET /a2a/v1/tasks/{taskId}** - Get task details
3. **GET /a2a/v1/tasks** - List all tasks

These endpoints enable asynchronous task execution and status monitoring.

### Technical Requirements

#### 2.1 Data Models (Go)

**File:** `internal/a2a/models/task.go`

```go
package models

import "time"

type SendMessageRequest struct {
    Message  Message  `json:"message"`
    Metadata Metadata `json:"metadata,omitempty"`
}

type Message struct {
    Content string             `json:"content"`
    Context map[string]any     `json:"context,omitempty"`
    Format  string             `json:"format,omitempty"` // "text", "markdown"
}

type Metadata struct {
    Priority    string            `json:"priority,omitempty"`    // "low", "medium", "high"
    Timeout     int               `json:"timeout,omitempty"`     // seconds
    CallbackURL string            `json:"callback_url,omitempty"`
    Extra       map[string]string `json:"extra,omitempty"`
}

type SendMessageResponse struct {
    TaskID    string `json:"task_id"`
    Status    string `json:"status"`
    CreatedAt string `json:"created_at"`
}

type Task struct {
    ID          string     `json:"id"`
    Status      string     `json:"status"` // "pending", "running", "completed", "failed"
    Message     Message    `json:"message"`
    Result      *Result    `json:"result,omitempty"`
    Artifacts   []Artifact `json:"artifacts,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Result struct {
    Content string         `json:"content"`
    Format  string         `json:"format"`
    Data    map[string]any `json:"data,omitempty"`
}

type TaskListResponse struct {
    Tasks      []Task `json:"tasks"`
    TotalCount int    `json:"total_count"`
}
```

#### 2.2 Task Storage Interface

**File:** `internal/task/store.go`

```go
package task

import (
    "context"
    "agent-shaker/internal/a2a/models"
)

type Store interface {
    CreateTask(ctx context.Context, task *models.Task) error
    GetTask(ctx context.Context, taskID string) (*models.Task, error)
    UpdateTask(ctx context.Context, task *models.Task) error
    ListTasks(ctx context.Context, filter *Filter) ([]models.Task, error)
}

type Filter struct {
    Status string
    Limit  int
    Offset int
}

// FileStore implements Store using file-based persistence
type FileStore struct {
    basePath string
}

func NewFileStore(basePath string) *FileStore {
    return &FileStore{basePath: basePath}
}

// Implement CreateTask, GetTask, UpdateTask, ListTasks...
```

#### 2.3 Task Manager

**File:** `internal/task/manager.go`

```go
package task

import (
    "context"
    "fmt"
    "time"
    "github.com/google/uuid"
    "agent-shaker/internal/a2a/models"
)

type Manager struct {
    store Store
}

func NewManager(store Store) *Manager {
    return &Manager{store: store}
}

func (m *Manager) CreateTask(ctx context.Context, req *models.SendMessageRequest) (*models.Task, error) {
    task := &models.Task{
        ID:        uuid.New().String(),
        Status:    "pending",
        Message:   req.Message,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := m.store.CreateTask(ctx, task); err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }

    // Trigger async execution
    go m.executeTask(task.ID)

    return task, nil
}

func (m *Manager) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
    return m.store.GetTask(ctx, taskID)
}

func (m *Manager) ListTasks(ctx context.Context, filter *Filter) ([]models.Task, error) {
    return m.store.ListTasks(ctx, filter)
}

func (m *Manager) executeTask(taskID string) {
    // Implementation: execute task logic
    // Update task status to "running"
    // Process message content
    // Update task with result
    // Set status to "completed" or "failed"
}
```

#### 2.4 HTTP Handlers

**File:** `internal/a2a/server/handler.go`

```go
package server

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
    "agent-shaker/internal/a2a/models"
    "agent-shaker/internal/task"
)

type A2AHandler struct {
    taskManager *task.Manager
}

func NewA2AHandler(tm *task.Manager) *A2AHandler {
    return &A2AHandler{taskManager: tm}
}

// POST /a2a/v1/message
func (h *A2AHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
    var req models.SendMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    task, err := h.taskManager.CreateTask(r.Context(), &req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    resp := models.SendMessageResponse{
        TaskID:    task.ID,
        Status:    task.Status,
        CreatedAt: task.CreatedAt.Format(time.RFC3339),
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(resp)
}

// GET /a2a/v1/tasks/{taskId}
func (h *A2AHandler) GetTask(w http.ResponseWriter, r *http.Request) {
    taskID := chi.URLParam(r, "taskId")

    task, err := h.taskManager.GetTask(r.Context(), taskID)
    if err != nil {
        http.Error(w, "task not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(task)
}

// GET /a2a/v1/tasks
func (h *A2AHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
    filter := &task.Filter{
        Status: r.URL.Query().Get("status"),
        Limit:  100, // default
    }

    tasks, err := h.taskManager.ListTasks(r.Context(), filter)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    resp := models.TaskListResponse{
        Tasks:      tasks,
        TotalCount: len(tasks),
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resp)
}
```

#### 2.5 Router Registration

```go
func RegisterA2ARoutes(r chi.Router, handler *A2AHandler) {
    r.Route("/a2a/v1", func(r chi.Router) {
        r.Post("/message", handler.SendMessage)
        r.Get("/tasks", handler.ListTasks)
        r.Get("/tasks/{taskId}", handler.GetTask)
    })
}
```

### Acceptance Criteria
- [ ] POST `/a2a/v1/message` creates a task and returns 202 Accepted with task ID
- [ ] GET `/a2a/v1/tasks/{taskId}` returns task details
- [ ] GET `/a2a/v1/tasks` returns a list of tasks with optional status filtering
- [ ] Tasks are persisted to file storage
- [ ] Task execution happens asynchronously
- [ ] Invalid requests return appropriate error codes (400, 404, 500)

### Testing
- Unit tests for task manager
- Unit tests for file store
- Integration tests for all three endpoints
- Edge case tests (invalid JSON, missing task ID, etc.)

---

## Story 3: Streaming Support for Real-time Updates

### Story Description
As an **external A2A client**, I want to **receive real-time task updates via Server-Sent Events (SSE)**, so that I can **monitor long-running tasks without polling**.

### Background
The A2A Protocol supports streaming via `POST /a2a/v1/message:stream`. This endpoint uses SSE to push task status updates, partial results, and completion notifications to clients in real-time.

### Technical Requirements

#### 3.1 SSE Event Format

```
event: status
data: {"task_id": "123", "status": "running"}

event: result
data: {"task_id": "123", "content": "Partial result..."}

event: completed
data: {"task_id": "123", "status": "completed", "result": {...}}
```

#### 3.2 Streaming Handler

**File:** `internal/a2a/server/streaming.go`

```go
package server

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    "agent-shaker/internal/a2a/models"
    "agent-shaker/internal/task"
)

type StreamingHandler struct {
    taskManager *task.Manager
}

func NewStreamingHandler(tm *task.Manager) *StreamingHandler {
    return &StreamingHandler{taskManager: tm}
}

// POST /a2a/v1/message:stream
func (h *StreamingHandler) StreamMessage(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req models.SendMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    // Create task
    task, err := h.taskManager.CreateTask(r.Context(), &req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    // Subscribe to task updates
    updates := h.subscribeToTask(task.ID)
    defer h.unsubscribeFromTask(task.ID)

    // Stream updates
    ctx := r.Context()
    for {
        select {
        case <-ctx.Done():
            return
        case update := <-updates:
            h.sendSSE(w, flusher, update)
            if update.IsFinal {
                return
            }
        case <-time.After(30 * time.Second):
            // Send keepalive
            fmt.Fprintf(w, ": keepalive\n\n")
            flusher.Flush()
        }
    }
}

func (h *StreamingHandler) sendSSE(w http.ResponseWriter, f http.Flusher, update TaskUpdate) {
    data, _ := json.Marshal(update.Data)
    fmt.Fprintf(w, "event: %s\n", update.Event)
    fmt.Fprintf(w, "data: %s\n\n", data)
    f.Flush()
}

type TaskUpdate struct {
    Event   string
    Data    any
    IsFinal bool
}

func (h *StreamingHandler) subscribeToTask(taskID string) <-chan TaskUpdate {
    // Implementation: subscribe to task updates from task manager
    // This could use channels, event bus, or existing WebSocket hub
    updates := make(chan TaskUpdate, 10)

    // Example: poll task status (replace with event-based approach)
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
            task, err := h.taskManager.GetTask(context.Background(), taskID)
            if err != nil {
                continue
            }

            update := TaskUpdate{
                Event: "status",
                Data: map[string]any{
                    "task_id": task.ID,
                    "status":  task.Status,
                },
            }

            if task.Status == "completed" || task.Status == "failed" {
                update.Event = "completed"
                update.Data = task
                update.IsFinal = true
                updates <- update
                return
            }

            updates <- update
        }
    }()

    return updates
}

func (h *StreamingHandler) unsubscribeFromTask(taskID string) {
    // Cleanup
}
```

#### 3.3 Integration with Existing WebSocket Hub

**Enhancement to:** `internal/websocket/hub.go`

```go
package websocket

// Add task update broadcasting capability
type Hub struct {
    // existing fields...
    taskSubscribers map[string][]chan TaskUpdate
}

func (h *Hub) SubscribeToTask(taskID string) <-chan TaskUpdate {
    ch := make(chan TaskUpdate, 10)
    h.taskSubscribers[taskID] = append(h.taskSubscribers[taskID], ch)
    return ch
}

func (h *Hub) BroadcastTaskUpdate(taskID string, update TaskUpdate) {
    for _, ch := range h.taskSubscribers[taskID] {
        select {
        case ch <- update:
        default:
            // Channel full, skip
        }
    }
}
```

### Acceptance Criteria
- [ ] POST `/a2a/v1/message:stream` initiates SSE stream
- [ ] Client receives task status updates in real-time
- [ ] Stream closes when task completes or fails
- [ ] Keepalive messages prevent connection timeout
- [ ] Multiple concurrent streams are supported
- [ ] Graceful handling of client disconnections

### Testing
- Integration test for SSE streaming
- Test concurrent streams
- Test client disconnect handling
- Performance test with multiple streams

---

## Story 4: Context Sharing as A2A Artifacts

### Story Description
As an **agent**, I want to **expose Agent Shaker's markdown contexts as A2A artifacts**, so that **other agents can access and utilize shared knowledge**.

### Background
Agent Shaker manages markdown documentation as "contexts." These contexts should be mappable to A2A artifacts, enabling other agents to discover and retrieve them.

### Technical Requirements

#### 4.1 Artifact Model

**File:** `internal/a2a/models/artifact.go`

```go
package models

type Artifact struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Type        string         `json:"type"` // "markdown", "json", "binary"
    ContentType string         `json:"content_type"`
    Content     string         `json:"content,omitempty"`
    URL         string         `json:"url,omitempty"`
    Size        int64          `json:"size"`
    CreatedAt   string         `json:"created_at"`
    Metadata    map[string]any `json:"metadata,omitempty"`
}
```

#### 4.2 Context-to-Artifact Mapper

**File:** `internal/a2a/mapper/mcp_a2a.go`

```go
package mapper

import (
    "agent-shaker/internal/a2a/models"
    "agent-shaker/internal/context"
)

func ContextToArtifact(ctx *context.Context, baseURL string) models.Artifact {
    return models.Artifact{
        ID:          ctx.ID,
        Name:        ctx.Name,
        Type:        "markdown",
        ContentType: "text/markdown",
        Content:     ctx.Content,
        URL:         fmt.Sprintf("%s/a2a/v1/artifacts/%s", baseURL, ctx.ID),
        Size:        int64(len(ctx.Content)),
        CreatedAt:   ctx.CreatedAt.Format(time.RFC3339),
        Metadata: map[string]any{
            "tags":        ctx.Tags,
            "description": ctx.Description,
        },
    }
}
```

#### 4.3 Artifact Endpoints

**New endpoints:**
- `GET /a2a/v1/artifacts` - List all artifacts
- `GET /a2a/v1/artifacts/{artifactId}` - Get artifact details

**File:** `internal/a2a/server/artifact_handler.go`

```go
package server

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
    "agent-shaker/internal/context"
    "agent-shaker/internal/a2a/mapper"
)

type ArtifactHandler struct {
    contextStorage context.Storage
    baseURL        string
}

func NewArtifactHandler(cs context.Storage, baseURL string) *ArtifactHandler {
    return &ArtifactHandler{
        contextStorage: cs,
        baseURL:        baseURL,
    }
}

func (h *ArtifactHandler) ListArtifacts(w http.ResponseWriter, r *http.Request) {
    contexts, err := h.contextStorage.ListContexts(r.Context())
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    artifacts := make([]models.Artifact, len(contexts))
    for i, ctx := range contexts {
        artifacts[i] = mapper.ContextToArtifact(&ctx, h.baseURL)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "artifacts": artifacts,
        "total":     len(artifacts),
    })
}

func (h *ArtifactHandler) GetArtifact(w http.ResponseWriter, r *http.Request) {
    artifactID := chi.URLParam(r, "artifactId")

    ctx, err := h.contextStorage.GetContext(r.Context(), artifactID)
    if err != nil {
        http.Error(w, "artifact not found", http.StatusNotFound)
        return
    }

    artifact := mapper.ContextToArtifact(ctx, h.baseURL)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(artifact)
}
```

#### 4.4 Task-Artifact Association

Tasks can produce or reference artifacts:

```go
type Task struct {
    // ... existing fields
    Artifacts []Artifact `json:"artifacts,omitempty"`
}

// When task execution creates/references a context
func (m *Manager) executeTask(taskID string) {
    // ... task logic

    // Associate context as artifact
    ctx := // ... get or create context
    artifact := mapper.ContextToArtifact(ctx, m.baseURL)

    task.Artifacts = append(task.Artifacts, artifact)
    m.store.UpdateTask(context.Background(), task)
}
```

### Acceptance Criteria
- [ ] GET `/a2a/v1/artifacts` returns all contexts as artifacts
- [ ] GET `/a2a/v1/artifacts/{id}` returns specific artifact details
- [ ] Artifacts include full markdown content
- [ ] Tasks can reference artifacts in their results
- [ ] Artifact metadata includes context tags and descriptions

### Testing
- Unit tests for context-to-artifact mapping
- Integration tests for artifact endpoints
- Test artifact association with tasks

---

## Story 5: A2A Client for External Agent Communication

### Story Description
As **Agent Shaker**, I want to **act as an A2A client to discover and communicate with external A2A agents**, so that I can **delegate specialized tasks to other agents**.

### Background
Agent Shaker should be able to:
1. Discover external A2A agents via their agent-card.json
2. Send tasks to external agents
3. Poll or stream results from external agents

### Technical Requirements

#### 5.1 A2A Client Interface

**File:** `internal/a2a/client/client.go`

```go
package client

import (
    "context"
    "agent-shaker/internal/a2a/models"
)

type Client interface {
    Discover(ctx context.Context, agentURL string) (*models.AgentCard, error)
    SendMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (*models.SendMessageResponse, error)
    GetTask(ctx context.Context, agentURL string, taskID string) (*models.Task, error)
    StreamMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (<-chan TaskUpdate, error)
}

type HTTPClient struct {
    httpClient *http.Client
}

func NewHTTPClient() *HTTPClient {
    return &HTTPClient{
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}
```

#### 5.2 Agent Discovery

**File:** `internal/a2a/client/discovery.go`

```go
package client

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "agent-shaker/internal/a2a/models"
)

func (c *HTTPClient) Discover(ctx context.Context, agentURL string) (*models.AgentCard, error) {
    url := fmt.Sprintf("%s/.well-known/agent-card.json", agentURL)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("discovery failed: status %d", resp.StatusCode)
    }

    var card models.AgentCard
    if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
        return nil, err
    }

    return &card, nil
}
```

#### 5.3 Task Submission

**File:** `internal/a2a/client/task.go`

```go
package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "agent-shaker/internal/a2a/models"
)

func (c *HTTPClient) SendMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (*models.SendMessageResponse, error) {
    url := fmt.Sprintf("%s/a2a/v1/message", agentURL)

    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusAccepted {
        return nil, fmt.Errorf("send message failed: status %d", resp.StatusCode)
    }

    var response models.SendMessageResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, err
    }

    return &response, nil
}

func (c *HTTPClient) GetTask(ctx context.Context, agentURL string, taskID string) (*models.Task, error) {
    url := fmt.Sprintf("%s/a2a/v1/tasks/%s", agentURL, taskID)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("get task failed: status %d", resp.StatusCode)
    }

    var task models.Task
    if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
        return nil, err
    }

    return &task, nil
}
```

#### 5.4 SSE Streaming Client

```go
func (c *HTTPClient) StreamMessage(ctx context.Context, agentURL string, req *models.SendMessageRequest) (<-chan TaskUpdate, error) {
    url := fmt.Sprintf("%s/a2a/v1/message:stream", agentURL)

    body, _ := json.Marshal(req)
    httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Accept", "text/event-stream")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }

    updates := make(chan TaskUpdate, 10)

    go func() {
        defer resp.Body.Close()
        defer close(updates)

        // Parse SSE stream
        scanner := bufio.NewScanner(resp.Body)
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "data: ") {
                data := strings.TrimPrefix(line, "data: ")
                var update TaskUpdate
                json.Unmarshal([]byte(data), &update)
                updates <- update
            }
        }
    }()

    return updates, nil
}
```

### Acceptance Criteria
- [ ] Client can discover external agents
- [ ] Client can send tasks to external agents
- [ ] Client can poll task status
- [ ] Client can stream task updates via SSE
- [ ] Error handling for network failures and invalid responses
- [ ] Configurable timeouts

### Testing
- Unit tests for client methods
- Mock HTTP server for testing
- Integration tests with real A2A agent (if available)

---

## Story 6: Integration with Existing MCP Functionality

### Story Description
As an **MCP user**, I want **Agent Shaker's MCP tools to delegate tasks to A2A agents**, so that I can **leverage external agents from within MCP workflows**.

### Background
Create a new MCP tool `delegate_to_a2a_agent` that allows MCP clients to discover and use external A2A agents.

### Technical Requirements

#### 6.1 New MCP Tool Definition

**File:** `internal/mcp/tools/a2a_delegate.go`

```go
package tools

import (
    "context"
    "encoding/json"
    "agent-shaker/internal/a2a/client"
    "agent-shaker/internal/a2a/models"
)

type A2ADelegateTool struct {
    a2aClient client.Client
}

func NewA2ADelegateTool(a2aClient client.Client) *A2ADelegateTool {
    return &A2ADelegateTool{a2aClient: a2aClient}
}

func (t *A2ADelegateTool) Name() string {
    return "delegate_to_a2a_agent"
}

func (t *A2ADelegateTool) Description() string {
    return "Delegate a task to an external A2A agent"
}

func (t *A2ADelegateTool) InputSchema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "agent_url": map[string]any{
                "type":        "string",
                "description": "Base URL of the A2A agent",
            },
            "message": map[string]any{
                "type":        "string",
                "description": "Message to send to the agent",
            },
            "wait_for_completion": map[string]any{
                "type":        "boolean",
                "description": "Whether to wait for task completion",
                "default":     false,
            },
        },
        "required": []string{"agent_url", "message"},
    }
}

func (t *A2ADelegateTool) Execute(ctx context.Context, input map[string]any) (any, error) {
    agentURL := input["agent_url"].(string)
    message := input["message"].(string)
    waitForCompletion := input["wait_for_completion"].(bool)

    // Discover agent
    card, err := t.a2aClient.Discover(ctx, agentURL)
    if err != nil {
        return nil, fmt.Errorf("failed to discover agent: %w", err)
    }

    // Send message
    req := &models.SendMessageRequest{
        Message: models.Message{
            Content: message,
            Format:  "text",
        },
    }

    resp, err := t.a2aClient.SendMessage(ctx, agentURL, req)
    if err != nil {
        return nil, fmt.Errorf("failed to send message: %w", err)
    }

    result := map[string]any{
        "agent_name": card.Name,
        "task_id":    resp.TaskID,
        "status":     resp.Status,
    }

    if waitForCompletion {
        // Poll for completion
        task, err := t.pollUntilComplete(ctx, agentURL, resp.TaskID)
        if err != nil {
            return nil, err
        }
        result["task"] = task
    }

    return result, nil
}

func (t *A2ADelegateTool) pollUntilComplete(ctx context.Context, agentURL, taskID string) (*models.Task, error) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-ticker.C:
            task, err := t.a2aClient.GetTask(ctx, agentURL, taskID)
            if err != nil {
                return nil, err
            }
            if task.Status == "completed" || task.Status == "failed" {
                return task, nil
            }
        }
    }
}
```

#### 6.2 Register Tool in MCP Server

```go
func (s *MCPServer) RegisterTools() {
    // Existing tools...

    a2aClient := client.NewHTTPClient()
    a2aTool := tools.NewA2ADelegateTool(a2aClient)
    s.RegisterTool(a2aTool)
}
```

### Acceptance Criteria
- [ ] `delegate_to_a2a_agent` tool is available in MCP
- [ ] Tool can discover external agents
- [ ] Tool can send tasks and return task ID
- [ ] Optional waiting for completion works
- [ ] Errors are properly reported to MCP client

### Testing
- Unit tests for A2A delegate tool
- Integration test with MCP client
- Test with mock A2A agent

---

## Story 7: Documentation and Usage Examples

### Story Description
As a **developer**, I want **comprehensive documentation for A2A integration**, so that I can **understand and use the new capabilities**.

### Technical Requirements

#### 7.1 Documentation Files

**File:** `docs/A2A_INTEGRATION.md`

```markdown
# A2A Protocol Integration Guide

## Overview
Agent Shaker now supports the Agent-to-Agent (A2A) Protocol...

## Quick Start
### Discovering Agent Shaker
curl https://your-agent-shaker.com/.well-known/agent-card.json

### Sending a Task
curl -X POST https://your-agent-shaker.com/a2a/v1/message \
  -H "Content-Type: application/json" \
  -d '{"message": {"content": "Analyze this document"}}'

## API Reference
[Detailed endpoint documentation]

## Examples
### Example 1: Simple Task Submission
[Code example]

### Example 2: Streaming Updates
[Code example]

### Example 3: Using A2A from MCP
[Code example]

## Architecture
[Architecture diagrams]

## Configuration
[Environment variables, settings]

## Troubleshooting
[Common issues and solutions]
```

**File:** `api/openapi.yaml`

```yaml
openapi: 3.0.0
info:
  title: Agent Shaker A2A API
  version: 1.0.0
paths:
  /.well-known/agent-card.json:
    get:
      summary: Get agent card
      responses:
        '200':
          description: Agent card
  /a2a/v1/message:
    post:
      summary: Send a message
      # ... detailed spec
```

#### 7.2 README Updates

Update main `README.md` with A2A section:

```markdown
## Features
- MCP Protocol Support
- **A2A Protocol Support** (NEW)
  - Agent discovery
  - Async task management
  - Real-time streaming
  - Artifact sharing
```

#### 7.3 Code Examples

**File:** `examples/a2a/send_task.go`

```go
package main

// Example: Sending a task to Agent Shaker via A2A
```

**File:** `examples/a2a/stream_task.go`

```go
package main

// Example: Streaming task updates
```

### Acceptance Criteria
- [ ] A2A_INTEGRATION.md is complete and accurate
- [ ] OpenAPI spec is available
- [ ] README.md references A2A support
- [ ] At least 3 code examples are provided
- [ ] Architecture diagram is included

### Testing
- Documentation review
- Code examples are tested and work

---

## Story 8: Testing and A2A Compatibility Validation

### Story Description
As a **QA engineer**, I want **comprehensive tests for A2A functionality**, so that I can **ensure compatibility and reliability**.

### Technical Requirements

#### 8.1 Unit Tests

```go
// internal/a2a/server/handler_test.go
package server_test

func TestSendMessage(t *testing.T) {
    // Test cases for SendMessage endpoint
}

func TestGetTask(t *testing.T) {
    // Test cases for GetTask endpoint
}

func TestListTasks(t *testing.T) {
    // Test cases for ListTasks endpoint
}
```

#### 8.2 Integration Tests

**File:** `tests/a2a/integration_test.go`

```go
package a2a_test

import (
    "testing"
    "net/http/httptest"
)

func TestA2AWorkflow(t *testing.T) {
    // 1. Start test server
    // 2. Discover agent card
    // 3. Send task
    // 4. Poll task status
    // 5. Verify completion
}

func TestStreamingWorkflow(t *testing.T) {
    // Test SSE streaming end-to-end
}

func TestArtifactRetrieval(t *testing.T) {
    // Test artifact endpoints
}
```

#### 8.3 Compatibility Tests

**File:** `tests/a2a/compatibility_test.go`

```go
package a2a_test

func TestAgentCardSchema(t *testing.T) {
    // Validate agent card against A2A schema
}

func TestA2AMessageFormat(t *testing.T) {
    // Validate message format compliance
}

func TestTaskLifecycle(t *testing.T) {
    // Test full task lifecycle per A2A spec
}
```

#### 8.4 Test Coverage Requirements
- Unit test coverage: >80%
- Integration test coverage: All major workflows
- Edge cases: Error handling, timeouts, invalid inputs

#### 8.5 CI/CD Integration

**File:** `.github/workflows/a2a-tests.yml`

```yaml
name: A2A Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run A2A tests
        run: go test ./tests/a2a/... -v
```

### Acceptance Criteria
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Compatibility tests validate A2A spec compliance
- [ ] Test coverage meets requirements
- [ ] CI/CD pipeline includes A2A tests

### Testing
- Run test suite
- Code coverage report
- CI/CD pipeline verification

---

## Implementation Timeline

### Phase 1: Foundation (Week 1-2)
- **Story 1:** Agent Discovery - Agent Card Endpoint
- **Story 2:** A2A Task Lifecycle Implementation

### Phase 2: Real-time & Sharing (Week 3)
- **Story 3:** Streaming Support
- **Story 4:** Context Sharing as Artifacts

### Phase 3: Client & Integration (Week 4)
- **Story 5:** A2A Client for External Agents
- **Story 6:** MCP Integration

### Phase 4: Documentation & Testing (Week 5)
- **Story 7:** Documentation
- **Story 8:** Testing & Validation

**Total Estimated Time:** 5-6 weeks

---

## Definition of Done

### Epic-Level DoD
- [ ] All 8 stories are completed
- [ ] Agent Shaker is A2A-compatible
- [ ] All acceptance criteria met
- [ ] Documentation is complete
- [ ] Test coverage >80%
- [ ] CI/CD pipeline passes
- [ ] Code reviewed and merged
- [ ] Production deployment successful

### Story-Level DoD
- [ ] Code implemented in Go
- [ ] Unit tests written and passing
- [ ] Integration tests passing
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] No critical bugs

---

## Technical Dependencies

### Go Libraries
- `github.com/go-chi/chi/v5` - HTTP router
- `github.com/google/uuid` - UUID generation
- Standard library: `net/http`, `encoding/json`, `context`, `time`

### External Services
- None (self-contained)

### Infrastructure
- File system for task/context storage
- Existing WebSocket hub

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| SSE complexity in Go | Medium | Use proven patterns, thorough testing |
| File storage scalability | Low | Document future migration to DB |
| A2A spec changes | Medium | Monitor spec, version API endpoints |
| Integration with existing MCP | High | Careful abstraction, unified task manager |

---

## Notes
- This specification assumes Agent Shaker uses a file-based storage system. Adapt to actual storage implementation.
- All code examples are illustrative; adjust based on existing codebase structure.
- A2A Protocol spec: https://a2a-protocol.org/
- Go version: 1.21+

---

**End of Technical Specification**