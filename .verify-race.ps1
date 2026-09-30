$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker
$env:CGO_ENABLED = '1'

$out = & go test ./... -race -count=1 -timeout=180s *>&1 | Out-String
Write-Host $out
Write-Host ("RACE EXIT: {0}" -f $LASTEXITCODE)
