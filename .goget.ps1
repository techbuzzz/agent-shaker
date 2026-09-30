$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

$out = & go get github.com/prometheus/client_golang/prometheus github.com/prometheus/client_golang/prometheus/promhttp *>&1 | Out-String
Write-Host $out
Write-Host ("EXIT={0}" -f $LASTEXITCODE)
