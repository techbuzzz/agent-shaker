$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

$out = & go mod why github.com/gorilla/mux *>&1 | Out-String
Write-Host $out
