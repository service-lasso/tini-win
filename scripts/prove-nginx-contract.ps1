param(
  [int]$StartPort = 18280,
  [int]$Iterations = 5,
  [string]$OutputDir = ''
)

$ErrorActionPreference = 'Stop'
Set-Location (Join-Path $PSScriptRoot '..')
$repoRoot = (Get-Location).Path
$tiniWinExe = Join-Path $repoRoot 'bin\tini-win.exe'
$localNginxExe = Join-Path $repoRoot 'tests\nginx\win32\nginx.exe'

if ([string]::IsNullOrWhiteSpace($OutputDir)) {
  $OutputDir = Join-Path $repoRoot 'artifacts\proof\nginx-contract'
}

New-Item -ItemType Directory -Force $OutputDir | Out-Null
$summaryPath = Join-Path $OutputDir 'nginx-contract.json'
$results = New-Object System.Collections.Generic.List[object]

function ConvertTo-IsoTime {
  param([datetime]$Value)
  $Value.ToUniversalTime().ToString('o')
}

function Write-ProofSummary {
  $summary = [ordered]@{
    generatedAt = ConvertTo-IsoTime (Get-Date)
    repoRoot = $repoRoot
    tiniWinExe = $tiniWinExe
    nginxExe = $localNginxExe
    startPort = $StartPort
    iterations = $Iterations
    total = $results.Count
    passed = @($results | Where-Object { $_.status -eq 'passed' }).Count
    failed = @($results | Where-Object { $_.status -eq 'failed' }).Count
    results = @($results)
  }
  $summary | ConvertTo-Json -Depth 12 | Set-Content -Path $summaryPath -Encoding utf8
}

function Invoke-ProofCase {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][scriptblock]$Body
  )

  Write-Host ""
  Write-Host "== $Name =="
  $started = Get-Date
  $status = 'passed'
  $errorMessage = $null
  $details = $null

  try {
    $details = & $Body
    Write-Host "PASS: $Name"
  } catch {
    $status = 'failed'
    $errorMessage = $_.Exception.Message
    Write-Host "FAIL: $Name"
    Write-Host $errorMessage
  }

  $ended = Get-Date
  $result = [ordered]@{
    name = $Name
    status = $status
    startedAt = ConvertTo-IsoTime $started
    endedAt = ConvertTo-IsoTime $ended
    durationMs = [int][math]::Round(($ended - $started).TotalMilliseconds)
    details = $details
    error = $errorMessage
  }
  $results.Add([pscustomobject]$result) | Out-Null
  Write-ProofSummary

  if ($status -ne 'passed') {
    throw $errorMessage
  }
}

function Wait-ForFile {
  param([Parameter(Mandatory = $true)][string]$Path, [int]$TimeoutSeconds = 15)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (Test-Path $Path) { return }
    Start-Sleep -Milliseconds 150
  }
  throw "Timed out waiting for file: $Path"
}

function Read-PidFile {
  param([Parameter(Mandatory = $true)][string]$Path)
  [int](Get-Content -Raw $Path).Trim()
}

function Wait-ForProcessGone {
  param([Parameter(Mandatory = $true)][int]$TargetPid, [int]$TimeoutSeconds = 15)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (-not (Get-Process -Id $TargetPid -ErrorAction SilentlyContinue)) { return }
    Start-Sleep -Milliseconds 200
  }
  throw "Timed out waiting for process $TargetPid to exit"
}

function Wait-ForHttpStatus {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [Parameter(Mandatory = $true)][int]$ExpectedStatus,
    [int]$TimeoutSeconds = 15
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    try {
      $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 2
      if ([int]$response.StatusCode -eq $ExpectedStatus) { return $response }
    } catch {
      if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq $ExpectedStatus) {
        return $_.Exception.Response
      }
    }
    Start-Sleep -Milliseconds 250
  }
  throw "Timed out waiting for HTTP $ExpectedStatus from $Url"
}

function Test-TcpConnect {
  param([int]$Port, [int]$TimeoutMilliseconds = 500)
  $client = [System.Net.Sockets.TcpClient]::new()
  try {
    $async = $client.BeginConnect('127.0.0.1', $Port, $null, $null)
    if (-not $async.AsyncWaitHandle.WaitOne($TimeoutMilliseconds)) { return $false }
    $client.EndConnect($async)
    return $true
  } catch {
    return $false
  } finally {
    if ($async -and $async.AsyncWaitHandle) { $async.AsyncWaitHandle.Dispose() }
    $client.Dispose()
  }
}

function Wait-ForPortClosed {
  param([int]$Port, [int]$TimeoutSeconds = 10)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (-not (Test-TcpConnect -Port $Port)) { return }
    Start-Sleep -Milliseconds 250
  }
  throw "Timed out waiting for TCP port $Port to close"
}

function Render-NginxScenario {
  param([string]$Scenario, [int]$Port, [string]$InstanceDir)
  pwsh -NoLogo -NoProfile -File .\scripts\render-nginx-test-config.ps1 -Scenario $Scenario -Port $Port -OutputDir $InstanceDir | Out-Null
}

function Get-NginxPidsForInstance {
  param([string]$ExePath, [string]$InstanceDir)
  $normalizedExe = [System.IO.Path]::GetFullPath($ExePath)
  $normalizedInstance = [System.IO.Path]::GetFullPath($InstanceDir)
  $procs = Get-CimInstance Win32_Process -Filter "Name = 'nginx.exe'" -ErrorAction SilentlyContinue
  @($procs | Where-Object {
    $_.ExecutablePath -and $_.CommandLine -and
    ([System.IO.Path]::GetFullPath($_.ExecutablePath) -ieq $normalizedExe) -and
    ($_.CommandLine -like ('*' + $normalizedInstance + '*'))
  } | Select-Object -ExpandProperty ProcessId)
}

function Stop-RemainingNginxForInstance {
  param([string]$InstanceDir)
  $remaining = @(Get-NginxPidsForInstance -ExePath $localNginxExe -InstanceDir $InstanceDir)
  foreach ($pid in $remaining) {
    taskkill /PID $pid /T /F | Out-Null
    try { Wait-ForProcessGone -TargetPid ([int]$pid) -TimeoutSeconds 5 } catch { }
  }
}

function Assert-NoNginxForInstance {
  param([string]$InstanceDir, [int]$TimeoutSeconds = 10)
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    $remaining = @(Get-NginxPidsForInstance -ExePath $localNginxExe -InstanceDir $InstanceDir)
    if ($remaining.Count -eq 0) { return @() }
    Start-Sleep -Milliseconds 250
  } while ((Get-Date) -lt $deadline)
  throw "expected no nginx.exe processes for instance $InstanceDir after wrapper teardown, remaining=$($remaining -join ',')"
}

function Start-NginxWithTini {
  param([string]$InstanceDir, [string]$ConfPath)
  $argsLine = '--stop-timeout 500ms --remap-exit 137:0 -- "' + $localNginxExe + '" -p "' + $InstanceDir + '" -c "' + $ConfPath + '"'
  Start-Process -FilePath $tiniWinExe -WorkingDirectory $repoRoot -ArgumentList $argsLine -PassThru
}

function Stop-WrapperAndAssertNginxCleanup {
  param(
    [Parameter(Mandatory = $true)]$WrapperProcess,
    [Parameter(Mandatory = $true)][string]$InstanceDir,
    [Parameter(Mandatory = $true)][int]$MasterPid,
    [Parameter(Mandatory = $true)][int]$Port,
    [int[]]$KnownNginxPids = @()
  )

  if (Get-Process -Id $WrapperProcess.Id -ErrorAction SilentlyContinue) {
    Stop-Process -Id $WrapperProcess.Id
  }

  Wait-ForProcessGone -TargetPid $WrapperProcess.Id -TimeoutSeconds 15
  Wait-ForProcessGone -TargetPid $MasterPid -TimeoutSeconds 15
  foreach ($pid in $KnownNginxPids) {
    Wait-ForProcessGone -TargetPid ([int]$pid) -TimeoutSeconds 15
  }

  $remaining = Assert-NoNginxForInstance -InstanceDir $InstanceDir -TimeoutSeconds 10
  Wait-ForPortClosed -Port $Port -TimeoutSeconds 10

  [pscustomobject]@{
    wrapperPid = $WrapperProcess.Id
    masterPid = $MasterPid
    knownNginxPids = @($KnownNginxPids)
    remainingNginxPids = @($remaining)
    portClosed = $true
  }
}

function Invoke-NginxStartHealthForcedStop {
  param([string]$Scenario, [int]$Port, [string]$CaseName)
  $instanceDir = Join-Path ([System.IO.Path]::GetTempPath()) ('tini-win-nginx-' + [guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Force $instanceDir | Out-Null
  Render-NginxScenario -Scenario $Scenario -Port $Port -InstanceDir $instanceDir
  $pidFile = Join-Path $instanceDir 'logs\nginx.pid'
  $conf = Join-Path $instanceDir 'conf\nginx.conf'
  $proc = $null

  try {
    Wait-ForPortClosed -Port $Port -TimeoutSeconds 2
    $proc = Start-NginxWithTini -InstanceDir $instanceDir -ConfPath $conf
    Wait-ForFile -Path $pidFile -TimeoutSeconds 15
    $masterPid = Read-PidFile $pidFile
    if (-not (Get-Process -Id $masterPid -ErrorAction SilentlyContinue)) {
      throw "nginx master pid $masterPid was not observed running"
    }

    $url = if ($Scenario -eq 'healthy') { "http://127.0.0.1:$Port/health" } else { "http://127.0.0.1:$Port/" }
    $response = Wait-ForHttpStatus -Url $url -ExpectedStatus 200 -TimeoutSeconds 15
    if ($Scenario -eq 'healthy' -and [string]$response.Content -notmatch 'ok') {
      throw 'nginx healthy /health response did not contain ok'
    }

    $nginxPidsBefore = @(Get-NginxPidsForInstance -ExePath $localNginxExe -InstanceDir $instanceDir)
    if ($nginxPidsBefore.Count -lt 2) {
      throw "expected nginx master+worker processes before forced kill, got $($nginxPidsBefore -join ',')"
    }

    $cleanup = Stop-WrapperAndAssertNginxCleanup -WrapperProcess $proc -InstanceDir $instanceDir -MasterPid $masterPid -Port $Port -KnownNginxPids $nginxPidsBefore

    [pscustomobject]@{
      caseName = $CaseName
      scenario = $Scenario
      port = $Port
      healthUrl = $url
      instanceDir = $instanceDir
      wrapperPid = $proc.Id
      masterPid = $masterPid
      nginxPidsBefore = @($nginxPidsBefore)
      cleanup = $cleanup
    }
  } finally {
    if ($proc -and (Get-Process -Id $proc.Id -ErrorAction SilentlyContinue)) {
      Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
    }
    Stop-RemainingNginxForInstance -InstanceDir $instanceDir
  }
}

function Invoke-NginxInvalidConfig {
  param([int]$Port)
  $instanceDir = Join-Path ([System.IO.Path]::GetTempPath()) ('tini-win-nginx-invalid-' + [guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Force $instanceDir | Out-Null
  Render-NginxScenario -Scenario 'invalid-config' -Port $Port -InstanceDir $instanceDir
  $conf = Join-Path $instanceDir 'conf\nginx.conf'

  & $tiniWinExe -- $localNginxExe -p $instanceDir -c $conf
  $code = $LASTEXITCODE
  if ($code -eq 0) { throw 'nginx invalid-config unexpectedly exited 0' }

  [pscustomobject]@{
    scenario = 'invalid-config'
    port = $Port
    instanceDir = $instanceDir
    exitCode = $code
  }
}

function Invoke-NginxPortRebindLoop {
  param([int]$Port, [int]$Count)
  $instanceDir = Join-Path ([System.IO.Path]::GetTempPath()) ('tini-win-nginx-rebind-' + [guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Force $instanceDir | Out-Null
  Render-NginxScenario -Scenario 'healthy' -Port $Port -InstanceDir $instanceDir
  $pidFile = Join-Path $instanceDir 'logs\nginx.pid'
  $conf = Join-Path $instanceDir 'conf\nginx.conf'
  $runs = New-Object System.Collections.Generic.List[object]

  try {
    for ($i = 1; $i -le $Count; $i++) {
      Remove-Item -Path $pidFile -Force -ErrorAction SilentlyContinue
      Wait-ForPortClosed -Port $Port -TimeoutSeconds 10
      $proc = Start-NginxWithTini -InstanceDir $instanceDir -ConfPath $conf
      try {
        Wait-ForFile -Path $pidFile -TimeoutSeconds 15
        $masterPid = Read-PidFile $pidFile
        $response = Wait-ForHttpStatus -Url "http://127.0.0.1:$Port/health" -ExpectedStatus 200 -TimeoutSeconds 15
        if ([string]$response.Content -notmatch 'ok') {
          throw 'nginx rebind /health response did not contain ok'
        }
        $nginxPidsBefore = @(Get-NginxPidsForInstance -ExePath $localNginxExe -InstanceDir $instanceDir)
        $cleanup = Stop-WrapperAndAssertNginxCleanup -WrapperProcess $proc -InstanceDir $instanceDir -MasterPid $masterPid -Port $Port -KnownNginxPids $nginxPidsBefore
        $runs.Add([pscustomobject]@{
          iteration = $i
          wrapperPid = $proc.Id
          masterPid = $masterPid
          nginxPidsBefore = @($nginxPidsBefore)
          cleanup = $cleanup
        }) | Out-Null
      } finally {
        if ($proc -and (Get-Process -Id $proc.Id -ErrorAction SilentlyContinue)) {
          Stop-Process -Id $proc.Id -ErrorAction SilentlyContinue
        }
        Stop-RemainingNginxForInstance -InstanceDir $instanceDir
      }
    }

    [pscustomobject]@{
      scenario = 'restart-port-rebind'
      port = $Port
      iterations = $Count
      instanceDir = $instanceDir
      runs = @($runs)
    }
  } finally {
    Stop-RemainingNginxForInstance -InstanceDir $instanceDir
  }
}

if (-not (Test-Path $tiniWinExe)) {
  .\scripts\build.ps1
}

if (-not (Test-Path $tiniWinExe)) {
  throw "tini-win.exe missing after build: $tiniWinExe"
}

if (-not (Test-Path $localNginxExe)) {
  throw "local nginx fixture missing: $localNginxExe"
}

Write-Host "tini-win: $tiniWinExe"
Write-Host "nginx fixture: $localNginxExe"
Write-Host "proof output: $OutputDir"

Invoke-ProofCase -Name 'nginx healthy starts through tini-win and forced teardown cleans process tree' -Body {
  Invoke-NginxStartHealthForcedStop -Scenario 'healthy' -Port $StartPort -CaseName 'healthy-forced-teardown'
}

Invoke-ProofCase -Name 'nginx no-health starts through tini-win and forced teardown cleans process tree' -Body {
  Invoke-NginxStartHealthForcedStop -Scenario 'no-health' -Port ($StartPort + 1) -CaseName 'no-health-forced-teardown'
}

Invoke-ProofCase -Name 'nginx invalid config exits non-zero through tini-win' -Body {
  Invoke-NginxInvalidConfig -Port ($StartPort + 2)
}

Invoke-ProofCase -Name 'nginx restarts repeatedly on same port after wrapper teardown' -Body {
  Invoke-NginxPortRebindLoop -Port ($StartPort + 3) -Count $Iterations
}

Write-Host ""
Write-Host "nginx contract proof passed"
Write-Host "summary: $summaryPath"
Write-ProofSummary
