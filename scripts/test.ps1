$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

Write-Host "Running go test..."
go test .\...
if ($LASTEXITCODE -ne 0) { throw "go test failed with code $LASTEXITCODE" }

Write-Host "Running Windows integration tests for runner package..."
go test .\internal\runner -run TestRunContext -v
if ($LASTEXITCODE -ne 0) { throw "runner integration tests failed with code $LASTEXITCODE" }

Write-Host "Running stdout/stderr passthrough proof..."
.\scripts\test-passthrough.ps1
if ($LASTEXITCODE -ne 0) { throw "test-passthrough failed with code $LASTEXITCODE" }

Write-Host "Running full proof flow..."
.\scripts\prove-mvp.ps1
if ($LASTEXITCODE -ne 0) { throw "prove-mvp failed with code $LASTEXITCODE" }
