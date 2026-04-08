$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

New-Item -ItemType Directory -Force .\bin | Out-Null

Write-Host "Building tini-win..."
go build -o .\bin\tini-win.exe .\cmd\tini-win

Write-Host "Built .\bin\tini-win.exe"
