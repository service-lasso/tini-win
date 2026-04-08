$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..

New-Item -ItemType Directory -Force .\bin\testapps | Out-Null

$targets = @(
  @{ Path = '.\testapps\simple-exit'; Out = '.\bin\testapps\simple-exit.exe' },
  @{ Path = '.\testapps\spawn-child'; Out = '.\bin\testapps\spawn-child.exe' },
  @{ Path = '.\testapps\ignore-stop'; Out = '.\bin\testapps\ignore-stop.exe' },
  @{ Path = '.\testapps\graceful-stop'; Out = '.\bin\testapps\graceful-stop.exe' }
)

foreach ($t in $targets) {
  Write-Host "Building $($t.Path) -> $($t.Out)"
  go build -o $t.Out $t.Path
}

Write-Host "Built test apps under .\bin\testapps"
