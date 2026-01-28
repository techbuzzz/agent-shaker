import { computed, unref } from 'vue'

/**
 * Composable for MCP setup configuration generation
 * Handles both ref and direct values for agent, project, apiUrl, and agents
 * @param {Object|Ref} agent - Agent object or ref
 * @param {Object|Ref} project - Project object or ref
 * @param {string|Ref} apiUrl - API base URL or ref
 * @param {Array|Ref} agents - All project agents or ref
 * @returns {Object} MCP configuration objects and utilities
 */
export const useMcpSetup = (agent, project, apiUrl, agents = []) => {
  // Helper to handle both refs and direct values
  const getAgentValue = computed(() => unref(agent))
  const getProjectValue = computed(() => unref(project))
  const getApiUrl = computed(() => unref(apiUrl))
  const getAgentsValue = computed(() => unref(agents))
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
    const allAgents = getAgentsValue.value

    if (!agentValue || !projectValue) return ''

    // Helper function to generate role-specific responsibilities
    const getResponsibilities = (role) => {
      const responsibilitiesMap = {
        'frontend': `### UI/UX Implementation
- Develop responsive and accessible user interfaces
- Implement components using Vue.js, React, or other frameworks
- Ensure cross-browser and cross-device compatibility
- Handle user interactions and form validations
- Manage client-side state and data flow

### Design & Styling
- Apply modern design patterns and best practices
- Implement styling with Tailwind CSS, SCSS, or similar
- Maintain consistent visual design across the application
- Ensure proper color contrast and accessibility standards
- Optimize UI performance and bundle size

### Testing & Quality
- Write unit tests for components using Vitest or Jest
- Perform visual regression testing
- Test user interactions and edge cases
- Ensure proper error handling in UI
- Document component APIs and usage examples

### Collaboration
- Work closely with backend team for API integration
- Discuss design decisions with stakeholders
- Review code from other frontend developers
- Maintain code consistency and best practices`,
        'backend': `### API Development
- Design and implement RESTful API endpoints
- Handle request validation and error responses
- Implement proper HTTP status codes and headers
- Optimize query performance and database operations
- Document API endpoints with OpenAPI/Swagger specs

### Database Management
- Design efficient database schemas
- Create and manage database migrations
- Optimize queries and indexes
- Implement data integrity constraints
- Handle data backup and recovery

### Business Logic
- Implement core business rules and workflows
- Handle authentication and authorization
- Manage transactions and data consistency
- Implement caching strategies
- Handle edge cases and error scenarios

### Testing & Quality
- Write unit and integration tests
- Test API endpoints with various scenarios
- Implement proper logging and monitoring
- Security testing and vulnerability scanning
- Performance testing and optimization`,
        'fullstack': `### Frontend Responsibilities
- Develop responsive and accessible user interfaces
- Implement components using modern frameworks
- Manage client-side state and routing
- Ensure UI performance and optimization
- Coordinate with backend on API integration

### Backend Responsibilities
- Design and implement API endpoints
- Manage database operations and migrations
- Implement business logic and workflows
- Handle authentication and authorization
- Optimize performance and query efficiency

### Full-Stack Coordination
- Bridge communication between frontend and backend teams
- Make architectural decisions considering both sides
- Ensure consistent error handling throughout the stack
- Implement end-to-end features across all layers
- Maintain code quality standards on both sides`,
        'qa': `### Testing Strategy
- Create comprehensive test plans
- Design test cases covering happy paths and edge cases
- Execute manual and automated testing
- Document test results and findings
- Prioritize bugs by severity and impact

### Quality Assurance
- Verify requirements are met
- Check for regressions in existing features
- Test cross-browser and cross-device compatibility
- Validate accessibility standards
- Performance and load testing

### Bug Reporting & Tracking
- Document bugs with clear reproduction steps
- Create detailed test reports
- Track bug status and closure
- Coordinate with developers on fixes
- Verify bug fixes before release

### Documentation
- Create test documentation
- Document known issues and workarounds
- Maintain test case libraries
- Document testing standards and procedures`,
        'devops': `### Infrastructure Management
- Design and maintain cloud infrastructure
- Set up CI/CD pipelines
- Manage containerization (Docker, Kubernetes)
- Configure load balancing and scaling
- Ensure high availability and disaster recovery

### Deployment & Release
- Automate deployment processes
- Manage configuration management
- Coordinate production deployments
- Implement zero-downtime deployments
- Manage release schedules and rollbacks

### Monitoring & Logging
- Set up monitoring and alerting systems
- Configure centralized logging
- Monitor application and infrastructure health
- Track performance metrics
- Investigate and resolve issues

### Security & Compliance
- Implement security best practices
- Manage secrets and credentials
- Set up firewall and network policies
- Ensure compliance with regulations
- Conduct security audits`,
        'tech-lead': `### Architecture & Design
- Design system architecture
- Make technology decisions
- Plan for scalability and performance
- Ensure architectural consistency
- Document technical designs

### Code Review & Standards
- Review code from all team members
- Enforce coding standards and best practices
- Mentor developers
- Ensure code quality and maintainability
- Share knowledge and best practices

### Planning & Coordination
- Plan sprints and technical roadmap
- Identify technical debt
- Prioritize tasks and features
- Coordinate between teams
- Track progress and blockers

### Problem Solving
- Resolve technical blockers
- Investigate complex issues
- Design solutions for tricky problems
- Guide team through challenges
- Share expertise with team members`,
        'default': `### General Responsibilities
- Contribute to project development
- Follow project standards and best practices
- Collaborate with team members
- Participate in code reviews
- Document your work
- Communicate progress and blockers
- Support other team members`
      }
      return responsibilitiesMap[role] || responsibilitiesMap['default']
    }

    // Build other team members list
    const otherAgents = allAgents && Array.isArray(allAgents) 
      ? allAgents.filter(a => a.id !== agentValue.id)
      : []
    
    let teamMembersSection = ''
    if (otherAgents.length > 0) {
      teamMembersSection = `
## Other Team Members
You are working with the following agents on this project:

${otherAgents.map(agent => `- **${agent.name}** (ID: ${agent.id}, Role: ${agent.role}) - Other Identity`).join('\n')}

### Collaboration Tips
- Check their recent task assignments before starting work
- Communicate openly about dependencies
- Review their work when they request feedback
- Share knowledge and help unblock them when possible
- Respect code ownership and review processes`
    } else {
      teamMembersSection = `
## Team Information
You are the only agent assigned to this project.

### Responsibilities
- Take full ownership of all aspects of the project
- Document decisions and implementations thoroughly
- Plan and prioritize your own work
- Identify and manage dependencies`
    }

    return `# Agent Identity and MCP Integration

## Your Identity
- **Agent Name**: ${agentValue.name}
- **Agent ID**: ${agentValue.id}
- **Role**: ${agentValue.role}
- **Team**: ${agentValue.team || 'Not specified'}
- **Project**: ${projectValue.name}
- **Project ID**: ${projectValue.id}
- **Status**: ${agentValue.status}

## MCP API Configuration
- **API URL**: ${urlValue}

## Your Responsibilities
As the **${agentValue.role}** agent, your key responsibilities include:

${getResponsibilities(agentValue.role)}

## Collaboration Guidelines
1. Always check for existing tasks before starting new work
2. Update task status when you begin and complete work
3. Document important decisions and implementation details
4. Check other agents' contexts to avoid conflicts
5. Communicate about dependencies and blockers
6. Review code and provide constructive feedback
7. Share knowledge and help team members when needed
8. Follow project standards and best practices
${teamMembersSection}

## General Notes
- Keep your task descriptions updated
- Document complex logic with comments
- Ask for help when blocked
- Share knowledge with your team
- Maintain code quality standards
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
      "servers": {
        "agent-shaker": {
          "type": "http",
          "url": mcpUrl 
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
 * @param {string|Blob} content - File content as string or Blob
 * @param {string} mimeType - MIME type
 */
export const downloadFile = (filename, content, mimeType = 'text/plain') => {
  try {
    let blob
    if (content instanceof Blob) {
      blob = content
    } else {
      blob = new Blob([content], { type: mimeType })
    }
    
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Error downloading file:', error)
    throw new Error(`Failed to download file ${filename}: ${error.message}`)
  }
}

/**
 * Download all MCP files as zip
 * @param {Object} mcpConfig - MCP configuration object
 * @param {string} agentName - Agent name for filename
 * @returns {Promise<void>}
 */
export const downloadAllMcpFiles = async (mcpConfig, agentName) => {
  try {
    if (!mcpConfig) {
      throw new Error('MCP configuration is required')
    }
    if (!agentName) {
      throw new Error('Agent name is required')
    }
    
    const { default: JSZip } = await import('jszip')
    const zip = new JSZip()
    
    // Validate that all required config properties exist
    const requiredProps = ['mcpSettingsJson', 'mcpVSCodeJson', 'mcpVS2026Json', 'mcpCopilotInstructions', 'mcpPowerShellScript', 'mcpBashScript']
    for (const prop of requiredProps) {
      if (!mcpConfig[prop]) {
        throw new Error(`Missing required MCP config property: ${prop}`)
      }
    }
    
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
2. The \`.mcp.json\` file will be automatically recognized
3. Restart Visual Studio 2026 to apply MCP configuration
4. Start using Copilot with your agent identity!

### For VS Code:
1. Extract to project root
2. The \`.vscode/settings.json\` and \`.vscode/mcp.json\` will be applied
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
  } catch (error) {
    console.error('Error creating MCP files zip:', error)
    throw new Error(`Failed to download MCP files: ${error.message}`)
  }
}

/**
 * Copy MCP files directly to project directory via API
 * @param {Object} mcpConfig - MCP configuration object
 * @param {string} projectId - Project ID for the directory
 * @returns {Promise<Object>} Status of file creation
 */
export const copyMcpFilesToProject = async (mcpConfig, projectId) => {
  try {
    if (!mcpConfig) {
      throw new Error('MCP configuration is required')
    }
    if (!projectId) {
      throw new Error('Project ID is required')
    }
    
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

    if (!response) {
      throw new Error('No response from server')
    }

    if (!response.ok) {
      const errorText = await response.text()
      throw new Error(`Failed to create MCP files: ${response.statusText}. ${errorText}`)
    }

    let result = {}
    const contentType = response.headers.get('content-type')
    
    if (contentType && contentType.includes('application/json')) {
      result = await response.json()
    }
    
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
    console.error('Error copying MCP files to project:', error)
    return {
      success: false,
      message: error.message,
      error: error
    }
  }
}
