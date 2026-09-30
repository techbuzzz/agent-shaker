$ErrorActionPreference = 'Continue'
$env:Path = "C:\Program Files\Go\bin;$env:Path"
Set-Location D:\Sources\Github\agent-shaker
New-Item -ItemType Directory -Force -Path bin | Out-Null

$out = & go build -o bin/mcp-server.exe ./cmd/server *>&1 | Out-String
Write-Host $out
Write-Host ("EXIT={0}" -f $LASTEXITCODE)
if ($LASTEXITCODE -eq 0 -and (Test-Path bin/mcp-server.exe)) {
    $size = (Get-Item bin/mcp-server.exe).Length
    Write-Host "BUILD OK: $size bytes"
}
