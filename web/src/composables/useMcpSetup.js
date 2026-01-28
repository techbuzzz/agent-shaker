import { computed, isRef, unref } from 'vue'

/**
 * Composable for MCP setup configuration generation
 * Handles both ref and direct values for agent, project, and apiUrl
 * @param {Object|Ref} agent - Agent object or ref
 * @param {Object|Ref} project - Project object or ref
 * @param {string|Ref} apiUrl - API base URL or ref
 * @returns {Object} MCP configuration objects and utilities
 */
export const useMcpSetup = (agent, project, apiUrl) => {
  // Helper to handle both refs and direct values
  const getAgentValue = computed(() => unref(agent))
  const getProjectValue = computed(() => unref(project))
  const getApiUrl = computed(() => unref(apiUrl))
  const mcpSettingsJson = computed(() => {
    const agentValue = getAgentValue.value
    const projectValue = getProjectValue.value
    const urlValue = getApiUrl.value

    if (!agentValue || !projectValue) return ''

    return JSON.stringify({
      "terminal.integrated.env.windows": {
        "MCP_AGENT_NAME": agentValue.name,
        "MCP_AGENT_ID": agentValue.id,
        "MCP_PROJECT_ID": projectValue.id,
        "MCP_PROJECT_NAME": projectValue.name,
        "MCP_API_URL": urlValue
      },
      "terminal.integrated.env.linux": {
        "MCP_AGENT_NAME": agentValue.name,
        "MCP_AGENT_ID": agentValue.id,
        "MCP_PROJECT_ID": projectValue.id,
        "MCP_PROJECT_NAME": projectValue.name,
        "MCP_API_URL": urlValue
      },
      "terminal.integrated.env.osx": {
        "MCP_AGENT_NAME": agentValue.name,
        "MCP_AGENT_ID": agentValue.id,
        "MCP_PROJECT_ID": projectValue.id,
        "MCP_PROJECT_NAME": projectValue.name,
        "MCP_API_URL": urlValue
      }
    }, null, 2)
  })

  const mcpCopilotInstructions = computed(() => {
    const agentValue = getAgentValue.value
    const projectValue = getProjectValue.value
    const urlValue = getApiUrl.value

    if (!agentValue || !projectValue) return ''

    return `# Agent Identity and MCP Integration

## Your Identity
- **Agent Name**: ${agentValue.name}
- **Agent ID**: ${agentValue.id}
- **Role**: ${agentValue.role}
- **Team**: ${agentValue.team || 'Not specified'}
- **Project**: ${projectValue.name}
- **Project ID**: ${projectValue.id}

## MCP API Configuration
- **API URL**: ${urlValue}

## Your Responsibilities
As the **${agentValue.role}** agent, you should:
${agentValue.role === 'frontend' ? `
- Focus on UI/UX implementation
- Work with Vue.js, React, or other frontend frameworks
- Implement responsive designs and accessibility
- Handle client-side state management
` : `
- Focus on API development and backend logic
- Work with databases and data models
- Implement business logic and validations
- Handle server-side security and authentication
`}

## Collaboration Guidelines
1. Always check for existing tasks before starting new work
2. Update task status when you begin and complete work
3. Document important decisions and implementation details
4. Check other agents' contexts to avoid conflicts
`
  })

  const mcpPowerShellScript = computed(() => {
    const agentValue = getAgentValue.value
    const projectValue = getProjectValue.value
    const urlValue = getApiUrl.value

    if (!agentValue || !projectValue) return ''

    return `# MCP Agent Helper Script for PowerShell
# Agent: ${agentValue.name}
# Project: ${projectValue.name}

$MCP_API_URL = "${urlValue}"
$MCP_AGENT_ID = "${agentValue.id}"
$MCP_PROJECT_ID = "${projectValue.id}"

function Get-MyTasks {
    Invoke-RestMethod -Uri "$MCP_API_URL/agents/$MCP_AGENT_ID/tasks" -Method GET
}

function Update-TaskStatus {
    param(
        [Parameter(Mandatory=$true)]
        [string]$TaskId,
        [Parameter(Mandatory=$true)]
        [ValidateSet("pending", "in_progress", "done", "blocked")]
        [string]$Status
    )
    
    $body = @{ status = $Status } | ConvertTo-Json
    Invoke-RestMethod -Uri "$MCP_API_URL/tasks/$TaskId/status" -Method PUT -Body $body -ContentType "application/json"
}

function Add-Context {
    param(
        [Parameter(Mandatory=$true)]
        [string]$Title,
        [Parameter(Mandatory=$true)]
        [string]$Content,
        [string[]]$Tags = @()
    )
    
    $body = @{
        project_id = $MCP_PROJECT_ID
        agent_id = $MCP_AGENT_ID
        title = $Title
        content = $Content
        tags = $Tags
    } | ConvertTo-Json
    
    Invoke-RestMethod -Uri "$MCP_API_URL/contexts" -Method POST -Body $body -ContentType "application/json"
}

function Get-ProjectContexts {
    Invoke-RestMethod -Uri "$MCP_API_URL/projects/$MCP_PROJECT_ID/contexts" -Method GET
}

# Usage examples:
# Get-MyTasks
# Update-TaskStatus -TaskId "task-uuid" -Status "in_progress"
# Add-Context -Title "API Design" -Content "Documentation content..." -Tags @("api", "design")
`
  })

  const mcpBashScript = computed(() => {
    const agentValue = getAgentValue.value
    const projectValue = getProjectValue.value
    const urlValue = getApiUrl.value

    if (!agentValue || !projectValue) return ''

    return `#!/bin/bash
# MCP Agent Helper Script for Bash
# Agent: ${agentValue.name}
# Project: ${projectValue.name}

MCP_API_URL="${urlValue}"
MCP_AGENT_ID="${agentValue.id}"
MCP_PROJECT_ID="${projectValue.id}"

# Get tasks assigned to this agent
get_my_tasks() {
    curl -s "$MCP_API_URL/agents/$MCP_AGENT_ID/tasks" | jq .
}

# Update task status
# Usage: update_task_status <task_id> <status>
# Status: pending, in_progress, done, blocked
update_task_status() {
    local task_id=$1
    local status=$2
    curl -s -X PUT "$MCP_API_URL/tasks/$task_id/status" \\
        -H "Content-Type: application/json" \\
        -d "{\\"status\\": \\"$status\\"}" | jq .
}

# Add context/documentation
# Usage: add_context "Title" "Content" "tag1,tag2"
add_context() {
    local title=$1
    local content=$2
    local tags=$3
    
    curl -s -X POST "$MCP_API_URL/contexts" \\
        -H "Content-Type: application/json" \\
        -d "{
            \\"project_id\\": \\"$MCP_PROJECT_ID\\",
            \\"agent_id\\": \\"$MCP_AGENT_ID\\",
            \\"title\\": \\"$title\\",
            \\"content\\": \\"$content\\",
            \\"tags\\": [\\"$tags\\"]
        }" | jq .
}

# Get project contexts
get_project_contexts() {
    curl -s "$MCP_API_URL/projects/$MCP_PROJECT_ID/contexts" | jq .
}

# Usage examples:
# get_my_tasks
# update_task_status "task-uuid" "in_progress"
# add_context "API Design" "Documentation content..." "api,design"
`
  })

  const mcpVSCodeJson = computed(() => {
    const agentValue = getAgentValue.value
    const projectValue = getProjectValue.value
    const urlValue = getApiUrl.value

    if (!agentValue || !projectValue) return ''

    // Build MCP URL with project and agent context
    const baseUrl = urlValue.replace('/api', '')
    const mcpUrl = `${baseUrl}?project_id=${projectValue.id}&agent_id=${agentValue.id}`

    const config = {
      "servers": {
        "agent-shaker": {
          "type": "http",
          "url": mcpUrl 
        }
      }
    }

    return JSON.stringify(config, null, 2)
  })

  const mcpVS2026Json = computed(() => {
    const agentValue = getAgentValue.value
    const projectValue = getProjectValue.value
    const urlValue = getApiUrl.value

    if (!agentValue || !projectValue) return ''

    // Build MCP URL with project and agent context
    const baseUrl = urlValue.replace('/api', '')
    const mcpUrl = `${baseUrl}?project_id=${projectValue.id}&agent_id=${agentValue.id}`

    const config = {
      "$schema": "https://aka.ms/mcp-server-schema",
      "description": "Agent Shaker MCP Server Configuration for Visual Studio 2026",
      "version": "1.0.0",
      "language": "en-US",
      "servers": {
        "agent-shaker": {
          "type": "http",
          "url": mcpUrl,
          "name": "Agent Shaker MCP Server",
          "description": "Multi-agent coordination platform for collaborative development",
          "capabilities": [
            "resources",
            "tools",
            "prompts",
            "context-sharing"
          ],
          "authentication": {
            "type": "none"
          },
          "project": {
            "id": projectValue.id,
            "name": projectValue.name,
            "description": projectValue.description || "",
            "status": projectValue.status,
            "type": "multi-agent",
            "environment": "visual-studio-2026"
          },
          "agent": {
            "id": agentValue.id,
            "name": agentValue.name,
            "role": agentValue.role,
            "team": agentValue.team || "default",
            "status": agentValue.status,
            "type": "ai-developer",
            "capabilities": [
              agentValue.role === 'frontend' ? 'ui-development' : 'backend-development',
              agentValue.role === 'frontend' ? 'component-design' : 'api-development',
              "task-management",
              "context-sharing",
              "documentation"
            ]
          },
          "resources": {
            "baseUrl": urlValue,
            "endpoints": {
              "health": "/health",
              "projects": "/projects",
              "agents": "/agents",
              "tasks": "/tasks",
              "contexts": "/contexts",
              "documentation": "/documentation",
              "dashboard": "/dashboard",
              "myTasks": `/agents/${agentValue.id}/tasks`,
              "myAgent": `/agents/${agentValue.id}`,
              "projectAgents": `/projects/${projectValue.id}/agents`,
              "projectTasks": `/projects/${projectValue.id}/tasks`,
              "projectContexts": `/projects/${projectValue.id}/contexts`
            }
          },
          "tools": [
            {
              "name": "get_my_identity",
              "description": "Get your agent identity and assigned project from MCP connection",
              "category": "identity"
            },
            {
              "name": "get_my_project",
              "description": "Get details of your assigned project",
              "category": "project"
            },
            {
              "name": "get_my_tasks",
              "description": "Get tasks assigned to this agent",
              "category": "task-management",
              "endpoint": `/agents/${agentValue.id}/tasks`,
              "method": "GET"
            },
            {
              "name": "claim_task",
              "description": "Claim a task and set it to in_progress",
              "category": "task-management"
            },
            {
              "name": "complete_task",
              "description": "Mark a task as done",
              "category": "task-management"
            },
            {
              "name": "update_task_status",
              "description": "Update the status of a task",
              "category": "task-management",
              "endpoint": "/tasks/{task_id}/status",
              "method": "PUT"
            },
            {
              "name": "create_task",
              "description": "Create a new task in the project",
              "category": "task-management",
              "endpoint": "/tasks",
              "method": "POST"
            },
            {
              "name": "get_project_contexts",
              "description": "Get all contexts/documentation for the project",
              "category": "documentation",
              "endpoint": `/projects/${projectValue.id}/contexts`,
              "method": "GET"
            },
            {
              "name": "add_context",
              "description": "Add context or documentation to the project",
              "category": "documentation",
              "endpoint": "/contexts",
              "method": "POST"
            },
            {
              "name": "get_project_agents",
              "description": "Get all agents working on the project",
              "category": "collaboration",
              "endpoint": `/projects/${projectValue.id}/agents`,
              "method": "GET"
            },
            {
              "name": "get_dashboard_stats",
              "description": "Get project statistics and overview",
              "category": "monitoring",
              "endpoint": "/dashboard",
              "method": "GET"
            }
          ],
          "behavior": {
            "autoReconnect": true,
            "maxRetries": 3,
            "timeout": 30000,
            "healthCheckInterval": 60000
          },
          "security": {
            "cors": {
              "enabled": true,
              "allowOrigins": ["http://localhost:5173", "http://localhost:3000", "http://localhost:8080"]
            },
            "rateLimiting": {
              "enabled": false,
              "requestsPerMinute": 60
            }
          },
          "logging": {
            "level": "info",
            "format": "json",
            "includeTimestamp": true
          }
        }
      }
    }

    return JSON.stringify(config, null, 2)
  })

  // Bundle all MCP configs together
  const mcpConfig = computed(() => ({
    mcpSettingsJson: mcpSettingsJson.value,
    mcpCopilotInstructions: mcpCopilotInstructions.value,
    mcpPowerShellScript: mcpPowerShellScript.value,
    mcpBashScript: mcpBashScript.value,
    mcpVSCodeJson: mcpVSCodeJson.value,
    mcpVS2026Json: mcpVS2026Json.value
  }))

  return {
    mcpSettingsJson,
    mcpCopilotInstructions,
    mcpPowerShellScript,
    mcpBashScript,
    mcpVSCodeJson,
    mcpVS2026Json,
    mcpConfig,
    downloadFile,
    downloadAllMcpFiles,
    copyMcpFilesToProject
  }
}

/**
 * Download file helper
 * @param {string} filename - Name of file to download
 * @param {string} content - File content
 * @param {string} mimeType - MIME type
 */
export const downloadFile = (filename, content, mimeType = 'text/plain') => {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Download all MCP files as zip
 * @param {Object} mcpConfig - MCP configuration object
 * @param {string} agentName - Agent name for filename
 */
export const downloadAllMcpFiles = async (mcpConfig, agentName) => {
  const { default: JSZip } = await import('jszip')
  const zip = new JSZip()
  
  zip.file('.vscode/settings.json', mcpConfig.mcpSettingsJson)
  zip.file('.vscode/mcp.json', mcpConfig.mcpVSCodeJson)
  zip.file('.mcp.json', mcpConfig.mcpVS2026Json)
  zip.file('.github/copilot-instructions.md', mcpConfig.mcpCopilotInstructions)
  zip.file('scripts/mcp-agent.ps1', mcpConfig.mcpPowerShellScript)
  zip.file('scripts/mcp-agent.sh', mcpConfig.mcpBashScript)
  
  const readmeContent = `# MCP Setup Files for ${agentName}

## Contents
- .vscode/settings.json - VS Code environment variables
- .vscode/mcp.json - VS Code MCP server configuration
- .mcp.json - Visual Studio 2026 MCP server configuration (root directory)
- .github/copilot-instructions.md - GitHub Copilot agent instructions
- scripts/mcp-agent.ps1 - PowerShell helper script
- scripts/mcp-agent.sh - Bash helper script

## Setup Instructions

### For Visual Studio 2026:
1. Extract this zip to your project's root directory
2. The `.mcp.json` file will be automatically recognized
3. Restart Visual Studio 2026 to apply MCP configuration
4. Start using Copilot with your agent identity!

### For VS Code:
1. Extract to project root
2. The `.vscode/settings.json` and `.vscode/mcp.json` will be applied
3. Restart VS Code
4. Terminal environment variables will be automatically set

### For Command Line:
1. Windows: Run \`scripts/mcp-agent.ps1\`
2. macOS/Linux: Run \`scripts/mcp-agent.sh\`
`
  zip.file('MCP_SETUP_README.md', readmeContent)
  
  const content = await zip.generateAsync({ type: 'blob' })
  const agentSlug = agentName.toLowerCase().replace(/[^a-z0-9]+/g, '-')
  downloadFile(`mcp-setup-${agentSlug}.zip`, content, 'application/zip')
}

/**
 * Copy MCP files directly to project directory via API
 * @param {Object} mcpConfig - MCP configuration object
 * @param {string} projectId - Project ID for the directory
 * @returns {Promise<Object>} Status of file creation
 */
export const copyMcpFilesToProject = async (mcpConfig, projectId) => {
  try {
    // Post request to API to create files in project directory
    const response = await fetch('/api/projects/' + projectId + '/mcp-files', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        files: {
          '.mcp.json': mcpConfig.mcpVS2026Json,
          '.vscode/settings.json': mcpConfig.mcpSettingsJson,
          '.vscode/mcp.json': mcpConfig.mcpVSCodeJson,
          '.github/copilot-instructions.md': mcpConfig.mcpCopilotInstructions,
          'scripts/mcp-agent.ps1': mcpConfig.mcpPowerShellScript,
          'scripts/mcp-agent.sh': mcpConfig.mcpBashScript
        }
      })
    })

    if (!response.ok) {
      throw new Error(`Failed to create MCP files: ${response.statusText}`)
    }

    const result = await response.json()
    return {
      success: true,
      message: 'MCP configuration files created successfully in project directory',
      files: result.files || [
        '.mcp.json',
        '.vscode/settings.json',
        '.vscode/mcp.json',
        '.github/copilot-instructions.md',
        'scripts/mcp-agent.ps1',
        'scripts/mcp-agent.sh'
      ]
    }
  } catch (error) {
    return {
      success: false,
      message: error.message,
      error: error
    }
  }
}
