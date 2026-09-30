$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

function Run-Step($name, [scriptblock]$block) {
    Write-Host ""
    Write-Host "=== $name ==="
    & $block
    Write-Host ("[exit={0}]" -f $LASTEXITCODE)
}

Run-Step 'gofmt -l .' {
    $o = & gofmt -l . *>&1 | Out-String
    if ($o) { Write-Host $o } else { Write-Host "(empty - all Go files formatted)" }
}

Run-Step 'go build ./...' {
    $o = & go build ./... *>&1 | Out-String
    if ($o) { Write-Host $o } else { Write-Host "(clean)" }
}

Run-Step 'go vet ./...' {
    $o = & go vet ./... *>&1 | Out-String
    if ($o) { Write-Host $o } else { Write-Host "(clean)" }
}

Run-Step 'go test ./... -count=1 -timeout=120s' {
    $o = & go test ./... -count=1 -timeout=120s *>&1 | Out-String
    Write-Host $o
}
