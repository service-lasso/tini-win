$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

New-Item -ItemType Directory -Force .\bin\testapps | Out-Null

$targets = @(
  @{ Path = '.\testapps\simple-exit'; Out = '.\bin\testapps\simple-exit.exe' },
  @{ Path = '.\testapps\spawn-child'; Out = '.\bin\testapps\spawn-child.exe' },
  @{ Path = '.\testapps\ignore-stop'; Out = '.\bin\testapps\ignore-stop.exe' },
  @{ Path = '.\testapps\graceful-stop'; Out = '.\bin\testapps\graceful-stop.exe' },
  @{ Path = '.\testapps\breakaway-child'; Out = '.\bin\testapps\breakaway-child.exe' },
  @{ Path = '.\testapps\relaunch-orphan'; Out = '.\bin\testapps\relaunch-orphan.exe' },
  @{ Path = '.\testapps\brokered-child'; Out = '.\bin\testapps\brokered-child.exe' },
  @{ Path = '.\testapps\port-rebind-server'; Out = '.\bin\testapps\port-rebind-server.exe' },
  @{ Path = '.\testapps\stdio-hold-open'; Out = '.\bin\testapps\stdio-hold-open.exe' },
  @{ Path = '.\testapps\console-trap'; Out = '.\bin\testapps\console-trap.exe' }
)

foreach ($t in $targets) {
  Write-Host "Building $($t.Path) -> $($t.Out)"
  go build -o $t.Out $t.Path
  if ($LASTEXITCODE -ne 0) { throw "build failed for $($t.Path) with code $LASTEXITCODE" }
}

Write-Host "Built test apps under .\bin\testapps"
