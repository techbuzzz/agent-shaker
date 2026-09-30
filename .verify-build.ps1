$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker
$env:GOPROXY = 'https://proxy.golang.org,direct'

# Run go build, capturing all output, but never let stderr cause the script to abort
$out = & go build ./... *>&1 | Out-String
$ec = $LASTEXITCODE

Write-Host "----- BUILD OUTPUT -----"
Write-Host $out
Write-Host "----- BUILD EXIT CODE: $ec -----"
exit $ec
