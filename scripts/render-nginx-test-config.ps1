param(
  [Parameter(Mandatory = $true)]
  [ValidateSet('healthy', 'no-health', 'invalid-config')]
  [string]$Scenario,

  [int]$Port = 18080,

  [Parameter(Mandatory = $true)]
  [string]$OutputDir
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
$repoRoot = (Get-Location).Path
$fixtureRoot = Join-Path $repoRoot 'tests\nginx'
$scenarioTemplate = Join-Path $fixtureRoot ('scenarios\' + $Scenario + '.nginx.conf.template')

if (-not (Test-Path $scenarioTemplate)) {
  throw "Scenario template not found: $scenarioTemplate"
}

New-Item -ItemType Directory -Force $OutputDir | Out-Null

foreach ($dir in @('conf', 'html', 'logs', 'temp')) {
  New-Item -ItemType Directory -Force (Join-Path $OutputDir $dir) | Out-Null
}
foreach ($dir in @('client_body_temp', 'proxy_temp', 'fastcgi_temp', 'uwsgi_temp', 'scgi_temp')) {
  New-Item -ItemType Directory -Force (Join-Path $OutputDir ('temp\' + $dir)) | Out-Null
}

Copy-Item -Path (Join-Path $fixtureRoot 'config\conf\mime.types') -Destination (Join-Path $OutputDir 'conf\mime.types') -Force
Copy-Item -Path (Join-Path $fixtureRoot 'config\html\*') -Destination (Join-Path $OutputDir 'html') -Recurse -Force

$template = Get-Content -Raw $scenarioTemplate
$rootForNginx = $OutputDir -replace '\\','/'
$config = $template.Replace('{{ROOT_DIR}}', $rootForNginx).Replace('{{PORT}}', [string]$Port)
$configPath = Join-Path $OutputDir 'conf\nginx.conf'
Set-Content -Path $configPath -Value $config -NoNewline

Write-Host "Scenario rendered: $Scenario"
Write-Host "Config: $configPath"
Write-Host "Port: $Port"
