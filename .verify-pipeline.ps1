$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

function Invoke-Step($name, [scriptblock]$block) {
    Write-Host ""
    Write-Host "================== $name =================="
    & $block
    Write-Host ("----- {0} EXIT CODE: {1} -----" -f $name, $LASTEXITCODE)
}

Invoke-Step 'go build ./...' {
    $out = & go build ./... *>&1 | Out-String
    Write-Host $out
}

Invoke-Step 'go vet ./...' {
    $out = & go vet ./... *>&1 | Out-String
    Write-Host $out
}

Invoke-Step 'gofmt -l (unformatted)' {
    $out = & gofmt -l . *>&1 | Out-String
    Write-Host $out
}

Invoke-Step 'go test ./... -count=1 -timeout=120s' {
    $out = & go test ./... -count=1 -timeout=120s *>&1 | Out-String
    Write-Host $out
}

Invoke-Step 'go mod verify' {
    $out = & go mod verify *>&1 | Out-String
    Write-Host $out
}
