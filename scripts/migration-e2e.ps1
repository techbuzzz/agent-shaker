# Proves that the migrations apply to a database that already holds data.
#
# The order is the point:
#
#   1. Apply every migration but the journal.
#   2. Put rows in the database through the API.
#   3. Apply the journal migration on top of those rows.
#   4. Check the rows survived, and that the journal now records what happens.
#
# A migration that only ever runs against an empty database proves nothing
# about the case it will actually meet.

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

function RunNode {
    param([string[]]$NodeArgs, [string]$SeedFile = '')
    $args = @(
        'run', '--rm',
        '--network', "${Project}_test",
        '-v', "${Root}:/repo:ro",
        '-w', '/repo',
        '-e', 'AGENT_SHAKER_URL=http://api:8080'
    )
    if ($SeedFile -ne '') {
        # The seeded ids go in as a file, not as an environment variable.
        #
        # Windows PowerShell 5.1 strips the double quotes out of an argument
        # before it reaches a native executable, so a JSON object passed as
        # `-e "SEEDED_JSON={...}"` arrives as `{...}` with its quotes gone —
        # which is not JSON, and fails with a parse error pointing at
        # position 1.
        $args += @('-v', "${SeedFile}:/seeded.json:ro", '-e', 'AGENT_SHAKER_SEEDED_FILE=/seeded.json')
    }
    $args += @('node:22-slim', 'node', 'scripts/migration-e2e.mjs')
    $args += $NodeArgs

    $output = & docker @args
    if ($LASTEXITCODE -ne 0) {
        throw "the migration proof failed with exit code $LASTEXITCODE"
    }
    return ($output | Where-Object { $_ -match '^\{' }) -join ''
}

function Query {
    param([string]$Sql)
    $result = & docker compose -p $Project -f $ComposeFile exec -T postgres `
        psql -U mcp -d agent_shaker_test -tAc $Sql
    if ($LASTEXITCODE -ne 0) {
        throw "the query failed: $Sql"
    }
    return ($result | Where-Object { $_ -ne '' } | Select-Object -First 1)
}

try {
    $total = (Get-ChildItem (Join-Path $Root 'migrations') -Filter '*.up.sql').Count
    $steps = $total - 1
    Write-Host "> $total migrations; applying all but the last one"

    Write-Host '> building the image'
    Compose @('build', 'api', 'migrate')

    Write-Host '> starting the throwaway database'
    Compose @('up', '-d', '--wait', 'postgres')

    Write-Host "> applying $steps of $total migrations"
    # The whole command, not just the flags: `compose run` replaces the
    # service's command outright, and the image has no entrypoint to add the
    # binary name back.
    Compose @('run', '--rm', 'migrate', './migrate', '-cmd', 'up', '-dir', 'migrations', '-steps', "$steps")

    Write-Host '> starting the API'
    Compose @('up', '-d', '--wait', 'api')

    Write-Host '> writing rows through the API'
    $seeded = RunNode @('seed')
    Write-Host "seeded $seeded"

    $seedFile = Join-Path ([System.IO.Path]::GetTempPath()) "agent-shaker-seeded.json"
    # Without a BOM. `Set-Content -Encoding UTF8` writes one in Windows
    # PowerShell 5.1, and a BOM is not JSON.
    [System.IO.File]::WriteAllText($seedFile, $seeded, (New-Object System.Text.UTF8Encoding($false)))

    Write-Host '> applying the last migration, over the data'
    Compose @('run', '--rm', 'migrate')

    Write-Host '> checking the rows survived'
    $projects = Query 'SELECT count(*) FROM projects'
    $tasks = Query 'SELECT count(*) FROM tasks'
    $contexts = Query 'SELECT count(*) FROM contexts'
    Write-Host "  projects=$projects tasks=$tasks contexts=$contexts"
    if ([int]$projects -lt 1 -or [int]$tasks -lt 1 -or [int]$contexts -lt 1) {
        throw 'the rows written before the migration are gone'
    }

    Write-Host '> restarting the API so it sees the journal'
    Compose @('restart', 'api')
    Compose @('up', '-d', '--wait', 'api')

    Write-Host '> checking the journal'
    RunNode @('verify') $seedFile | Out-Null
    Remove-Item -LiteralPath $seedFile -Force -ErrorAction SilentlyContinue

    $entries = Query 'SELECT count(*) FROM audit_events'
    Write-Host "  audit_events=$entries"
    if ([int]$entries -lt 1) {
        throw 'the journal table is empty'
    }
}
finally {
    Write-Host '> tearing the test stack down'
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    & docker compose -p $Project -f $ComposeFile down -v --remove-orphans 2>&1 | Out-Null
    $ErrorActionPreference = $previous
}

Write-Host "`n> the migrations apply to a database that already holds data" -ForegroundColor Green
