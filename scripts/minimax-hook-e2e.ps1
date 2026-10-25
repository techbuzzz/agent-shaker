# Proves the MiniMax Code → Agent Shaker bridge.
#
# It brings up the throwaway stack, then drives the hook script the way
# MiniMax Code would: the payload on stdin, the environment in place. The
# script is ours, so proving it is proving the part we own — it reads the
# payload, resolves the agent, renews the lease and pushes the compacted
# transcript.
#
# The hook itself stays inert until an operator enables it in
# Settings → Hooks → Project; proving that needs the host.

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

    Write-Host '> running the bridge proof'
    & docker run --rm `
        --network "${Project}_test" `
        -v "${Root}:/repo:ro" `
        -w /repo `
        -e AGENT_SHAKER_URL=http://api:8080 `
        node:22-slim node scripts/minimax-hook-e2e.mjs
    if ($LASTEXITCODE -ne 0) {
        throw "the bridge proof failed with exit code $LASTEXITCODE"
    }
}
finally {
    Write-Host '> tearing the test stack down'
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & docker compose -p $Project -f $ComposeFile down -v --remove-orphans 2>&1 | Out-Null
    $ErrorActionPreference = $previous
}

Write-Host "`n> the MiniMax Code hook bridge is proven" -ForegroundColor Green
