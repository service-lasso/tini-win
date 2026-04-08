$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

Write-Host "Running go test..."
go test .\...

Write-Host "Running Windows integration tests for runner package..."
go test .\internal\runner -run TestRunContext -v
