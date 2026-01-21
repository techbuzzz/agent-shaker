# API Verification Script
# Run this after starting the backend server to verify all endpoints work correctly

$baseUrl = "http://localhost:8080/api"
$headers = @{
    "Content-Type" = "application/json"
}

Write-Host "=== API Endpoint Verification ===" -ForegroundColor Cyan
Write-Host ""

# Initialize variables at script scope
$projectId = $null
$agentId = $null
$taskId = $null
$contextId = $null

# Test 1: Health Check
Write-Host "1. Testing Health Check..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
    Write-Host "   ✓ Health check passed" -ForegroundColor Green
} catch {
    Write-Host "   ✗ Health check failed: $($_.Exception.Message)" -ForegroundColor Red
}

# Test 2: List Projects
Write-Host "2. Testing GET /api/projects..." -ForegroundColor Yellow
try {
    $projects = Invoke-RestMethod -Uri "$baseUrl/projects" -Method Get -Headers $headers
    Write-Host "   ✓ Projects retrieved: $($projects.Count) projects" -ForegroundColor Green
} catch {
    Write-Host "   ✗ Failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "   Error Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
}

# Test 3: Create Project
Write-Host "3. Testing POST /api/projects..." -ForegroundColor Yellow
try {
    $projectData = @{
        name = "Test Project $(Get-Date -Format 'yyyyMMdd-HHmmss')"
        description = "API verification test project"
    } | ConvertTo-Json
    
    $newProject = Invoke-RestMethod -Uri "$baseUrl/projects" -Method Post -Headers $headers -Body $projectData
    
    if (-not $newProject) {
        throw "Project creation returned null response"
    }
    if (-not $newProject.id) {
        Write-Host "   Response: $($newProject | ConvertTo-Json)" -ForegroundColor Gray
        throw "Project creation response missing 'id' field"
    }
    
    Write-Host "   ✓ Project created: $($newProject.name)" -ForegroundColor Green
    $script:projectId = $newProject.id
    Write-Host "   Project ID: $script:projectId" -ForegroundColor Gray
    
    # Test 4: Get Single Project
    Write-Host "4. Testing GET /api/projects/{id}..." -ForegroundColor Yellow
    if (-not $projectId) {
        throw "Project ID is null, cannot continue with tests"
    }
    $project = Invoke-RestMethod -Uri "$baseUrl/projects/$projectId" -Method Get -Headers $headers
    Write-Host "   ✓ Project retrieved: $($project.name)" -ForegroundColor Green
    
    # Test 5: List Agents (empty)
    Write-Host "5. Testing GET /api/agents..." -ForegroundColor Yellow
    $agents = Invoke-RestMethod -Uri "$baseUrl/agents" -Method Get -Headers $headers
    Write-Host "   ✓ Agents retrieved: $($agents.Count) agents" -ForegroundColor Green
    
    # Test 6: List Agents by Project
    Write-Host "6. Testing GET /api/agents?project_id={id}..." -ForegroundColor Yellow
    $projectAgents = Invoke-RestMethod -Uri "$baseUrl/agents?project_id=$projectId" -Method Get -Headers $headers
    Write-Host "   ✓ Project agents retrieved: $($projectAgents.Count) agents" -ForegroundColor Green
    
    # Test 7: Create Agent
    Write-Host "7. Testing POST /api/agents..." -ForegroundColor Yellow
    if (-not $projectId) {
        throw "Project ID is null, cannot create agent"
    }
    
    $agentData = @{
        project_id = if ($projectId -is [string]) { $projectId } else { $projectId.ToString() }
        name = "Test Agent"
        role = "backend"
        team = "development"
    } | ConvertTo-Json
    
    Write-Host "   Sending: $agentData" -ForegroundColor Gray
    $newAgent = Invoke-RestMethod -Uri "$baseUrl/agents" -Method Post -Headers $headers -Body $agentData
    
    if (-not $newAgent -or -not $newAgent.id) {
        Write-Host "   Response: $($newAgent | ConvertTo-Json)" -ForegroundColor Gray
        throw "Agent creation failed or returned invalid response"
    }
    
    Write-Host "   ✓ Agent created: $($newAgent.name)" -ForegroundColor Green
    $script:agentId = $newAgent.id
    Write-Host "   Agent ID: $script:agentId" -ForegroundColor Gray
    
    # Test 8: Get Single Agent
    Write-Host "8. Testing GET /api/agents/{id}..." -ForegroundColor Yellow
    $agent = Invoke-RestMethod -Uri "$baseUrl/agents/$agentId" -Method Get -Headers $headers
    Write-Host "   ✓ Agent retrieved: $($agent.name)" -ForegroundColor Green
    
    # Test 9: Update Agent Status
    Write-Host "9. Testing PUT /api/agents/{id}/status..." -ForegroundColor Yellow
    $statusData = @{
        status = "idle"
    } | ConvertTo-Json
    
    $updatedAgent = Invoke-RestMethod -Uri "$baseUrl/agents/$agentId/status" -Method Put -Headers $headers -Body $statusData
    Write-Host "   ✓ Agent status updated: $($updatedAgent.status)" -ForegroundColor Green
    
    # Test 10: List Tasks (empty)
    Write-Host "10. Testing GET /api/tasks..." -ForegroundColor Yellow
    $tasks = Invoke-RestMethod -Uri "$baseUrl/tasks" -Method Get -Headers $headers
    Write-Host "   ✓ Tasks retrieved: $($tasks.Count) tasks" -ForegroundColor Green
    
    # Test 11: Create Task
    Write-Host "11. Testing POST /api/tasks..." -ForegroundColor Yellow
    if (-not $projectId -or -not $agentId) {
        throw "Project ID or Agent ID is null, cannot create task"
    }
    
    $taskData = @{
        project_id = if ($projectId -is [string]) { $projectId } else { $projectId.ToString() }
        title = "Test Task"
        description = "API verification test task"
        priority = "high"
        created_by = if ($agentId -is [string]) { $agentId } else { $agentId.ToString() }
        assigned_to = if ($agentId -is [string]) { $agentId } else { $agentId.ToString() }
    } | ConvertTo-Json
    
    $newTask = Invoke-RestMethod -Uri "$baseUrl/tasks" -Method Post -Headers $headers -Body $taskData
    
    if (-not $newTask -or -not $newTask.id) {
        Write-Host "   Response: $($newTask | ConvertTo-Json)" -ForegroundColor Gray
        throw "Task creation failed or returned invalid response"
    }
    
    Write-Host "   ✓ Task created: $($newTask.title)" -ForegroundColor Green
    $script:taskId = $newTask.id
    
    # Test 12: Get Single Task
    Write-Host "12. Testing GET /api/tasks/{id}..." -ForegroundColor Yellow
    $task = Invoke-RestMethod -Uri "$baseUrl/tasks/$taskId" -Method Get -Headers $headers
    Write-Host "   ✓ Task retrieved: $($task.title)" -ForegroundColor Green
    
    # Test 13: List Tasks by Project
    Write-Host "13. Testing GET /api/tasks?project_id={id}..." -ForegroundColor Yellow
    $projectTasks = Invoke-RestMethod -Uri "$baseUrl/tasks?project_id=$projectId" -Method Get -Headers $headers
    Write-Host "   ✓ Project tasks retrieved: $($projectTasks.Count) tasks" -ForegroundColor Green
    
    # Test 14: List Tasks by Agent
    Write-Host "14. Testing GET /api/tasks?agent_id={id}..." -ForegroundColor Yellow
    $agentTasks = Invoke-RestMethod -Uri "$baseUrl/tasks?agent_id=$agentId" -Method Get -Headers $headers
    Write-Host "   ✓ Agent tasks retrieved: $($agentTasks.Count) tasks" -ForegroundColor Green
    
    # Test 15: Update Task Status
    Write-Host "15. Testing PUT /api/tasks/{id}/status..." -ForegroundColor Yellow
    $taskStatusData = @{
        status = "in_progress"
    } | ConvertTo-Json
    
    $updatedTask = Invoke-RestMethod -Uri "$baseUrl/tasks/$taskId/status" -Method Put -Headers $headers -Body $taskStatusData
    Write-Host "   ✓ Task status updated: $($updatedTask.status)" -ForegroundColor Green
    
    # Test 16: Update Task (full)
    Write-Host "16. Testing PUT /api/tasks/{id}..." -ForegroundColor Yellow
    $taskUpdateData = @{
        status = "completed"
        output = "Task completed successfully"
    } | ConvertTo-Json
    
    $fullyUpdatedTask = Invoke-RestMethod -Uri "$baseUrl/tasks/$taskId" -Method Put -Headers $headers -Body $taskUpdateData
    Write-Host "   ✓ Task fully updated: $($fullyUpdatedTask.status)" -ForegroundColor Green
    
    # Test 17: List Contexts
    Write-Host "17. Testing GET /api/contexts..." -ForegroundColor Yellow
    $contexts = Invoke-RestMethod -Uri "$baseUrl/contexts" -Method Get -Headers $headers
    Write-Host "   ✓ Contexts retrieved: $($contexts.Count) contexts" -ForegroundColor Green
    
    # Test 18: Create Context
    Write-Host "18. Testing POST /api/contexts..." -ForegroundColor Yellow
    if (-not $projectId) {
        throw "Project ID is null, cannot create context"
    }
    
    $contextData = @{
        project_id = if ($projectId -is [string]) { $projectId } else { $projectId.ToString() }
        type = "documentation"
        title = "Test Context"
        content = "This is a test context for API verification"
    } | ConvertTo-Json
    
    $newContext = Invoke-RestMethod -Uri "$baseUrl/contexts" -Method Post -Headers $headers -Body $contextData
    
    if (-not $newContext -or -not $newContext.id) {
        Write-Host "   Response: $($newContext | ConvertTo-Json)" -ForegroundColor Gray
        throw "Context creation failed or returned invalid response"
    }
    
    Write-Host "   ✓ Context created: $($newContext.title)" -ForegroundColor Green
    $script:contextId = $newContext.id
    
    # Test 19: Get Single Context
    Write-Host "19. Testing GET /api/contexts/{id}..." -ForegroundColor Yellow
    $context = Invoke-RestMethod -Uri "$baseUrl/contexts/$contextId" -Method Get -Headers $headers
    Write-Host "   ✓ Context retrieved: $($context.title)" -ForegroundColor Green
    
    # Test 20: List Contexts by Project
    Write-Host "20. Testing GET /api/contexts?project_id={id}..." -ForegroundColor Yellow
    $projectContexts = Invoke-RestMethod -Uri "$baseUrl/contexts?project_id=$projectId" -Method Get -Headers $headers
    Write-Host "   ✓ Project contexts retrieved: $($projectContexts.Count) contexts" -ForegroundColor Green
    
    # Test 21: Update Context
    Write-Host "21. Testing PUT /api/contexts/{id}..." -ForegroundColor Yellow
    $contextUpdateData = @{
        title = "Updated Test Context"
        content = "This context has been updated"
    } | ConvertTo-Json
    
    $updatedContext = Invoke-RestMethod -Uri "$baseUrl/contexts/$contextId" -Method Put -Headers $headers -Body $contextUpdateData
    Write-Host "   ✓ Context updated: $($updatedContext.title)" -ForegroundColor Green
    
    # Test 22: Delete Context
    Write-Host "22. Testing DELETE /api/contexts/{id}..." -ForegroundColor Yellow
    Invoke-RestMethod -Uri "$baseUrl/contexts/$contextId" -Method Delete -Headers $headers
    Write-Host "   ✓ Context deleted" -ForegroundColor Green
    
    Write-Host ""
    Write-Host "=== All Tests Passed! ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "Test Resources Created:" -ForegroundColor Cyan
    Write-Host "  Project ID: $projectId"
    Write-Host "  Agent ID: $agentId"
    Write-Host "  Task ID: $taskId"
    
} catch {
    Write-Host "   ✗ Test failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "   Error Details: $($_.ErrorDetails.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Note: You can manually clean up test data by deleting the test project from the database." -ForegroundColor Gray
