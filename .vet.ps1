$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

$out = & go vet ./... *>&1 | Out-String
Write-Host $out
Write-Host ("EXIT={0}" -f $LASTEXITCODE)
