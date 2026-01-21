# Docker Compose Setup for Agent Shaker

## Overview
This Docker Compose setup provides a complete development environment with:
- **PostgreSQL** (Database) - Port 5433
- **MCP Server** (Go API + WebSocket) - Port 8080
- **Web App** (Vue.js UI) - Port 80

## Quick Start

### 1. Start All Services
```bash
docker-compose up --build
```

### 2. Access the Application
- **Web UI**: http://localhost
- **API**: http://localhost/api
- **WebSocket**: ws://localhost/ws
- **Database**: localhost:5433 (from host machine)

### 3. Stop Services
```bash
docker-compose down
```

## Service Details

### PostgreSQL
- **Image**: postgres:16-alpine
- **Database**: mcp_tracker
- **User**: mcp
- **Password**: secret
- **Port**: 5433 (host) → 5432 (container)
- **Health Check**: Enabled

### MCP Server (Go)
- **Build**: Current directory (.)
- **Port**: 8080
- **Environment**:
  - DATABASE_URL: postgres://mcp:secret@postgres:5432/mcp_tracker?sslmode=disable
  - PORT: 8080
- **Depends on**: PostgreSQL (healthy)
- **Restart**: unless-stopped

### Web App (Vue.js)
- **Build**: ./web directory
- **Port**: 80
- **Depends on**: MCP Server (started)
- **Restart**: unless-stopped
- **Nginx Config**: Proxies API/WebSocket to mcp-server:8080

## Development Workflow

### First Time Setup
```bash
# Clone repository
git clone <repository-url>
cd agent-shaker

# Start all services (builds and runs)
docker-compose up --build
```

### Development with Hot Reload
For development with hot reload, run the frontend separately:
```bash
# Terminal 1: Start backend + database
docker-compose up postgres mcp-server

# Terminal 2: Start frontend with hot reload
cd web
npm run dev
```

### Database Management
```bash
# Access PostgreSQL shell
docker-compose exec postgres psql -U mcp -d mcp_tracker

# View logs
docker-compose logs postgres

# Reset database
docker-compose down -v  # Removes volumes
docker-compose up --build
```

## API Testing

### Using the Verification Script
```powershell
# Start services first
docker-compose up -d

# Run API verification (from host)
.\scripts\verify-api.ps1
```

### Manual API Testing
```bash
# Health check
curl http://localhost/api/health

# List projects
curl http://localhost/api/projects

# WebSocket test (requires WebSocket client)
# ws://localhost/ws?project_id=<uuid>
```

## Troubleshooting

### Common Issues

1. **Port Conflicts**
   - Ensure ports 80, 8080, 5433 are available
   - Check: `netstat -an | findstr :80`

2. **Database Connection Issues**
   - Wait for PostgreSQL health check to pass
   - Check logs: `docker-compose logs postgres`

3. **Build Failures**
   - Clear Docker cache: `docker system prune -a`
   - Rebuild: `docker-compose build --no-cache`

4. **WebSocket Issues**
   - Ensure MCP server is running
   - Check WebSocket URL: `ws://localhost/ws?project_id=<uuid>`

### Logs and Debugging
```bash
# All service logs
docker-compose logs

# Specific service logs
docker-compose logs web
docker-compose logs mcp-server
docker-compose logs postgres

# Follow logs in real-time
docker-compose logs -f mcp-server
```

## Production Deployment

For production, consider:
1. Using environment-specific docker-compose files
2. Adding SSL/TLS termination
3. Setting up proper secrets management
4. Configuring health checks and monitoring
5. Using Docker Swarm or Kubernetes for orchestration

## Architecture

```
Browser → Nginx:80 → Vue.js SPA (static files)
Browser → Nginx:80/api/* → Go Server:8080/api/* (REST API)
Browser → Nginx:80/ws → Go Server:8080/ws (WebSocket)
AI Agents → Go Server:8080/api/* (direct API access)
Go Server → PostgreSQL:5432 (database)
```
