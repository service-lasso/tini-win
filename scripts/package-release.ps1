param(
  [string]$Version = 'dev',
  [string]$Platform = 'windows-amd64',
  [string]$OutputDir = '.\dist'
)

$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

New-Item -ItemType Directory -Force $OutputDir | Out-Null
New-Item -ItemType Directory -Force .\bin | Out-Null

$binaryName = 'tini-win.exe'
$binaryPath = Join-Path '.\bin' $binaryName
$archiveBase = "tini-win-$Version-$Platform"
$stageDir = Join-Path $OutputDir $archiveBase
$zipPath = Join-Path $OutputDir ($archiveBase + '.zip')
$checksumPath = Join-Path $OutputDir ($archiveBase + '.sha256')

if (Test-Path $stageDir) { Remove-Item $stageDir -Recurse -Force }
if (Test-Path $zipPath) { Remove-Item $zipPath -Force }
if (Test-Path $checksumPath) { Remove-Item $checksumPath -Force }

Write-Host "Building $binaryName for $Platform..."
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -trimpath -ldflags="-s -w" -o $binaryPath .\cmd\tini-win
if ($LASTEXITCODE -ne 0) { throw "go build failed with code $LASTEXITCODE" }

New-Item -ItemType Directory -Force $stageDir | Out-Null
Copy-Item $binaryPath (Join-Path $stageDir $binaryName) -Force
Copy-Item .\README.md (Join-Path $stageDir 'README.md') -Force
Copy-Item .\LICENSE (Join-Path $stageDir 'LICENSE') -Force

Compress-Archive -Path (Join-Path $stageDir '*') -DestinationPath $zipPath -Force
$hash = (Get-FileHash $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
Set-Content -Path $checksumPath -Value ($hash + '  ' + [System.IO.Path]::GetFileName($zipPath)) -NoNewline

Write-Host "Built release package: $zipPath"
Write-Host "Checksum: $checksumPath"
