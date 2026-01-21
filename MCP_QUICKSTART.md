# 🎯 Quick Start: Connecting Copilot to Agent Shaker

## TL;DR - 3 Ways to Connect

### 1. ✅ **MCP Bridge Script** (Recommended - Ready Now!)

```powershell
# Setup (one time)
./setup-mcp-bridge.ps1

# Run the bridge
npm start

# Use commands
> list agents
> list projects
> create task
```

### 2. ⚡ **Direct API Calls** (Use in Code)

```javascript
// In your code
const axios = require('axios');
const agents = await axios.get('http://localhost:8080/api/agents');
```

### 3. 🔧 **True MCP Implementation** (Future)

Requires implementing the MCP protocol specification in Go. See `COPILOT_MCP_INTEGRATION.md` for details.

---

## Quick Setup (5 Minutes)

### Step 1: Ensure Containers Are Running

```powershell
docker-compose ps

# If not running:
docker-compose up -d
```

### Step 2: Install MCP Bridge

```powershell
# Run setup script
./setup-mcp-bridge.ps1

# This will:
# - Check Node.js installation
# - Install dependencies (axios)
# - Test API connection
# - Start the bridge
```

### Step 3: Use the Bridge

```powershell
npm start
```

Then type commands:
```
agent-shaker> list agents
agent-shaker> list projects
agent-shaker> list tasks project:550e8400-e29b-41d4-a716-446655440001
agent-shaker> create task
agent-shaker> help
```

---

## Available Commands

| Command | Description | Example |
|---------|-------------|---------|
| `list agents` | List all agents | `list agents` |
| `list agents project:ID` | List agents for a project | `list agents project:550e8400...` |
| `list projects` | List all projects | `list projects` |
| `list tasks project:ID` | List tasks for a project | `list tasks project:550e8400...` |
| `get project ID` | Get project details | `get project 550e8400...` |
| `create task` | Create a new task (interactive) | `create task` |
| `help` | Show help | `help` |
| `exit` | Exit bridge | `exit` |

---

## Using with GitHub Copilot Chat

### Method 1: Copy/Paste Context

1. Run the bridge: `npm start`
2. Get data: `list agents`
3. Copy the output
4. In VS Code Copilot Chat:
   ```
   Given these agents:
   [paste output]
   
   Create a task to fix the payment integration bug
   ```

### Method 2: VS Code Task Integration

Create `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Get Agents Context",
      "type": "shell",
      "command": "Invoke-WebRequest -Uri http://localhost:8080/api/agents | ConvertFrom-Json | Format-List",
      "problemMatcher": []
    }
  ]
}
```

Then run: `Terminal → Run Task → Get Agents Context`

### Method 3: VS Code Snippet

Create `.vscode/snippets.code-snippets`:

```json
{
  "Get Agents": {
    "prefix": "agents",
    "body": [
      "// Agents context from API",
      "// Run: Invoke-WebRequest http://localhost:8080/api/agents",
      "$0"
    ],
    "description": "Insert agents context"
  }
}
```

---

## API Endpoints Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/agents` | GET | List all agents |
| `/api/agents?project_id={id}` | GET | List project agents |
| `/api/projects` | GET | List all projects |
| `/api/tasks?project_id={id}` | GET | List project tasks |
| `/api/contexts?project_id={id}` | GET | List project contexts |
| `/api/docs` | GET | List documentation |
| `/ws` | WebSocket | Real-time updates |

---

## Testing

### Test API with PowerShell

```powershell
# Get all agents
Invoke-RestMethod -Uri http://localhost:8080/api/agents

# Get agents as formatted table
Invoke-RestMethod -Uri http://localhost:8080/api/agents | 
    Format-Table name, role, team, status

# Get projects
Invoke-RestMethod -Uri http://localhost:8080/api/projects | 
    Format-Table name, status, description

# Get specific project tasks
$projectId = "550e8400-e29b-41d4-a716-446655440001"
Invoke-RestMethod -Uri "http://localhost:8080/api/tasks?project_id=$projectId" | 
    Format-Table title, status, priority
```

### Test with curl (Git Bash)

```bash
# List agents
curl http://localhost:8080/api/agents | jq

# List projects
curl http://localhost:8080/api/projects | jq

# List tasks
curl "http://localhost:8080/api/tasks?project_id=550e8400-e29b-41d4-a716-446655440001" | jq
```

---

## Example Workflows

### Workflow 1: Get Agent Status for Code Comment

```powershell
# 1. Get agents
npm start
> list agents

# 2. Copy output
# 3. In your code:
/**
 * Current Active Agents:
 * - React Frontend Agent (UI Team) - active
 * - Node Backend Agent (API Team) - active
 * - Payment Integration Agent (Integration Team) - active
 */
```

### Workflow 2: Create Task from Copilot Suggestion

```
In Copilot Chat:
"@workspace Create a task to implement OAuth2 authentication for the API"

Copilot suggests:
- Title: "Implement OAuth2 Authentication"
- Description: "Add OAuth2 authentication middleware..."
- Priority: "high"

Then run:
> create task
Title: Implement OAuth2 Authentication
Description: Add OAuth2 authentication middleware...
Project ID: 550e8400-e29b-41d4-a716-446655440001
Priority: high
```

### Workflow 3: Context-Aware Development

```powershell
# Get current project state
Invoke-RestMethod http://localhost:8080/api/projects | Select-Object -First 1 | Format-List

# Get agents for that project
$projectId = "550e8400-e29b-41d4-a716-446655440001"
Invoke-RestMethod "http://localhost:8080/api/agents?project_id=$projectId"

# Get tasks
Invoke-RestMethod "http://localhost:8080/api/tasks?project_id=$projectId"

# Now ask Copilot:
"Given this project state, what should be the next task priority?"
```

---

## Troubleshooting

### Bridge Won't Start

```powershell
# Check Node.js
node --version  # Should be >= 14

# Install dependencies
npm install

# Check containers
docker-compose ps

# Check API
Invoke-WebRequest http://localhost:8080/api/projects
```

### API Returns HTML

This was fixed! If you still see HTML:

```powershell
# Rebuild web container
docker-compose up -d --build web

# Verify
Invoke-WebRequest http://localhost/api/agents | Select-Object Content
```

### Connection Refused

```powershell
# Start containers
docker-compose up -d

# Wait for health check
Start-Sleep -Seconds 10

# Check logs
docker-compose logs mcp-server
```

---

## Next Steps

1. ✅ **Try the MCP bridge** - Run `npm start`
2. 📖 **Read full guide** - See `COPILOT_MCP_INTEGRATION.md`
3. 🔧 **Customize commands** - Edit `mcp-bridge.js`
4. 🚀 **Build VS Code extension** - For deeper integration

---

## Resources

- **Full Guide**: `COPILOT_MCP_INTEGRATION.md`
- **API Docs**: `docs/API.md`
- **Bridge Script**: `mcp-bridge.js`
- **Setup Script**: `setup-mcp-bridge.ps1`

---

**Status**: ✅ Ready to use!  
**Time to Setup**: ~5 minutes  
**Dependencies**: Node.js, Docker (containers running)
