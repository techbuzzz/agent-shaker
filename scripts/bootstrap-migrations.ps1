#!/usr/bin/env pwsh
# Bootstrap existing database with migration tracking
# Run this ONCE if you have an existing Agent Shaker database

param(
    [string]$DatabaseUrl = $env:DATABASE_URL
)

if (-not $DatabaseUrl) {
    $DatabaseUrl = "postgres://mcp:secret@localhost:5433/mcp_tracker?sslmode=disable"
    Write-Host "Using default DATABASE_URL: $DatabaseUrl" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "╔═══════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║  Agent Shaker Database Migration Bootstrap               ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "This script will:" -ForegroundColor White
Write-Host "  1. Create schema_migrations table" -ForegroundColor Gray
Write-Host "  2. Detect existing database schema" -ForegroundColor Gray
Write-Host "  3. Mark applied migrations without re-running them" -ForegroundColor Gray
Write-Host ""

# Confirm action
$confirm = Read-Host "Continue? (y/N)"
if ($confirm -ne 'y' -and $confirm -ne 'Y') {
    Write-Host "Aborted." -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "Running bootstrap..." -ForegroundColor Cyan

# Check if psql is available
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue
if (-not $psqlPath) {
    Write-Host "Error: psql command not found" -ForegroundColor Red
    Write-Host "Please install PostgreSQL client tools" -ForegroundColor Yellow
    exit 1
}

# Run bootstrap script
$bootstrapFile = Join-Path $PSScriptRoot ".." "migrations" "bootstrap_existing_db.sql"

if (-not (Test-Path $bootstrapFile)) {
    Write-Host "Error: Bootstrap file not found: $bootstrapFile" -ForegroundColor Red
    exit 1
}

try {
    $env:PGPASSWORD = if ($DatabaseUrl -match 'postgres://([^:]+):([^@]+)@') { $matches[2] } else { "" }
    
    psql $DatabaseUrl -f $bootstrapFile 2>&1 | ForEach-Object {
        if ($_ -match "NOTICE:") {
            Write-Host $_ -ForegroundColor Green
        } elseif ($_ -match "ERROR:") {
            Write-Host $_ -ForegroundColor Red
        } else {
            Write-Host $_ -ForegroundColor Gray
        }
    }
    
    Write-Host ""
    Write-Host "✓ Bootstrap completed successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Yellow
    Write-Host "  1. Start your application normally" -ForegroundColor Gray
    Write-Host "  2. New migrations will be applied automatically" -ForegroundColor Gray
    Write-Host "  3. Check logs for migration status" -ForegroundColor Gray
} catch {
    Write-Host ""
    Write-Host "✗ Bootstrap failed: $_" -ForegroundColor Red
    exit 1
}
