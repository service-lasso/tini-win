$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

New-Item -ItemType Directory -Force .\bin\samples\go | Out-Null
New-Item -ItemType Directory -Force .\bin\samples\java\classes | Out-Null
New-Item -ItemType Directory -Force .\bin\samples\java | Out-Null

Write-Host "Building Go sample project..."
go build -o .\bin\samples\go\edgecase-go.exe .\samples\go\edgecase-app
if ($LASTEXITCODE -ne 0) { throw "go sample build failed with code $LASTEXITCODE" }

Write-Host "Building Java sample project..."
if (-not (Get-Command javac -ErrorAction SilentlyContinue)) {
  throw "javac not found"
}
if (-not (Get-Command jar -ErrorAction SilentlyContinue)) {
  throw "jar not found"
}

Remove-Item .\bin\samples\java\classes\* -Recurse -Force -ErrorAction SilentlyContinue
javac -d .\bin\samples\java\classes .\samples\java\edgecase-app\src\com\servicelasso\tiniwin\EdgeCaseApp.java
if ($LASTEXITCODE -ne 0) { throw "javac sample build failed with code $LASTEXITCODE" }
jar --create --file .\bin\samples\java\edgecase-app.jar -C .\bin\samples\java\classes .
if ($LASTEXITCODE -ne 0) { throw "jar packaging failed with code $LASTEXITCODE" }

$cmd = @"
@echo off
setlocal
java -cp "%~dp0edgecase-app.jar" com.servicelasso.tiniwin.EdgeCaseApp %*
exit /b %ERRORLEVEL%
"@
Set-Content -Path .\bin\samples\java\edgecase-app.cmd -Value $cmd -NoNewline

$childWrapper = @"
@echo off
setlocal
set DURATION=%~1
if "%DURATION%"=="" set DURATION=30
ping 127.0.0.1 -n %DURATION% >nul
exit /b 0
"@
Set-Content -Path .\bin\samples\java\child-wrapper.cmd -Value $childWrapper -NoNewline

Write-Host "Built sample projects under .\bin\samples"
