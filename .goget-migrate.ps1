$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker

$out = & go get github.com/golang-migrate/migrate/v4/source/file github.com/golang-migrate/migrate/v4/database/postgres github.com/golang-migrate/migrate/v4/source/iofs *>&1 | Out-String
Write-Host $out
Write-Host ("EXIT={0}" -f $LASTEXITCODE)
