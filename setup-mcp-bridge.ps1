# Setup MCP Bridge for Agent Shaker
# This script installs dependencies and tests the bridge

Write-Host "🚀 Agent Shaker MCP Bridge Setup" -ForegroundColor Green
Write-Host ""

# Check if Node.js is installed
Write-Host "Checking Node.js installation..." -ForegroundColor Yellow
try {
    $nodeVersion = node --version
    Write-Host "✅ Node.js $nodeVersion found" -ForegroundColor Green
} catch {
    Write-Host "❌ Node.js not found. Please install Node.js from https://nodejs.org/" -ForegroundColor Red
    exit 1
}

# Check if npm is installed
Write-Host "Checking npm installation..." -ForegroundColor Yellow
try {
    $npmVersion = npm --version
    Write-Host "✅ npm $npmVersion found" -ForegroundColor Green
} catch {
    Write-Host "❌ npm not found. Please install npm" -ForegroundColor Red
    exit 1
}

# Install dependencies
Write-Host ""
Write-Host "Installing dependencies..." -ForegroundColor Yellow
npm install

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Dependencies installed successfully" -ForegroundColor Green
} else {
    Write-Host "❌ Failed to install dependencies" -ForegroundColor Red
    exit 1
}

# Check if containers are running
Write-Host ""
Write-Host "Checking if Agent Shaker containers are running..." -ForegroundColor Yellow
$containers = docker-compose ps --services --filter "status=running"

if ($containers -match "mcp-server") {
    Write-Host "✅ MCP Server is running" -ForegroundColor Green
} else {
    Write-Host "⚠️  MCP Server is not running. Starting containers..." -ForegroundColor Yellow
    docker-compose up -d
    Start-Sleep -Seconds 5
}

# Test API connection
Write-Host ""
Write-Host "Testing API connection..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri http://localhost:8080/api/projects -UseBasicParsing -TimeoutSec 5
    if ($response.StatusCode -eq 200) {
        Write-Host "✅ API is accessible" -ForegroundColor Green
    }
} catch {
    Write-Host "❌ API is not accessible. Please check if containers are running" -ForegroundColor Red
    Write-Host "   Run: docker-compose up -d" -ForegroundColor Yellow
    exit 1
}

# Success message
Write-Host ""
Write-Host "╔═══════════════════════════════════════════╗" -ForegroundColor Green
Write-Host "║   ✅ Setup Complete!                      ║" -ForegroundColor Green
Write-Host "╚═══════════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""
Write-Host "To start the MCP bridge, run:" -ForegroundColor Cyan
Write-Host "  npm start" -ForegroundColor Yellow
Write-Host ""
Write-Host "Or use Node directly:" -ForegroundColor Cyan
Write-Host "  node mcp-bridge.js" -ForegroundColor Yellow
Write-Host ""
Write-Host "For usage instructions, see:" -ForegroundColor Cyan
Write-Host "  COPILOT_MCP_INTEGRATION.md" -ForegroundColor Yellow
Write-Host ""
