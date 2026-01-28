# MCP Setup Execution Checklist

## Pre-Execution Verification

### Environment Prerequisites
- [ ] Windows PowerShell 5.0+ installed
- [ ] Node.js 14.0.0+ installed (`node --version`)
- [ ] npm 6.0.0+ installed (`npm --version`)
- [ ] Docker Desktop installed (`docker --version`)
- [ ] Docker-Compose installed (`docker-compose --version`)
- [ ] Port 8080 is available (not in use by other services)
- [ ] Port 5433 is available (PostgreSQL for database)

### Verify Agent Shaker is Running
```bash
# Check if API is accessible
curl -I http://localhost:8080/api/projects

# Should return: HTTP/1.1 200 OK
# If not, start containers:
docker-compose up -d
docker-compose logs -f agent-shaker
```

---

## Setup Execution Steps

### Step 1: Run Setup Script
```powershell
# Navigate to project directory
cd c:\Sources\GitHub\agent-shaker

# Run the setup script
./scripts/setup-mcp-bridge.ps1
```

**Expected Output:**
```
🚀 Agent Shaker MCP Bridge Setup

Checking Node.js installation...
✅ Node.js v16.13.0 found

Checking npm installation...
✅ npm 8.1.0 found

Installing dependencies...
✅ Dependencies installed successfully

Checking if Agent Shaker containers are running...
✅ MCP Server is running

Testing API connection...
✅ API is accessible

╔═══════════════════════════════════════════╗
║   ✅ Setup Complete!                      ║
╚═══════════════════════════════════════════╝

To start the MCP bridge, run:
  npm start

Or use Node directly:
  node mcp-bridge.js
```

**Troubleshooting:**
- [ ] If Node.js not found: Install from https://nodejs.org/
- [ ] If npm not found: Reinstall Node.js (includes npm)
- [ ] If dependencies fail: Run `npm cache clean --force` then retry
- [ ] If API not accessible: Verify containers with `docker-compose ps`

---

### Step 2: Start the MCP Bridge
```powershell
# Option A: Using npm script
npm start

# Option B: Using Node directly
node mcp-bridge.js

# Option C: As global command (if installed globally)
agent-shaker
```

**Expected Output:**
```
╔═══════════════════════════════════════════╗
║    Agent Shaker MCP Bridge v1.0.0         ║
╚═══════════════════════════════════════════╝

API Base: http://localhost:8080/api
Type help for available commands

agent-shaker>
```

**If Bridge Fails to Start:**
- [ ] Check if port 8080 is accessible: `curl http://localhost:8080/api/projects`
- [ ] Verify containers are running: `docker-compose ps`
- [ ] Check Node.js version: `node --version` (should be >= 14.0.0)
- [ ] Try with custom API URL: `$env:AGENT_SHAKER_URL="http://localhost:8080/api"; npm start`

---

### Step 3: Verify Bridge Functionality
```powershell
# In the bridge prompt, run these commands:

# Command 1: List all projects
> list projects

# Expected: Shows available projects with descriptions
# Example:
# Found 3 projects:
# E-Commerce Platform
#   Building a modern e-commerce solution
#   Status: active
```

- [ ] Successfully listed projects
- [ ] Output is colored and formatted nicely
- [ ] No error messages

```powershell
# Command 2: List all agents
> list agents

# Expected: Shows all agents across all projects
# Example:
# Found 9 agents:
# React Frontend Agent (frontend) - active
#   Team: UI Team
#   Last seen: 2026-01-28T08:00:00Z
```

- [ ] Successfully listed agents
- [ ] Shows agent roles and status
- [ ] Team information displayed

```powershell
# Command 3: Filter agents by project
> list agents project:550e8400-e29b-41d4-a716-446655440001

# Expected: Shows only agents in that project
```

- [ ] Successfully filtered by project
- [ ] Shows only relevant agents

```powershell
# Command 4: List tasks for a project
> list tasks project:550e8400-e29b-41d4-a716-446655440001

# Expected: Shows tasks for the project
# Example:
# Found 3 tasks:
# Implement Product Listing Page
#   Priority: high | Status: in_progress
#   Create responsive product grid with filtering
```

- [ ] Successfully listed tasks
- [ ] Shows task priority and status
- [ ] Descriptions are displayed

```powershell
# Command 5: Get specific project details
> get project 550e8400-e29b-41d4-a716-446655440001

# Expected: JSON output with full project details
```

- [ ] Successfully retrieved project details
- [ ] JSON output is formatted nicely
- [ ] All fields are present

```powershell
# Command 6: Test help command
> help

# Expected: Shows all available commands and examples
```

- [ ] Help command displays all commands
- [ ] Examples are provided
- [ ] Usage instructions are clear

---

## Integration Testing

### Test 1: API Error Handling
```powershell
# Try with invalid project ID
> list agents project:invalid-id

# Expected: Error message
# Error: API Error: 400 - Bad Request
```

- [ ] Error handling works correctly
- [ ] User receives meaningful error message

### Test 2: Command Validation
```powershell
# Try unknown command
> unknown command

# Expected: Error message
# Error: Unknown command: unknown command
# Type "help" for available commands
```

- [ ] Unknown commands are caught
- [ ] Helpful suggestion is provided

### Test 3: Exit Handling
```powershell
# Exit the bridge
> exit

# Expected: Goodbye message and prompt closes
# Goodbye!
```

- [ ] Bridge closes gracefully
- [ ] Proper cleanup occurs

---

## Environment Variable Testing

### Test Custom API URL
```powershell
# Set custom API URL (if running on different host)
$env:AGENT_SHAKER_URL="http://api.example.com:8080/api"

# Start bridge
npm start

# Verify it's using custom URL
# API Base: http://api.example.com:8080/api
```

- [ ] Custom API URL is respected
- [ ] Bridge connects to custom host

### Verify Default Fallback
```powershell
# Clear the environment variable
$env:AGENT_SHAKER_URL=""

# Start bridge
npm start

# Should use default
# API Base: http://localhost:8080/api
```

- [ ] Default URL is used when env var not set

---

## Configuration Verification

### Check Package Configuration
```powershell
# View package.json
cat package.json | more

# Verify these fields:
# - "name": "agent-shaker-mcp-bridge"
# - "main": "mcp-bridge.js"
# - "bin": { "agent-shaker": "./mcp-bridge.js" }
# - "scripts": { "start": "node mcp-bridge.js" }
```

- [ ] Package name is correct
- [ ] Main entry point is mcp-bridge.js
- [ ] Scripts are properly configured
- [ ] Dependencies include axios

### Check Setup Script
```powershell
# View setup script
Get-Content ./scripts/setup-mcp-bridge.ps1 | more

# Verify it checks:
# - Node.js installation
# - npm installation
# - Dependencies installation
# - Container status
# - API connectivity
```

- [ ] All validation checks are present
- [ ] Error messages are helpful
- [ ] Success criteria are clear

---

## Performance Verification

### Response Time Testing
```powershell
# Time a list projects command
# (Runs automatically - just observe)
> list projects

# Should complete in < 1 second
# Check that response time is reasonable
```

- [ ] Response time is acceptable (< 1s)
- [ ] No timeouts occur
- [ ] No memory leaks (bridge still responsive after multiple commands)

### Concurrent Request Testing
```powershell
# Run multiple commands in sequence
> list projects
> list agents
> list agents project:550e8400-e29b-41d4-a716-446655440001
> list projects
> list agents

# Bridge should handle all smoothly
```

- [ ] Multiple commands execute without issues
- [ ] No connection pooling problems
- [ ] No memory growth over time

---

## Documentation Verification

### Check Setup Guide Accessibility
- [ ] MCP_QUICKSTART.md exists and is clear
- [ ] Setup instructions are step-by-step
- [ ] Examples are provided
- [ ] Troubleshooting section exists

### Check API Documentation
- [ ] API endpoints are documented
- [ ] curl examples are provided
- [ ] Request/response formats are shown
- [ ] Error codes are explained

---

## Final Verification Checklist

### ✅ Setup Complete
- [ ] Setup script ran successfully
- [ ] All validation checks passed
- [ ] No errors in npm install

### ✅ Bridge Running
- [ ] Bridge starts without errors
- [ ] API Base URL is correct
- [ ] Prompt appears and is ready for input

### ✅ Core Functions
- [ ] `list projects` works
- [ ] `list agents` works
- [ ] `list tasks` works
- [ ] `get project` works
- [ ] `help` command works
- [ ] `exit` command works

### ✅ Error Handling
- [ ] Invalid inputs are handled gracefully
- [ ] API errors are reported clearly
- [ ] Network errors are caught

### ✅ Environment
- [ ] Default API URL works
- [ ] Custom API URL can be set
- [ ] Configuration is respected

### ✅ Documentation
- [ ] Setup guide is complete
- [ ] Commands are documented
- [ ] Examples are provided
- [ ] Troubleshooting is available

---

## Sign-Off

**Verification Date:** ___________

**Verified By:** ___________

**System:** Windows / Linux / macOS (circle one)

**Node.js Version:** ___________

**npm Version:** ___________

**Notes:**
```
_________________________________
_________________________________
_________________________________
```

**Overall Status:** 
- [ ] ✅ ALL TESTS PASSED - MCP Setup is fully operational
- [ ] ⚠️  TESTS PASSED WITH NOTES - See notes above
- [ ] ❌ TESTS FAILED - Document issues above

