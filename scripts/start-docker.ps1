# Start Agent Shaker with Docker Compose
# This script starts the complete application stack

Write-Host "🚀 Starting Agent Shaker..." -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if docker-compose is available
try {
    $null = Get-Command docker-compose -ErrorAction Stop
} catch {
    Write-Host "❌ docker-compose not found. Please install Docker Desktop or Docker Compose." -ForegroundColor Red
    exit 1
}

# Check if Docker is running
try {
    $null = docker info 2>$null
} catch {
    Write-Host "❌ Docker is not running. Please start Docker Desktop." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Docker is running" -ForegroundColor Green

# Stop any existing containers
Write-Host ""
Write-Host "🛑 Stopping existing containers..." -ForegroundColor Yellow
docker-compose down

# Start services
Write-Host ""
Write-Host "🏗️  Building and starting services..." -ForegroundColor Yellow
Write-Host "   - PostgreSQL (Database)"
Write-Host "   - MCP Server (Go API + WebSocket)"
Write-Host "   - Web App (Vue.js UI)"
Write-Host ""

docker-compose up --build -d

# Wait for services to be ready
Write-Host ""
Write-Host "⏳ Waiting for services to start..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Check service health
Write-Host ""
Write-Host "🔍 Checking service health..." -ForegroundColor Yellow

# Check PostgreSQL
try {
    $postgresHealth = docker-compose ps postgres | Select-String "Up"
    if ($postgresHealth) {
        Write-Host "✅ PostgreSQL: Running" -ForegroundColor Green
    } else {
        Write-Host "❌ PostgreSQL: Not running" -ForegroundColor Red
    }
} catch {
    Write-Host "❌ PostgreSQL: Error checking status" -ForegroundColor Red
}

# Check MCP Server
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/health" -TimeoutSec 5 -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
        Write-Host "✅ MCP Server: Running (http://localhost:8080)" -ForegroundColor Green
    }
} catch {
    Write-Host "❌ MCP Server: Not responding" -ForegroundColor Red
}

# Check Web App
try {
    $response = Invoke-WebRequest -Uri "http://localhost" -TimeoutSec 5 -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
        Write-Host "✅ Web App: Running (http://localhost)" -ForegroundColor Green
    }
} catch {
    Write-Host "❌ Web App: Not responding" -ForegroundColor Red
}

Write-Host ""
Write-Host "🎉 Agent Shaker is ready!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "🌐 Web UI:     http://localhost" -ForegroundColor Cyan
Write-Host "🔌 API:        http://localhost/api" -ForegroundColor Cyan
Write-Host "📡 WebSocket:  ws://localhost/ws" -ForegroundColor Cyan
Write-Host "🐘 Database:   localhost:5433" -ForegroundColor Cyan
Write-Host ""
Write-Host "📋 Useful commands:" -ForegroundColor Gray
Write-Host "   docker-compose logs -f          # View logs"
Write-Host "   docker-compose down             # Stop services"
Write-Host "   .\scripts\verify-api.ps1        # Test API endpoints"
Write-Host ""
Write-Host "Press Ctrl+C to stop viewing logs (services continue running)" -ForegroundColor Gray
Write-Host ""

# Show logs
docker-compose logs -f
