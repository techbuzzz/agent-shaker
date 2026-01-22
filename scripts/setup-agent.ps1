# Agent Setup Script for MCP Application
# This script helps you quickly set up a project and register agents

param(
    [Parameter(Mandatory=$false)]
    [string]$ProjectName = "",
    
    [Parameter(Mandatory=$false)]
    [string]$AgentName = "",
    
    [Parameter(Mandatory=$false)]
    [ValidateSet(
        # Development Roles
        "frontend", "backend", "fullstack", "mobile", "devops", "qa", "security",
        # Agile Roles
        "product-owner", "scrum-master", "agile-coach",
        # R&D Roles
        "architect", "tech-lead", "researcher", "data-scientist", "ml-engineer",
        # Design & UX
        "ux-designer", "ui-designer", "ux-researcher"
    )]
    [string]$Role = "",
    
    [Parameter(Mandatory=$false)]
    [switch]$Interactive = $false
)

$baseUrl = "http://localhost:8080/api"

# Colors
function Write-Success { param([string]$Message) Write-Host "✅ $Message" -ForegroundColor Green }
function Write-Info { param([string]$Message) Write-Host "ℹ️  $Message" -ForegroundColor Cyan }
function Write-Warning { param([string]$Message) Write-Host "⚠️  $Message" -ForegroundColor Yellow }
function Write-Error { param([string]$Message) Write-Host "❌ $Message" -ForegroundColor Red }

# Check if server is running
function Test-MCPServer {
    try {
        $health = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 3
        if ($health -eq "OK") {
            Write-Success "MCP Server is running"
            return $true
        }
    }
    catch {
        Write-Error "MCP Server is not running!"
        Write-Info "Please start it with: docker-compose up -d"
        return $false
    }
}

# Interactive mode
if ($Interactive) {
    Write-Host ""
    Write-Host "=== MCP Agent Setup Wizard ===" -ForegroundColor Yellow
    Write-Host ""
    
    if (-not (Test-MCPServer)) { exit 1 }
    
    # Get project name
    if ([string]::IsNullOrWhiteSpace($ProjectName)) {
        $ProjectName = Read-Host "Enter project name (e.g., InvoiceAI)"
    }
    
    # Get agent name
    if ([string]::IsNullOrWhiteSpace($AgentName)) {
        $AgentName = Read-Host "Enter agent name (e.g., InvoiceAI-Frontend)"
    }
    
    # Get role
    if ([string]::IsNullOrWhiteSpace($Role)) {
        Write-Host ""
        Write-Host "Select agent role:" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Development Roles:" -ForegroundColor Cyan
        Write-Host "  1. Frontend Developer"
        Write-Host "  2. Backend Developer"
        Write-Host "  3. Full Stack Developer"
        Write-Host "  4. Mobile Developer"
        Write-Host "  5. DevOps Engineer"
        Write-Host "  6. QA Engineer"
        Write-Host "  7. Security Engineer"
        Write-Host ""
        Write-Host "Agile Roles:" -ForegroundColor Cyan
        Write-Host "  8. Product Owner"
        Write-Host "  9. Scrum Master"
        Write-Host " 10. Agile Coach"
        Write-Host ""
        Write-Host "R&D Roles:" -ForegroundColor Cyan
        Write-Host " 11. Solution Architect"
        Write-Host " 12. Tech Lead"
        Write-Host " 13. Research Engineer"
        Write-Host " 14. Data Scientist"
        Write-Host " 15. ML Engineer"
        Write-Host ""
        Write-Host "Design & UX:" -ForegroundColor Cyan
        Write-Host " 16. UX Designer"
        Write-Host " 17. UI Designer"
        Write-Host " 18. UX Researcher"
        Write-Host ""
        $roleChoice = Read-Host "Enter choice (1-18)"
        
        $Role = switch ($roleChoice) {
            "1"  { "frontend" }
            "2"  { "backend" }
            "3"  { "fullstack" }
            "4"  { "mobile" }
            "5"  { "devops" }
            "6"  { "qa" }
            "7"  { "security" }
            "8"  { "product-owner" }
            "9"  { "scrum-master" }
            "10" { "agile-coach" }
            "11" { "architect" }
            "12" { "tech-lead" }
            "13" { "researcher" }
            "14" { "data-scientist" }
            "15" { "ml-engineer" }
            "16" { "ux-designer" }
            "17" { "ui-designer" }
            "18" { "ux-researcher" }
            default { "frontend" }
        }
    }
    
    Write-Host ""
    Write-Info "Creating project: $ProjectName"
    Write-Info "Registering agent: $AgentName"
    Write-Info "Role: $Role"
    Write-Host ""
    
    $confirm = Read-Host "Continue? (y/n)"
    if ($confirm -ne "y") {
        Write-Warning "Setup cancelled"
        exit 0
    }
}

# Validate parameters
if ([string]::IsNullOrWhiteSpace($ProjectName) -or 
    [string]::IsNullOrWhiteSpace($AgentName) -or 
    [string]::IsNullOrWhiteSpace($Role)) {
    Write-Error "Missing required parameters!"
    Write-Host ""
    Write-Host "Usage Examples:"
    Write-Host "  Interactive mode:"
    Write-Host "    .\setup-agent.ps1 -Interactive"
    Write-Host ""
    Write-Host "  Command line:"
    Write-Host "    .\setup-agent.ps1 -ProjectName 'InvoiceAI' -AgentName 'InvoiceAI-Frontend' -Role 'frontend'"
    Write-Host "    .\setup-agent.ps1 -ProjectName 'DataPlatform' -AgentName 'ML-Engineer' -Role 'ml-engineer'"
    Write-Host "    .\setup-agent.ps1 -ProjectName 'AppProject' -AgentName 'ProductOwner' -Role 'product-owner'"
    Write-Host ""
    Write-Host "Available Roles:"
    Write-Host "  Development: frontend, backend, fullstack, mobile, devops, qa, security"
    Write-Host "  Agile: product-owner, scrum-master, agile-coach"
    Write-Host "  R&D: architect, tech-lead, researcher, data-scientist, ml-engineer"
    Write-Host "  Design: ux-designer, ui-designer, ux-researcher"
    Write-Host ""
    exit 1
}

# Check server
if (-not (Test-MCPServer)) { exit 1 }

try {
    # Create Project
    Write-Info "Creating project '$ProjectName'..."
    $projectBody = @{
        name = $ProjectName
        description = "Project managed by MCP agents"
    } | ConvertTo-Json
    
    $project = Invoke-RestMethod -Uri "$baseUrl/projects" `
        -Method POST `
        -ContentType "application/json" `
        -Body $projectBody
    
    Write-Success "Project created: $($project.name)"
    Write-Host "  Project ID: $($project.id)" -ForegroundColor Gray
    
    # Register Agent
    Write-Info "Registering agent '$AgentName'..."
    $agentBody = @{
        project_id = $project.id
        name = $AgentName
        role = $Role
        team = "$Role Development Team"
    } | ConvertTo-Json
    
    $agent = Invoke-RestMethod -Uri "$baseUrl/agents" `
        -Method POST `
        -ContentType "application/json" `
        -Body $agentBody
    
    Write-Success "Agent registered: $($agent.name)"
    Write-Host "  Agent ID: $($agent.id)" -ForegroundColor Gray
    Write-Host "  Role: $($agent.role)" -ForegroundColor Gray
    
    # Create agent configuration file
    Write-Info "Creating agent configuration file..."
    
    $agentConfig = @{
        mcp = @{
            server_url = $baseUrl
            project_id = $project.id
            project_name = $project.name
            agent_id = $agent.id
            agent_name = $agent.name
            role = $agent.role
            team = $agent.team
            created_at = $agent.created_at
        }
    } | ConvertTo-Json -Depth 10
    
    $configFile = "agent-config.json"
    $agentConfig | Out-File -FilePath $configFile -Encoding utf8
    
    Write-Success "Configuration saved to: $configFile"
    
    # Create helper script
    Write-Info "Creating helper script..."
    
    $helperScript = @"
# Agent Helper Script - Auto-generated
# Agent: $AgentName
# Project: $ProjectName

`$agentId = "$($agent.id)"
`$projectId = "$($project.id)"
`$apiUrl = "$baseUrl"

function Get-MyTasks {
    Write-Host "Fetching tasks for $AgentName..." -ForegroundColor Cyan
    `$tasks = Invoke-RestMethod -Uri "`$apiUrl/agents/`$agentId/tasks"
    `$tasks | Format-Table -Property title, status, priority, created_at
}

function New-Task {
    param(
        [Parameter(Mandatory=`$true)]
        [string]`$Title,
        
        [Parameter(Mandatory=`$false)]
        [string]`$Description = "",
        
        [Parameter(Mandatory=`$false)]
        [ValidateSet("low", "medium", "high")]
        [string]`$Priority = "medium"
    )
    
    `$taskBody = @{
        project_id = `$projectId
        agent_id = `$agentId
        title = `$Title
        description = `$Description
        priority = `$Priority
        dependencies = @()
    } | ConvertTo-Json
    
    `$task = Invoke-RestMethod -Uri "`$apiUrl/tasks" ``
        -Method POST ``
        -ContentType "application/json" ``
        -Body `$taskBody
    
    Write-Host "✅ Task created: `$(`$task.title)" -ForegroundColor Green
    Write-Host "   Task ID: `$(`$task.id)" -ForegroundColor Gray
    return `$task
}

function Update-TaskStatus {
    param(
        [Parameter(Mandatory=`$true)]
        [string]`$TaskId,
        
        [Parameter(Mandatory=`$true)]
        [ValidateSet("pending", "in_progress", "done", "blocked")]
        [string]`$Status
    )
    
    `$statusBody = @{ status = `$Status } | ConvertTo-Json
    
    Invoke-RestMethod -Uri "`$apiUrl/tasks/`$TaskId/status" ``
        -Method PUT ``
        -ContentType "application/json" ``
        -Body `$statusBody
    
    Write-Host "✅ Task status updated to: `$Status" -ForegroundColor Green
}

function Add-TaskContext {
    param(
        [Parameter(Mandatory=`$true)]
        [string]`$TaskId,
        
        [Parameter(Mandatory=`$true)]
        [string]`$Context
    )
    
    `$contextBody = @{
        task_id = `$TaskId
        context = `$Context
    } | ConvertTo-Json
    
    Invoke-RestMethod -Uri "`$apiUrl/documentation" ``
        -Method POST ``
        -ContentType "application/json" ``
        -Body `$contextBody
    
    Write-Host "✅ Context added to task" -ForegroundColor Green
}

function Show-AgentInfo {
    Write-Host ""
    Write-Host "=== Agent Information ===" -ForegroundColor Yellow
    Write-Host "Agent Name: $AgentName"
    Write-Host "Agent ID: `$agentId"
    Write-Host "Project: $ProjectName"
    Write-Host "Project ID: `$projectId"
    Write-Host "Role: $Role"
    Write-Host "API URL: `$apiUrl"
    Write-Host "=========================" -ForegroundColor Yellow
    Write-Host ""
}

# Export functions
Export-ModuleMember -Function Get-MyTasks, New-Task, Update-TaskStatus, Add-TaskContext, Show-AgentInfo

Write-Host ""
Write-Host "Agent helper loaded! Available commands:" -ForegroundColor Green
Write-Host "  Get-MyTasks           - View your tasks"
Write-Host "  New-Task              - Create a new task"
Write-Host "  Update-TaskStatus     - Update task status"
Write-Host "  Add-TaskContext       - Add documentation to task"
Write-Host "  Show-AgentInfo        - Display agent information"
Write-Host ""
Write-Host "Example: Get-MyTasks" -ForegroundColor Cyan
Write-Host ""
"@
    
    $helperFile = "agent-helper-$($AgentName.Replace(' ', '-')).ps1"
    $helperScript | Out-File -FilePath $helperFile -Encoding utf8
    
    Write-Success "Helper script created: $helperFile"
    
    # Create .env file
    Write-Info "Creating .env file..."
    
    $envContent = @"
# MCP Agent Configuration - Auto-generated
# Created: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

MCP_SERVER_URL=$baseUrl
MCP_PROJECT_ID=$($project.id)
MCP_PROJECT_NAME=$ProjectName
MCP_AGENT_ID=$($agent.id)
MCP_AGENT_NAME=$AgentName
MCP_AGENT_ROLE=$Role
"@
    
    $envContent | Out-File -FilePath ".env.agent" -Encoding utf8
    
    Write-Success ".env.agent file created"
    
    # Summary
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host "     🎉 Setup Complete! 🎉" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Project Information:" -ForegroundColor Cyan
    Write-Host "  Name: $($project.name)"
    Write-Host "  ID: $($project.id)"
    Write-Host ""
    Write-Host "Agent Information:" -ForegroundColor Cyan
    Write-Host "  Name: $($agent.name)"
    Write-Host "  ID: $($agent.id)"
    Write-Host "  Role: $($agent.role)"
    Write-Host ""
    Write-Host "Files Created:" -ForegroundColor Cyan
    Write-Host "  📄 $configFile"
    Write-Host "  📄 $helperFile"
    Write-Host "  📄 .env.agent"
    Write-Host ""
    Write-Host "Next Steps:" -ForegroundColor Yellow
    Write-Host "  1. Load helper functions:"
    Write-Host "     . .\$helperFile" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  2. View your tasks:"
    Write-Host "     Get-MyTasks" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  3. Create a new task:"
    Write-Host "     New-Task -Title 'Build login page' -Priority 'high'" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  4. See full documentation:"
    Write-Host "     Get-Content AGENT_SETUP_GUIDE.md" -ForegroundColor Gray
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Yellow
    
}
catch {
    Write-Error "Setup failed: $($_.Exception.Message)"
    Write-Host $_.Exception.StackTrace -ForegroundColor Red
    exit 1
}
