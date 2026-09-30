$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

$out = & go build ./... *>&1 | Out-String
if ($out) { Write-Host $out } else { Write-Host "(clean)" }
Write-Host ("EXIT={0}" -f $LASTEXITCODE)
