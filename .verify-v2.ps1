$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

function Run-Step($name, [scriptblock]$block) {
    Write-Host ""
    Write-Host "=== $name ==="
    & $block
    Write-Host ("[exit={0}]" -f $LASTEXITCODE)
}

Run-Step 'go build ./...' {
    $o = & go build ./... *>&1 | Out-String
    if ($o) { Write-Host $o }
}

Run-Step 'go vet ./...' {
    $o = & go vet ./... *>&1 | Out-String
    if ($o) { Write-Host $o }
}

Run-Step 'go mod verify' {
    $o = & go mod verify *>&1 | Out-String
    Write-Host $o
}

Run-Step 'go test ./... -count=1 -timeout=120s' {
    $o = & go test ./... -count=1 -timeout=120s *>&1 | Out-String
    Write-Host $o
}

Run-Step 'gofmt -l (count + list)' {
    $o = & gofmt -l . *>&1 | Out-String
    $lines = ($o -split "`n" | Where-Object { $_ -and $_ -notmatch '^\s*$' })
    Write-Host ("Unformatted files: {0}" -f $lines.Count)
    $lines | Select-Object -First 5 | ForEach-Object { Write-Host " - $_" }
    if ($lines.Count -gt 5) { Write-Host " ... and $($lines.Count - 5) more" }
}

Run-Step 'go env (sanity)' {
    & go env GOVERSION GOMODCACHE GOPATH
}
