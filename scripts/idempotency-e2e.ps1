# Proves the idempotency keys on the mutating MCP tools.
#
# The proof is behavioural: a retry with the same key replays the first
# answer, and the task is claimed exactly once. The same key on a different
# task executes for real, because the key identifies one logical operation and
# the request's fingerprint says which one.

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$ComposeFile = Join-Path $Root 'docker-compose.test.yml'
$Project = 'agent-shaker-test'

function Compose {
    param([string[]]$Arguments)
    & docker compose -p $Project -f $ComposeFile @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

try {
    Write-Host '> building the image'
    Compose @('build', 'api', 'migrate')

    Write-Host '> starting the throwaway database'
    Compose @('up', '-d', '--wait', 'postgres')

    Write-Host '> applying the migrations'
    Compose @('run', '--rm', 'migrate')

    Write-Host '> starting the API'
    Compose @('up', '-d', '--wait', 'api')

    Write-Host '> running the proof'
    & docker run --rm `
        --network "${Project}_test" `
        -v "${Root}:/repo:ro" `
        -w /repo `
        -e AGENT_SHAKER_URL=http://api:8080 `
        node:22-slim node scripts/idempotency-e2e.mjs
    if ($LASTEXITCODE -ne 0) {
        throw "the idempotency proof failed with exit code $LASTEXITCODE"
    }
}
finally {
    Write-Host '> tearing the test stack down'
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & docker compose -p $Project -f $ComposeFile down -v --remove-orphans 2>&1 | Out-Null
    $ErrorActionPreference = $previous
}

Write-Host "`n> a retry does not claim a task twice" -ForegroundColor Green
