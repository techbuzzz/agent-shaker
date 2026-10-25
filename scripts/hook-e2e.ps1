# End-to-end proof of the hook layer.
#
# The Windows counterpart of scripts/hook-e2e.sh. It runs the real server
# against a throwaway database, binds a hook to a task lifecycle event,
# completes the task, and checks that the run log shows the command's own
# captured output.
#
# Everything happens in containers, in the `agent-shaker-test` compose project,
# so nothing here can reach a developer's running stack.

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

    Write-Host '> starting the API with the hook layer enabled'
    Compose @('up', '-d', '--wait', 'api')

    Write-Host '> running the proof'
    & docker run --rm `
        --network "${Project}_test" `
        -v "${Root}/scripts:/scripts:ro" `
        -e AGENT_SHAKER_URL=http://api:8080 `
        -e AGENT_SHAKER_API_KEY=e2e-only `
        node:22-slim node /scripts/hook-e2e.mjs
    if ($LASTEXITCODE -ne 0) {
        throw "the proof failed with exit code $LASTEXITCODE"
    }
}
finally {
    Write-Host '> tearing the test stack down'
    # docker compose writes its progress to stderr, and Stop would turn that
    # into a terminating error — so the teardown runs with the policy relaxed.
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & docker compose -p $Project -f $ComposeFile down -v --remove-orphans 2>&1 | Out-Null
    $ErrorActionPreference = $previous
}

Write-Host "`n> the hook layer is proven end to end" -ForegroundColor Green
