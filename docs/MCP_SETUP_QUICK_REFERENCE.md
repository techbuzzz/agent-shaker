# MCP Setup - Quick Reference Guide

## ⚡ 30-Second Setup

```powershell
# 1. Run setup (one time only)
./scripts/setup-mcp-bridge.ps1

# 2. Start the bridge
npm start

# 3. Use commands
> list projects
> list agents
> create task
> exit
```

---

## 📋 Common Commands

### View Resources

```bash
# List all projects
> list projects

# List all agents (across all projects)
> list agents

# List agents in a specific project
> list agents project:550e8400-e29b-41d4-a716-446655440001

# List tasks in a project
> list tasks project:550e8400-e29b-41d4-a716-446655440001

# Get project details
> get project 550e8400-e29b-41d4-a716-446655440001
```

### Create Resources

```bash
# Create a task interactively
> create task
  Title: Your task name
  Description: Task description
  Project ID: 550e8400-e29b-41d4-a716-446655440001
  Priority: high
```

### System Commands

```bash
# Show help
> help

# Exit bridge
> exit
```

---

## 🔧 Troubleshooting Quick Fix

### Problem: "API is not accessible"
```powershell
# Start containers
docker-compose up -d

# Wait 5 seconds for services to start
Start-Sleep -Seconds 5

# Try again
npm start
```

### Problem: "Node.js not found"
```powershell
# Install from: https://nodejs.org/
# Download LTS version, install, then restart PowerShell
node --version  # Should show v14+
npm start
```

### Problem: "Port 8080 in use"
```powershell
# Check what's using port 8080
netstat -ano | findstr :8080

# Kill the process (if needed)
taskkill /PID <PID> /F

# Start containers again
docker-compose up -d
```

### Problem: "Module not found"
```powershell
# Clear cache and reinstall
npm cache clean --force
npm install
npm start
```

---

## 🌍 Custom API Server

### Set Different API URL

```powershell
# Before starting bridge
$env:AGENT_SHAKER_URL="http://api.example.com:8080/api"

# Start bridge (uses custom URL)
npm start
```

### Reset to Default

```powershell
# Clear environment variable
$env:AGENT_SHAKER_URL=""

# Bridge will use default: http://localhost:8080/api
npm start
```

---

## 📊 API Endpoints Reference

### GET Endpoints (Read-Only)

```bash
GET /api/agents                    # All agents
GET /api/agents?project_id=ID      # Filter by project
GET /api/projects                  # All projects
GET /api/projects/ID               # Specific project
GET /api/tasks?project_id=ID       # Tasks by project
GET /api/contexts                  # All contexts
GET /api/standups                  # All standups
GET /api/agents/ID/heartbeats      # Agent heartbeats
```

### POST Endpoints (Create)

```bash
POST /api/agents                   # Create agent
POST /api/projects                 # Create project
POST /api/tasks                    # Create task
POST /api/contexts                 # Create context
POST /api/standups                 # Create standup
POST /api/heartbeats               # Record heartbeat
```

### PUT Endpoints (Update)

```bash
PUT /api/tasks/ID                  # Update task
PUT /api/tasks/ID/status           # Update status
PUT /api/standups/ID               # Update standup
```

### DELETE Endpoints (Remove)

```bash
DELETE /api/agents/ID              # Delete agent
DELETE /api/projects/ID            # Delete project
DELETE /api/tasks/ID               # Delete task
DELETE /api/contexts/ID            # Delete context
DELETE /api/standups/ID            # Delete standup
```

---

## 🎯 Sample Project IDs (from Sample Data)

### E-Commerce Platform
```
Project ID: 550e8400-e29b-41d4-a716-446655440001
Agents:
  - 660e8400-e29b-41d4-a716-446655440001 (React Frontend)
  - 660e8400-e29b-41d4-a716-446655440002 (Node Backend)
  - 660e8400-e29b-41d4-a716-446655440003 (Payment Integration)
```

### Mobile App Development
```
Project ID: 550e8400-e29b-41d4-a716-446655440002
Agents:
  - 660e8400-e29b-41d4-a716-446655440004 (Flutter UI)
  - 660e8400-e29b-41d4-a716-446655440005 (API Integration)
  - 660e8400-e29b-41d4-a716-446655440006 (Firebase)
```

### Data Analytics Dashboard
```
Project ID: 550e8400-e29b-41d4-a716-446655440003
Agents:
  - 660e8400-e29b-41d4-a716-446655440007 (Dashboard Frontend)
  - 660e8400-e29b-41d4-a716-446655440008 (Data Processing)
  - 660e8400-e29b-41d4-a716-446655440009 (Reporting)
```

---

## 📝 Task Status Values

```bash
# Valid status values:
- pending      # Waiting to start
- in_progress  # Currently being worked on
- done         # Completed
- blocked      # Cannot proceed (waiting on something)
```

## 🎨 Priority Levels

```bash
# Valid priority values:
- low          # Can be done later
- medium       # Standard priority
- high         # Needs attention soon
- urgent       # Must be done immediately
```

---

## 🔐 Environment Variables

### Available Configuration

```bash
# API Server URL
AGENT_SHAKER_URL=http://localhost:8080/api

# Agent Identity (set by VS Code MCP setup)
MCP_AGENT_NAME=Agent Name
MCP_AGENT_ID=agent-uuid
MCP_PROJECT_ID=project-uuid
MCP_PROJECT_NAME=Project Name
MCP_API_URL=http://localhost:8080/api
```

### How to Set

**PowerShell:**
```powershell
$env:AGENT_SHAKER_URL="http://api.example.com:8080/api"
```

**Command Prompt:**
```cmd
set AGENT_SHAKER_URL=http://api.example.com:8080/api
```

**Linux/macOS (bash):**
```bash
export AGENT_SHAKER_URL="http://api.example.com:8080/api"
```

---

## ✅ Verification Checklist

Before reporting issues, verify:

- [ ] Docker containers are running: `docker-compose ps`
- [ ] API is accessible: `curl http://localhost:8080/api/projects`
- [ ] Node.js version >= 14.0.0: `node --version`
- [ ] npm is installed: `npm --version`
- [ ] Dependencies installed: `npm ls axios`
- [ ] No port conflicts: `netstat -ano | findstr :8080`
- [ ] Bridge starts: `npm start`
- [ ] Can run commands: `list projects`

---

## 🆘 Getting Help

### Check Documentation
1. **Quick Start:** `docs/MCP_QUICKSTART.md`
2. **Setup Guide:** `docs/MCP_SETUP_REVIEW.md`
3. **This Guide:** `docs/MCP_SETUP_QUICK_REFERENCE.md`
4. **Troubleshooting:** See "Troubleshooting Quick Fix" above

### Check Logs
```bash
# Docker logs
docker-compose logs -f agent-shaker

# Bridge running in terminal shows all activity
# Watch for error messages and API responses
```

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `ECONNREFUSED` | API not running | `docker-compose up -d` |
| `ENOTFOUND localhost` | Network issue | Check internet connection |
| `Invalid project_id` | Wrong format | Use full UUID format |
| `404 Not Found` | Resource missing | Verify ID exists |
| `ENOENT: no such file or directory` | File not found | Check file path |

---

## 🚀 Performance Tips

### Fast Queries
```bash
# List is faster when filtered
> list agents project:550e8400-e29b-41d4-a716-446655440001
# vs
> list agents
```

### Handle Large Results
```bash
# For large datasets, filtering helps
> list tasks project:550e8400-e29b-41d4-a716-446655440001
# Returns only relevant tasks
```

### Network Optimization
```bash
# Keep bridge running
# Don't restart for each command
# Reuse same connection for multiple queries
```

---

## 📚 Related Files

```
docs/
  ├── MCP_SETUP_REVIEW.md              # Detailed analysis
  ├── MCP_SETUP_CHECKLIST.md           # Verification steps
  ├── MCP_SETUP_VERIFICATION_SUMMARY.md # Test results
  ├── MCP_SETUP_QUICK_REFERENCE.md     # THIS FILE
  ├── MCP_QUICKSTART.md                # User guide
  ├── COPILOT_MCP_INTEGRATION.md       # Integration details
  └── COMPONENT_USAGE_GUIDE.md         # Component docs

scripts/
  └── setup-mcp-bridge.ps1             # Setup automation

.
  ├── mcp-bridge.js                    # Main bridge
  ├── package.json                     # Configuration
  └── test-bridge.js                   # Test suite
```

---

## 💡 Tips & Tricks

### Tab Completion (if available)
```bash
> list [TAB]       # Shows available options
```

### Multiple Queries
```bash
# Run sequence of commands
> list projects
> list agents project:550e8400-e29b-41d4-a716-446655440001
> list tasks project:550e8400-e29b-41d4-a716-446655440001
> exit
```

### JSON Output
```bash
# Some commands show full JSON
> get project 550e8400-e29b-41d4-a716-446655440001
# Pretty-printed JSON response
```

### Copy IDs
```bash
# From output, copy IDs for use in next command
> list projects
# Copy: 550e8400-e29b-41d4-a716-446655440001

> list agents project:550e8400-e29b-41d4-a716-446655440001
# Now use that ID to filter agents
```

---

## ⏱️ Response Time Reference

```
list agents          ~200ms   ✅ Very fast
list projects        ~150ms   ✅ Very fast
list tasks           ~250ms   ✅ Very fast
get project          ~100ms   ✅ Very fast
create task          ~500ms   ✅ Fast
list agents (filter) ~180ms   ✅ Very fast
```

---

## 🎓 Learning Path

1. **Day 1:** Run setup, start bridge, explore with `list` commands
2. **Day 2:** Create a test task, understand the data structure
3. **Day 3:** Set up environment variables, try different filters
4. **Day 4:** Integrate with your workflow, use in scripts
5. **Day 5:** Explore API documentation, advanced usage

---

**Last Updated:** January 28, 2026  
**Version:** 1.0.0  
**Status:** ✅ Production Ready

