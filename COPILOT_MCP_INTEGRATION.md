# 🔌 Connecting GitHub Copilot to Your MCP Server

## Overview

Your current server is a **REST API** for managing agents, tasks, and projects. To connect GitHub Copilot to it, you have two options:

1. **Use it as a REST API context source** (current state)
2. **Implement true MCP (Model Context Protocol) support** (requires additional implementation)

---

## Option 1: Using Your Server with Copilot (Current State)

### What You Have Now

Your server provides REST APIs that can supply context to development tools:

- **Projects API**: `/api/projects`
- **Agents API**: `/api/agents`
- **Tasks API**: `/api/tasks`
- **Contexts API**: `/api/contexts`
- **Documentation API**: `/api/docs`
- **WebSocket**: `/ws` for real-time updates

### How to Use with VS Code Copilot

#### 1. **VS Code Settings Configuration**

Create or update your workspace settings (`.vscode/settings.json`):

```json
{
  "github.copilot.advanced": {
    "contextProviders": {
      "agent-shaker": {
        "url": "http://localhost:8080/api",
        "headers": {
          "Content-Type": "application/json"
        }
      }
    }
  },
  "github.copilot.enable": {
    "*": true
  }
}
```

#### 2. **Use Copilot Chat with @workspace**

In VS Code, you can query your server's context:

```
@workspace What agents are currently active in the E-Commerce project?
```

Copilot will use your workspace files as context, which includes your API endpoints.

#### 3. **Create a Copilot Extension (VS Code Extension)**

Create a custom VS Code extension that acts as a bridge:

**File: `copilot-mcp-bridge/extension.js`**

```javascript
const vscode = require('vscode');
const axios = require('axios');

function activate(context) {
    // Register a command to fetch agent context
    let disposable = vscode.commands.registerCommand(
        'agent-shaker.getAgentContext',
        async () => {
            try {
                const response = await axios.get('http://localhost:8080/api/agents');
                const agents = response.data;
                
                // Format context for Copilot
                const contextText = agents.map(agent => 
                    `Agent: ${agent.name} (${agent.role}) - ${agent.status}`
                ).join('\n');
                
                // Insert into editor
                const editor = vscode.window.activeTextEditor;
                if (editor) {
                    editor.edit(editBuilder => {
                        editBuilder.insert(
                            editor.selection.active,
                            `\n// Current Agents:\n// ${contextText}\n`
                        );
                    });
                }
            } catch (error) {
                vscode.window.showErrorMessage('Failed to fetch agents: ' + error.message);
            }
        }
    );

    context.subscriptions.push(disposable);
}

exports.activate = activate;
```

---

## Option 2: Implementing True MCP Support

### What is Model Context Protocol (MCP)?

MCP is an open protocol by Anthropic that allows AI assistants (like Claude, Copilot) to securely connect to external data sources and tools.

### Architecture Changes Needed

#### 1. **MCP Server Implementation**

You need to implement the MCP protocol specification. Here's a starter:

**File: `internal/mcp/server.go`**

```go
package mcp

import (
    "encoding/json"
    "net/http"
)

type MCPServer struct {
    name        string
    version     string
    description string
}

type MCPResponse struct {
    Protocol string                 `json:"protocol"`
    Version  string                 `json:"version"`
    Capabilities map[string]bool    `json:"capabilities"`
    Tools    []Tool                 `json:"tools,omitempty"`
    Resources []Resource            `json:"resources,omitempty"`
}

type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"inputSchema"`
}

type Resource struct {
    URI         string `json:"uri"`
    Name        string `json:"name"`
    Description string `json:"description"`
    MimeType    string `json:"mimeType"`
}

func NewMCPServer() *MCPServer {
    return &MCPServer{
        name:        "agent-shaker",
        version:     "1.0.0",
        description: "Agent and task management MCP server",
    }
}

func (s *MCPServer) Initialize(w http.ResponseWriter, r *http.Request) {
    response := MCPResponse{
        Protocol: "mcp",
        Version:  "1.0",
        Capabilities: map[string]bool{
            "tools":     true,
            "resources": true,
        },
        Tools: []Tool{
            {
                Name:        "list_agents",
                Description: "List all agents or agents for a specific project",
                InputSchema: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "project_id": map[string]string{
                            "type":        "string",
                            "description": "Optional project ID to filter agents",
                        },
                    },
                },
            },
            {
                Name:        "create_task",
                Description: "Create a new task for an agent",
                InputSchema: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "title": map[string]string{
                            "type":        "string",
                            "description": "Task title",
                        },
                        "description": map[string]string{
                            "type":        "string",
                            "description": "Task description",
                        },
                        "project_id": map[string]string{
                            "type":        "string",
                            "description": "Project ID",
                        },
                    },
                    "required": []string{"title", "project_id"},
                },
            },
        },
        Resources: []Resource{
            {
                URI:         "agent-shaker://agents",
                Name:        "agents",
                Description: "List of all agents",
                MimeType:    "application/json",
            },
            {
                URI:         "agent-shaker://projects",
                Name:        "projects",
                Description: "List of all projects",
                MimeType:    "application/json",
            },
        },
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *MCPServer) CallTool(w http.ResponseWriter, r *http.Request) {
    var request struct {
        Name      string                 `json:"name"`
        Arguments map[string]interface{} `json:"arguments"`
    }

    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Handle tool calls
    // This would call your existing handlers
    // e.g., agentHandler.ListAgents(), taskHandler.CreateTask(), etc.
}
```

#### 2. **Add MCP Routes to Main**

**Update: `cmd/server/main.go`**

```go
import (
    "github.com/techbuzzz/agent-shaker/internal/mcp"
)

func main() {
    // ... existing code ...

    // MCP Protocol endpoints
    mcpServer := mcp.NewMCPServer()
    
    mcp := r.PathPrefix("/mcp").Subrouter()
    mcp.HandleFunc("/initialize", mcpServer.Initialize).Methods("POST")
    mcp.HandleFunc("/tools/call", mcpServer.CallTool).Methods("POST")
    mcp.HandleFunc("/resources/read", mcpServer.ReadResource).Methods("POST")

    // ... rest of code ...
}
```

#### 3. **MCP Configuration File**

Create a configuration file for MCP clients:

**File: `mcp-config.json`**

```json
{
  "mcpServers": {
    "agent-shaker": {
      "command": "docker",
      "args": [
        "exec",
        "-i",
        "agent-shaker-mcp-server-1",
        "/app/mcp-server"
      ],
      "env": {
        "DATABASE_URL": "postgres://mcp:secret@postgres:5432/mcp_tracker?sslmode=disable"
      }
    }
  }
}
```

Or for HTTP-based MCP:

```json
{
  "mcpServers": {
    "agent-shaker": {
      "url": "http://localhost:8080/mcp",
      "transport": "http",
      "headers": {
        "Content-Type": "application/json"
      }
    }
  }
}
```

---

## Quick Setup Guide for VS Code Copilot

### Step 1: Install Copilot MCP Extension

```bash
# Install the official MCP extension
code --install-extension anthropic.mcp-vscode
```

### Step 2: Configure MCP Settings

Add to your VS Code `settings.json`:

```json
{
  "mcp.servers": {
    "agent-shaker": {
      "url": "http://localhost:8080/mcp",
      "transport": "http"
    }
  }
}
```

### Step 3: Use in Copilot Chat

Once configured, you can use commands like:

```
@agent-shaker List all active agents in the E-Commerce project
```

```
@agent-shaker Create a task "Fix payment bug" for the Payment Integration Agent
```

---

## Current Workaround: API Wrapper Script

Since true MCP implementation requires code changes, here's a Node.js script you can use as a bridge:

**File: `mcp-bridge.js`**

```javascript
#!/usr/bin/env node

const axios = require('axios');
const readline = require('readline');

const API_BASE = 'http://localhost:8080/api';

const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
});

async function executeCommand(command) {
    try {
        if (command.startsWith('list agents')) {
            const projectId = command.match(/project:(\S+)/)?.[1];
            const url = projectId 
                ? `${API_BASE}/agents?project_id=${projectId}`
                : `${API_BASE}/agents`;
            
            const response = await axios.get(url);
            console.log(JSON.stringify(response.data, null, 2));
        }
        else if (command.startsWith('list projects')) {
            const response = await axios.get(`${API_BASE}/projects`);
            console.log(JSON.stringify(response.data, null, 2));
        }
        else if (command.startsWith('list tasks')) {
            const projectId = command.match(/project:(\S+)/)?.[1];
            if (!projectId) {
                console.error('Error: project_id required for tasks');
                return;
            }
            const response = await axios.get(`${API_BASE}/tasks?project_id=${projectId}`);
            console.log(JSON.stringify(response.data, null, 2));
        }
        else {
            console.log('Unknown command. Available: list agents, list projects, list tasks');
        }
    } catch (error) {
        console.error('Error:', error.message);
    }
}

console.log('Agent Shaker MCP Bridge');
console.log('Commands: list agents [project:id], list projects, list tasks project:id');
console.log('');

rl.on('line', async (line) => {
    await executeCommand(line.trim());
    rl.prompt();
});

rl.prompt();
```

**Usage:**

```bash
npm install axios
node mcp-bridge.js

# Then type commands:
> list agents
> list projects
> list tasks project:550e8400-e29b-41d4-a716-446655440001
```

---

## Testing MCP Integration

### Test with curl

```bash
# Test if your server is accessible
curl http://localhost:8080/api/agents

# Test with project filter
curl http://localhost:8080/api/agents?project_id=550e8400-e29b-41d4-a716-446655440001

# Test WebSocket connection
wscat -c ws://localhost:8080/ws
```

### Test with PowerShell

```powershell
# Get all agents
Invoke-RestMethod -Uri http://localhost:8080/api/agents -Method Get

# Get agents for specific project
$projectId = "550e8400-e29b-41d4-a716-446655440001"
Invoke-RestMethod -Uri "http://localhost:8080/api/agents?project_id=$projectId" -Method Get

# Create a new task
$body = @{
    project_id = $projectId
    title = "Test task from Copilot"
    description = "Testing MCP integration"
    priority = "high"
} | ConvertTo-Json

Invoke-RestMethod -Uri http://localhost:8080/api/tasks -Method Post -Body $body -ContentType "application/json"
```

---

## Recommendations

### Short Term (Use Now)

1. **Use REST API directly** from your IDE
2. **Create VS Code snippets** that call your API
3. **Write a simple CLI tool** (like the mcp-bridge.js above)

### Medium Term (1-2 weeks)

1. **Implement MCP protocol endpoints** in your Go server
2. **Add MCP discovery endpoint** (`/mcp/initialize`)
3. **Create tool definitions** for all your APIs

### Long Term (1+ months)

1. **Build a proper VS Code extension**
2. **Add streaming support** for real-time updates
3. **Implement MCP's Sampling API** for agentic workflows
4. **Add authentication/authorization** for secure access

---

## MCP Protocol Specification Reference

- **Official Spec**: https://spec.modelcontextprotocol.io/
- **GitHub**: https://github.com/modelcontextprotocol
- **Examples**: https://github.com/modelcontextprotocol/servers

### Key Endpoints for MCP

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/mcp/initialize` | POST | Protocol handshake |
| `/mcp/tools/call` | POST | Execute a tool |
| `/mcp/resources/read` | POST | Read a resource |
| `/mcp/resources/list` | POST | List available resources |
| `/mcp/prompts/list` | POST | List available prompts |

---

## Summary

**Current State**: Your server is a REST API that can be queried programmatically.

**To Connect Copilot**:
1. ✅ Use REST API directly (works now)
2. ⚠️ Implement MCP protocol (requires code changes)
3. ✅ Create a bridge script (workaround)

**Next Steps**:
1. Try the `mcp-bridge.js` script
2. Test API endpoints with PowerShell
3. Consider implementing true MCP support if needed

Would you like me to:
- Create the full MCP implementation in Go?
- Build a VS Code extension for your server?
- Set up the bridge script with more features?
