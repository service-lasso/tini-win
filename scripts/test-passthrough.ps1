$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

.\scripts\build.ps1 | Out-Null
.\scripts\build-testapps.ps1 | Out-Null

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tini-win-passthrough-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force $tempDir | Out-Null

$stdoutFile = Join-Path $tempDir 'stdout.txt'
$stderrFile = Join-Path $tempDir 'stderr.txt'
$pidFile = Join-Path $tempDir 'stdout-stderr.pid'

$proc = Start-Process -FilePath .\bin\tini-win.exe -ArgumentList @(
  '--',
  '.\bin\testapps\stdout-stderr.exe',
  '--pid-file', $pidFile,
  '--stdout', 'alpha-out',
  '--stderr', 'beta-err'
) -RedirectStandardOutput $stdoutFile -RedirectStandardError $stderrFile -PassThru -Wait

if ($proc.ExitCode -ne 0) {
  throw "stdout/stderr passthrough case failed with code $($proc.ExitCode)"
}

$stdoutText = Get-Content -Raw $stdoutFile
$stderrText = Get-Content -Raw $stderrFile

if ($stdoutText -notmatch 'alpha-out') {
  throw "expected stdout passthrough to contain alpha-out, got: $stdoutText"
}
if ($stderrText -notmatch 'beta-err') {
  throw "expected stderr passthrough to contain beta-err, got: $stderrText"
}

Write-Host 'stdout/stderr passthrough proof passed'
