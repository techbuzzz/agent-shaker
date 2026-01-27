#!/usr/bin/env pwsh
# Create a new migration file with proper naming and timestamp

param(
    [Parameter(Mandatory=$true)]
    [string]$Name
)

# Get the migrations directory
$migrationsDir = Join-Path $PSScriptRoot ".." "migrations"

# Get the next migration number
$existingMigrations = Get-ChildItem -Path $migrationsDir -Filter "*.sql" | 
    Where-Object { $_.Name -match '^\d{3}_' } |
    Sort-Object Name

$nextNumber = 1
if ($existingMigrations.Count -gt 0) {
    $lastMigration = $existingMigrations[-1]
    if ($lastMigration.Name -match '^(\d{3})_') {
        $nextNumber = [int]$matches[1] + 1
    }
}

# Format migration number with leading zeros
$migrationNumber = $nextNumber.ToString("000")

# Sanitize migration name (replace spaces with underscores, lowercase)
$sanitizedName = $Name.ToLower() -replace '\s+', '_' -replace '[^a-z0-9_]', ''

# Create migration filename
$filename = "${migrationNumber}_${sanitizedName}.sql"
$filepath = Join-Path $migrationsDir $filename

# Create migration file with template
$template = @"
-- Migration: $Name
-- Created: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
-- Description: Add description here

-- Add your migration SQL here

"@

Set-Content -Path $filepath -Value $template -Encoding UTF8

Write-Host "✓ Created migration: $filename" -ForegroundColor Green
Write-Host "  Path: $filepath" -ForegroundColor Gray
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Edit the migration file to add your SQL"
Write-Host "  2. Test locally before deploying"
Write-Host "  3. Commit the migration file to version control"
