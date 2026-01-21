# 🎯 MCP Multi-Agent System - Quick Reference Card

## 🚀 Start/Stop Commands

```powershell
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# Rebuild and start
docker-compose up -d --build

# View logs
docker-compose logs -f

# Check status
docker ps
```

---

## 🌐 Access Points

| Who | URL | Purpose |
|-----|-----|---------|
| 👤 **Web Users** | http://localhost:80 | Beautiful Vue.js UI |
| 🤖 **AI Agents** | http://localhost:8080 | Direct REST API access |
| 💻 **Developers** | `npm start` | Interactive CLI bridge |

---

## 📡 API Endpoints (Port 8080)

### Root & Health

```bash
GET  /                    # API info (JSON)
GET  /health              # Health check
```

### Projects

```bash
POST /api/projects        # Create project
GET  /api/projects        # List projects
GET  /api/projects/{id}   # Get project
```

### Agents

```bash
POST /api/agents          # Register agent
GET  /api/agents          # List agents (filter by project_id)
PUT  /api/agents/{id}/status  # Update agent status
```

### Tasks

```bash
POST /api/tasks           # Create task
GET  /api/tasks           # List tasks (filter by project_id)
GET  /api/tasks/{id}      # Get task
PUT  /api/tasks/{id}      # Update task
```

### Contexts (Documentation)

```bash
POST   /api/contexts      # Add documentation
GET    /api/contexts      # List documentation
GET    /api/contexts/{id} # Get documentation
PUT    /api/contexts/{id} # Update documentation
DELETE /api/contexts/{id} # Delete documentation
```

### WebSocket

```bash
WS   /ws?project_id={id}  # Real-time updates
```

---

## 🧪 Quick Tests

```powershell
# Test Go API directly
curl http://localhost:8080/

# Test API endpoint
curl http://localhost:8080/api/agents

# Test via Nginx
curl http://localhost:80/api/projects

# Test health
curl http://localhost:8080/health

# Open web UI
Start-Process http://localhost:80
```

---

## 🏗️ Architecture at a Glance

```
Web Browser :80
    ↓ HTTP
┌─────────────┐
│   Nginx     │ (Serves Vue.js, Proxies API)
└──────┬──────┘
       │ Proxy
       ↓
┌─────────────┐ ←── AI Agents :8080 (Direct)
│  Go Server  │ (REST API + WebSocket)
└──────┬──────┘
       │ SQL
       ↓
┌─────────────┐
│ PostgreSQL  │ :5433
└─────────────┘
```

---

## 📦 Container Ports

| Container | Port Mapping | Purpose |
|-----------|--------------|---------|
| postgres | 5433:5432 | Database |
| mcp-server | 8080:8080 | Go API (Direct + Proxy) |
| web | 80:80 | Nginx + Vue.js |

---

## 🎨 What's Different Now?

### ✅ Go Server (Port 8080)

**Before**: Served Vue.js + API  
**Now**: **Only API** (no Vue.js)

```go
// Root returns JSON, not HTML
GET / → {"name": "MCP API", "endpoints": {...}}
```

### ✅ Nginx (Port 80)

**Before**: Basic proxy  
**Now**: **Enhanced** with clear docs

```nginx
# API/WS proxied to Go:8080
location /api/ { proxy_pass http://mcp-server:8080; }
location /ws { proxy_pass http://mcp-server:8080; }

# SPA served from Nginx
location / { try_files $uri /index.html; }
```

---

## 🤖 For AI Agents

**Use**: http://localhost:8080 (Direct access, no Nginx)

**Example**:
```python
import requests

# List agents
response = requests.get("http://localhost:8080/api/agents")
agents = response.json()

# Create task
task = {
    "project_id": "...",
    "title": "Implement feature",
    "description": "...",
    "priority": "high",
    "created_by": "agent-id"
}
response = requests.post("http://localhost:8080/api/tasks", json=task)
```

---

## 💻 MCP Bridge CLI

**Setup**:
```powershell
npm install
npm start
```

**Commands**:
- `list agents` - Show all agents
- `list projects` - Show all projects
- `list tasks` - Show all tasks
- `create task` - Interactive task creation
- `help` - Show available commands
- `exit` - Quit

---

## 📚 Documentation

| File | Quick Description |
|------|-------------------|
| `ARCHITECTURE_SUMMARY.md` | 👈 **This file** - Quick reference |
| `CLEAN_ARCHITECTURE.md` | Detailed architecture with diagrams |
| `ARCHITECTURE_VERIFICATION.md` | Test results and verification |
| `API.md` | Complete API documentation |
| `COPILOT_INTEGRATION.md` | GitHub Copilot integration guide |
| `MCP_QUICKSTART.md` | MCP bridge quick start |

---

## 🔧 Troubleshooting

### Container not starting?

```powershell
docker-compose logs [service-name]
docker-compose restart [service-name]
```

### Can't access API?

```powershell
# Check container is running
docker ps | Select-String "8080"

# Test direct access
curl http://localhost:8080/health
```

### Web UI not loading?

```powershell
# Check Nginx logs
docker logs agent-shaker-web-1

# Verify Nginx is running
docker ps | Select-String "web"
```

---

## 🎯 Key Points to Remember

1. **Go Server (8080)**: Only API, no Vue.js
2. **Nginx (80)**: Serves Vue.js + Proxies API
3. **AI Agents**: Use port 8080 directly
4. **Web Users**: Use port 80 (Nginx)
5. **Clean Separation**: Each layer has one job

---

## ✅ Current Status

- ✅ PostgreSQL: Healthy with sample data
- ✅ Go Server: Running, serving JSON only
- ✅ Nginx: Running, proxying correctly
- ✅ Vue.js: Loaded and accessible
- ✅ Architecture: Clean and verified

---

**Last Updated**: January 21, 2026  
**Status**: All systems operational 🟢  
**Ready for**: Production use and AI agent integration 🚀
