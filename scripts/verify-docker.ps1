# Every check this project has, run in containers.
#
# The Windows counterpart of scripts/verify-docker.sh. Same rules: nothing is
# installed on the host, nothing is run on the host. A checkout is not a
# toolchain.
#
# Usage:
#   scripts\verify-docker.ps1              # every check
#   scripts\verify-docker.ps1 -Target go   # the Go checks only
#   scripts\verify-docker.ps1 -Target web  # the frontend checks only
#   scripts\verify-docker.ps1 -Target integration

[CmdletBinding()]
param(
    [ValidateSet('all', 'go', 'web', 'integration')]
    [string]$Target = 'all'
)

$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$ComposeFile = Join-Path $Root 'docker-compose.test.yml'
$ComposeProject = 'agent-shaker-test'

$GoImage = if ($env:GO_IMAGE) { $env:GO_IMAGE } else { 'golang:1.26-alpine' }
$LintImage = if ($env:LINT_IMAGE) { $env:LINT_IMAGE } else { 'golangci/golangci-lint:v2.15.0' }
$NodeImage = if ($env:NODE_IMAGE) { $env:NODE_IMAGE } else { 'node:22-slim' }

$GoCacheVolume = 'agent-shaker-go-cache'
$GoModCacheVolume = 'agent-shaker-go-mod-cache'
$NpmCacheVolume = 'agent-shaker-npm-cache'

function Write-Step([string]$Message) {
    Write-Host "`n> $Message" -ForegroundColor Cyan
}

function Invoke-Docker {
    param([string[]]$Arguments)
    & docker @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Invoke-Go {
    param([string]$Command)
    Invoke-Docker @(
        'run', '--rm',
        '-v', "${Root}:/repo",
        '-v', "${GoCacheVolume}:/root/.cache/go-build",
        '-v', "${GoModCacheVolume}:/go/pkg/mod",
        '-w', '/repo',
        '-e', 'CGO_ENABLED=1',
        '-e', 'GOFLAGS=-mod=mod',
        $GoImage,
        'sh', '-c', "apk add --no-cache gcc musl-dev >/dev/null && $Command"
    )
}

function Invoke-Web {
    param([string]$Command)
    # The frontend is copied into the container rather than bind-mounted.
    #
    # `npm ci` deletes and recreates node_modules, so a bind mount would write
    # that tree into the checkout — and a Go test walking ./... at the same
    # moment would trip over a directory that vanished under it. The copy keeps
    # the host tree untouched.
    Invoke-Docker @(
        'run', '--rm',
        '-v', "${Root}/web:/src:ro",
        '-v', "${NpmCacheVolume}:/root/.npm",
        '-w', '/app',
        $NodeImage,
        'sh', '-c', "cp -r /src/. /app/ && $Command"
    )
}

function Test-Go {
    Write-Step 'gofmt'
    Invoke-Go "gofmt -l \`$(find cmd internal tests -name '*.go') | tee /tmp/unformatted | (! grep . /tmp/unformatted)"

    Write-Step 'go vet'
    Invoke-Go 'go vet ./cmd/... ./internal/... ./tests/...'

    Write-Step 'go build'
    Invoke-Go 'go build ./...'

    Write-Step 'golangci-lint'
    Invoke-Docker @(
        'run', '--rm',
        '-v', "${Root}:/repo",
        '-v', "${GoCacheVolume}:/root/.cache/go-build",
        '-v', "${GoModCacheVolume}:/go/pkg/mod",
        '-w', '/repo',
        $LintImage,
        'golangci-lint', 'run'
    )

    Write-Step 'go test'
    # The package list is explicit rather than ./...: the Go module has no
    # packages under web/, and ./... walks web/node_modules when it exists —
    # which is both slow and a race with any npm run in the same tree.
    Invoke-Go 'go test -count=1 ./cmd/... ./internal/... ./tests/...'

    Write-Step 'go test -race'
    Invoke-Go 'go test -race -count=1 ./cmd/... ./internal/... ./tests/...'
}

function Test-Web {
    Write-Step 'web typecheck'
    Invoke-Web 'npm ci --no-audit --no-fund && npm run typecheck'

    Write-Step 'web test'
    Invoke-Web 'npm ci --no-audit --no-fund && npm test'
}

function Test-Integration {
    Write-Step 'test database'
    Invoke-Docker @(
        'compose', '-p', $ComposeProject, '-f', $ComposeFile, 'up', '-d', '--wait', 'postgres'
    )

    try {
        Write-Step 'migrations'
        Invoke-Docker @(
            'compose', '-p', $ComposeProject, '-f', $ComposeFile, 'run', '--rm', 'migrate'
        )

        Write-Step 'integration tests'
        Invoke-Docker @(
            'run', '--rm',
            '-v', "${Root}:/repo",
            '-v', "${GoCacheVolume}:/root/.cache/go-build",
            '-v', "${GoModCacheVolume}:/go/pkg/mod",
            '-w', '/repo',
            '-e', 'CGO_ENABLED=1',
            '-e', 'DATABASE_URL=postgres://mcp:test-only@postgres:5432/agent_shaker_test?sslmode=disable',
            '--network', "${ComposeProject}_test",
            $GoImage,
            'sh', '-c', 'apk add --no-cache gcc musl-dev >/dev/null && go test -tags=integration -count=1 ./...'
        )
    }
    finally {
        Write-Step 'tearing the test stack down'
        & docker compose -p $ComposeProject -f $ComposeFile down -v --remove-orphans 2>&1 | Out-Null
    }
}

switch ($Target) {
    'go' { Test-Go }
    'web' { Test-Web }
    'integration' { Test-Integration }
    'all' {
        Test-Go
        Test-Web
        Test-Integration
    }
}

Write-Host "`n> all checks passed" -ForegroundColor Green
