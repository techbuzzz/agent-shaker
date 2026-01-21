# MCP Multi-Agent System - Clean Architecture

## Overview

This document describes the **clean separation** between the Go MCP API server, Vue.js frontend, and how AI agents interact with the system.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Client Layer                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌──────────────────┐              ┌─────────────────────┐          │
│  │   Web Browser    │              │  AI Agents          │          │
│  │   (Human Users)  │              │  (VS Code Copilot)  │          │
│  └────────┬─────────┘              └──────────┬──────────┘          │
│           │                                    │                     │
└───────────┼────────────────────────────────────┼─────────────────────┘
            │                                    │
            │ HTTP/WS                            │ HTTP (Direct)
            │ Port 80                            │ Port 8080
            ▼                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       Presentation Layer                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    Nginx Web Server (Port 80)                   │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  • Serves Vue.js SPA (HTML, CSS, JS)                           │ │
│  │  • Proxies /api/* → Go Server:8080/api/*                       │ │
│  │  • Proxies /ws → Go Server:8080/ws                             │ │
│  │  • Client-side routing: /* → index.html                        │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
└──────────────────────────────┼───────────────────────────────────────┘
                               │ Proxy
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Application Layer                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │              Go MCP API Server (Port 8080)                      │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  REST API Endpoints:                                            │ │
│  │    GET  /                     → API info (JSON)                 │ │
│  │    GET  /health               → Health check                    │ │
│  │    POST /api/projects         → Create project                  │ │
│  │    GET  /api/projects         → List projects                   │ │
│  │    POST /api/agents           → Register agent                  │ │
│  │    GET  /api/agents           → List agents                     │ │
│  │    POST /api/tasks            → Create task                     │ │
│  │    GET  /api/tasks            → List tasks                      │ │
│  │    PUT  /api/tasks/{id}       → Update task                     │ │
│  │    POST /api/contexts         → Add documentation               │ │
│  │    GET  /api/contexts         → List documentation              │ │
│  │    PUT  /api/contexts/{id}    → Update documentation            │ │
│  │    DELETE /api/contexts/{id}  → Delete documentation            │ │
│  │    GET  /api/docs             → API documentation               │ │
│  │                                                                  │ │
│  │  WebSocket Endpoint:                                            │ │
│  │    WS  /ws?project_id={id}    → Real-time updates               │ │
│  │                                                                  │ │
│  │  Features:                                                       │ │
│  │    • CORS enabled (all origins)                                 │ │
│  │    • Request logging middleware                                 │ │
│  │    • Error recovery middleware                                  │ │
│  │    • 10MB request size limit                                    │ │
│  │    • No Vue.js serving (pure API)                               │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                     WebSocket Hub                               │ │
│  │  • Real-time task updates                                       │ │
│  │  • Agent status broadcasts                                      │ │
│  │  • Project-based rooms                                          │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
└──────────────────────────────┼───────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Data Layer                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │              PostgreSQL Database (Port 5433)                    │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  Tables:                                                         │ │
│  │    • projects    → Project definitions                          │ │
│  │    • agents      → Registered AI agents                         │ │
│  │    • tasks       → Task assignments                             │ │
│  │    • contexts    → Documentation & knowledge base               │ │
│  │                                                                  │ │
│  │  Connection Pool:                                                │ │
│  │    • Max open: 25 connections                                   │ │
│  │    • Max idle: 5 connections                                    │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

## Access Patterns

### 1. Web Browser Users (via Nginx)

**URL**: `http://localhost:80`

**Flow**:
```
Browser → Nginx:80 → Vue.js SPA
                   ↓
              User clicks button
                   ↓
       Vue.js makes API call to /api/tasks
                   ↓
       Nginx proxies to Go:8080/api/tasks
                   ↓
              Go processes request
                   ↓
              Returns JSON data
                   ↓
       Vue.js renders in UI
```

**Features**:
- Beautiful Vue.js interface
- Real-time WebSocket updates
- Client-side routing
- Responsive design with Tailwind CSS

### 2. AI Agents (Direct API Access)

**URL**: `http://localhost:8080`

**Flow**:
```
AI Agent → Go:8080/api/tasks (Direct)
              ↓
         Go processes request
              ↓
         Returns JSON data
              ↓
    AI Agent processes response
```

**Features**:
- No nginx overhead
- Direct database access
- Full API access
- No CORS restrictions

### 3. MCP Bridge (Command Line Tool)

**URL**: `http://localhost:8080`

**Flow**:
```
mcp-bridge.js → Go:8080/api/* (Direct)
                   ↓
              Interactive CLI
                   ↓
       Commands: list agents, create task, etc.
```

## Port Configuration

| Service | Port | Purpose | Access |
|---------|------|---------|--------|
| **PostgreSQL** | 5433 | Database | Internal only |
| **Go MCP Server** | 8080 | API + WebSocket | Direct (AI agents, MCP bridge) |
| **Nginx** | 80 | Vue.js SPA + API Proxy | Web browsers |

## Environment Variables

### Go MCP Server

```bash
DATABASE_URL=postgres://mcp:secret@postgres:5432/mcp_tracker?sslmode=disable
PORT=8080
```

### Docker Compose

```yaml
services:
  postgres:
    ports: ["5433:5432"]  # External:Internal
  
  mcp-server:
    ports: ["8080:8080"]  # Go API - direct access
    environment:
      DATABASE_URL: postgres://mcp:secret@postgres:5432/mcp_tracker?sslmode=disable
      PORT: 8080
  
  web:
    ports: ["80:80"]      # Nginx - web UI
```

## API Response Examples

### Root Endpoint (/)

```bash
# Direct Go server access
curl http://localhost:8080/

# Response
{
  "name": "MCP Multi-Agent Task Tracker API",
  "version": "1.0.0",
  "endpoints": {
    "projects": "/api/projects",
    "agents": "/api/agents",
    "tasks": "/api/tasks",
    "contexts": "/api/contexts",
    "docs": "/api/docs",
    "websocket": "/ws",
    "health": "/health"
  },
  "documentation": "/api/docs"
}
```

### List Agents

```bash
# Via Nginx (from browser or curl)
curl http://localhost:80/api/agents?project_id=<uuid>

# Direct Go server (from AI agent)
curl http://localhost:8080/api/agents?project_id=<uuid>

# Both return identical JSON
[
  {
    "id": "uuid",
    "project_id": "uuid",
    "name": "InvoiceAI-Frontend",
    "role": "frontend",
    "team": "development",
    "status": "idle",
    "last_seen": "2026-01-21T10:30:00Z",
    "created_at": "2026-01-20T09:00:00Z"
  }
]
```

## Key Benefits of This Architecture

### ✅ Clean Separation

- **Go Server**: Pure API, no frontend concerns
- **Nginx**: Pure frontend, reverse proxy only
- **No mixing**: Each layer has single responsibility

### ✅ Multiple Access Methods

- **Web UI**: Beautiful interface via Nginx
- **Direct API**: Fast access for AI agents
- **MCP Bridge**: Command-line interface

### ✅ Scalability

- Can scale Go server independently
- Can add more nginx instances
- Database connection pooling

### ✅ Development Workflow

- Frontend developers work in `web/` folder
- Backend developers work in `internal/` and `cmd/`
- No conflicts between teams

### ✅ AI Agent Integration

- Direct access to API (no proxy overhead)
- Full CORS support
- Real-time WebSocket updates

## Configuration Files

### 1. Go Server: `cmd/server/main.go`

**What it does**:
- ✅ Serves REST API endpoints (`/api/*`)
- ✅ Serves WebSocket endpoint (`/ws`)
- ✅ Serves health check (`/health`)
- ✅ Serves API info at root (`/`)
- ❌ Does NOT serve Vue.js (removed)

### 2. Nginx: `web/nginx.conf`

**What it does**:
- ✅ Serves Vue.js SPA from `/usr/share/nginx/html`
- ✅ Proxies `/api/*` to Go server
- ✅ Proxies `/ws` to Go server
- ✅ Handles client-side routing (`/*` → `index.html`)

### 3. Docker Compose: `docker-compose.yml`

**What it does**:
- ✅ Runs PostgreSQL on port 5433
- ✅ Runs Go MCP server on port 8080
- ✅ Runs Nginx on port 80
- ✅ Links all services together

## Testing the Setup

### 1. Test Go API Directly

```powershell
# Health check
curl http://localhost:8080/health

# API info
curl http://localhost:8080/

# List projects (direct)
curl http://localhost:8080/api/projects
```

### 2. Test via Nginx

```powershell
# Web UI (browser)
Start-Process http://localhost:80

# API via proxy
curl http://localhost:80/api/projects
```

### 3. Test with MCP Bridge

```powershell
# Setup and run
.\setup-mcp-bridge.ps1
npm start

# Commands
list agents
list projects
create task
```

## Troubleshooting

### Issue: "API returns HTML instead of JSON"

**Cause**: Nginx proxy rules not ordered correctly

**Solution**: Ensure `/api/` and `/ws` locations come BEFORE `/` location in nginx.conf

### Issue: "Cannot access API directly at :8080"

**Cause**: Docker port not exposed

**Solution**: Check `docker-compose.yml` has `ports: ["8080:8080"]` for mcp-server

### Issue: "CORS errors in browser"

**Cause**: Go server CORS not configured

**Solution**: CORS already enabled in `main.go` with `AllowedOrigins: ["*"]`

### Issue: "WebSocket connection fails"

**Cause**: Proxy timeout too short

**Solution**: Nginx already has `proxy_read_timeout 86400` for WebSocket

## Summary

| Component | Purpose | Port | Serves |
|-----------|---------|------|--------|
| **Go MCP Server** | API + Business Logic | 8080 | JSON API, WebSocket |
| **Nginx** | Reverse Proxy + Static Files | 80 | Vue.js SPA, API proxy |
| **PostgreSQL** | Database | 5433 | Data storage |

**Access Points**:
- 🌐 **Web UI**: http://localhost:80 (humans)
- 🤖 **API Direct**: http://localhost:8080 (AI agents)
- 💻 **MCP Bridge**: `npm start` (developers)

This architecture ensures **clean separation**, **multiple access methods**, and **no confusion** between layers! 🎯
