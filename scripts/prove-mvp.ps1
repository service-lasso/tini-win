$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot\..
$repoRoot = (Get-Location).Path
$tiniWinExe = Join-Path $repoRoot 'bin\tini-win.exe'
$testAppsDir = Join-Path $repoRoot 'bin\testapps'
$goSampleExe = Join-Path $repoRoot 'bin\samples\go\edgecase-go.exe'
$javaSampleCmd = Join-Path $repoRoot 'bin\samples\java\edgecase-app.cmd'
$localNginxExe = Join-Path $repoRoot 'tests\nginx\win32\nginx.exe'

function Wait-ForFile {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [int]$TimeoutSeconds = 10
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (Test-Path $Path) { return }
    Start-Sleep -Milliseconds 150
  }
  throw "Timed out waiting for file: $Path"
}

function Wait-ForProcessGone {
  param(
    [Parameter(Mandatory = $true)][int]$TargetPid,
    [int]$TimeoutSeconds = 10
  )
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    if (-not (Get-Process -Id $TargetPid -ErrorAction SilentlyContinue)) { return }
    Start-Sleep -Milliseconds 200
  }
  throw "Timed out waiting for process $TargetPid to exit"
}

function Read-PidFile {
  param([Parameter(Mandatory = $true)][string]$Path)
  [int](Get-Content -Raw $Path).Trim()
}

function Assert-ProcessExists {
  param(
    [Parameter(Mandatory = $true)][int]$TargetPid,
    [Parameter(Mandatory = $true)][string]$Message
  )
  if (-not (Get-Process -Id $TargetPid -ErrorAction SilentlyContinue)) {
    throw $Message
  }
}

function Quote-CommandPath {
  param([Parameter(Mandatory = $true)][string]$Path)
  '"' + $Path + '"'
}

function Start-TiniWrapped {
  param([Parameter(Mandatory = $true)][string]$ArgsLine)
  Start-Process -FilePath $tiniWinExe -WorkingDirectory $repoRoot -ArgumentList $ArgsLine -PassThru
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
      if ([int]$response.StatusCode -eq $ExpectedStatus) {
        return $response
      }
    } catch {
      if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq $ExpectedStatus) {
        return $_.Exception.Response
      }
    }
    Start-Sleep -Milliseconds 250
  }
  throw "Timed out waiting for HTTP $ExpectedStatus from $Url"
}

function Render-NginxScenario {
  param(
    [Parameter(Mandatory = $true)][string]$Scenario,
    [Parameter(Mandatory = $true)][int]$Port,
    [Parameter(Mandatory = $true)][string]$OutputDir
  )
  pwsh -NoLogo -NoProfile -File .\scripts\render-nginx-test-config.ps1 -Scenario $Scenario -Port $Port -OutputDir $OutputDir | Out-Null
}

function Invoke-GracefulProof {
  param(
    [Parameter(Mandatory = $true)][string]$Label,
    [Parameter(Mandatory = $true)][string]$AppCommand,
    [Parameter(Mandatory = $true)][string]$SignalCommand,
    [Parameter(Mandatory = $true)][string]$PidFile,
    [Parameter(Mandatory = $true)][string]$SignalFile,
    [string]$RunArgs = ''
  )

  $argsLine = '--stop-timeout 2s -- ' + $AppCommand + ' ' + $RunArgs + ' --signal-file "' + $SignalFile + '" --pid-file "' + $PidFile + '"'
  $proc = Start-TiniWrapped -ArgsLine $argsLine.Trim()
  Wait-ForFile -Path $PidFile
  $targetProcPid = Read-PidFile $PidFile
  Assert-ProcessExists -TargetPid $targetProcPid -Message "$Label pid $targetProcPid was not observed running"
  Invoke-Expression $SignalCommand | Out-Null
  Wait-ForFile -Path $SignalFile
  Wait-ForProcessGone -TargetPid $targetProcPid
  Wait-Process -Id $proc.Id -ErrorAction SilentlyContinue
  Write-Host "$Label pid=$targetProcPid exited cleanly after signal"
}

function Invoke-SpawnCleanupProof {
  param(
    [Parameter(Mandatory = $true)][string]$Label,
    [Parameter(Mandatory = $true)][string]$AppCommand,
    [Parameter(Mandatory = $true)][string]$ParentPidFile,
    [Parameter(Mandatory = $true)][string]$ChildPidFile,
    [string]$RunArgs = ''
  )

  $argsLine = '--stop-timeout 500ms --remap-exit 137:0 -- ' + $AppCommand + ' ' + $RunArgs + ' --duration 30 --pid-file "' + $ParentPidFile + '" --child-pid-file "' + $ChildPidFile + '"'
  $proc = Start-TiniWrapped -ArgsLine $argsLine.Trim()
  Wait-ForFile -Path $ParentPidFile
  Wait-ForFile -Path $ChildPidFile
  $parentPid = Read-PidFile $ParentPidFile
  $childPid = Read-PidFile $ChildPidFile
  Assert-ProcessExists -TargetPid $parentPid -Message "$Label parent pid $parentPid was not observed running"
  Assert-ProcessExists -TargetPid $childPid -Message "$Label child pid $childPid was not observed running"
  Stop-Process -Id $proc.Id
  Wait-ForProcessGone -TargetPid $parentPid
  Wait-ForProcessGone -TargetPid $childPid
  Write-Host "$Label parent pid=$parentPid child pid=$childPid cleaned up after wrapper close"
}

function Invoke-IgnoreStopProof {
  param(
    [Parameter(Mandatory = $true)][string]$Label,
    [Parameter(Mandatory = $true)][string]$AppCommand,
    [Parameter(Mandatory = $true)][string]$PidFile,
    [string]$RunArgs = ''
  )

  $argsLine = '--stop-timeout 500ms -- ' + $AppCommand + ' ' + $RunArgs + ' --pid-file "' + $PidFile + '"'
  $proc = Start-TiniWrapped -ArgsLine $argsLine.Trim()
  Wait-ForFile -Path $PidFile
  $targetProcPid = Read-PidFile $PidFile
  Assert-ProcessExists -TargetPid $targetProcPid -Message "$Label pid $targetProcPid was not observed running"
  Stop-Process -Id $proc.Id
  Wait-ForProcessGone -TargetPid $targetProcPid
  Write-Host "$Label pid=$targetProcPid cleaned up after wrapper close"
}

function Start-NginxTiniJob {
  param(
    [Parameter(Mandatory = $true)][string]$InstanceDir,
    [Parameter(Mandatory = $true)][string]$ConfPath
  )

  Start-Job -ScriptBlock {
    param($tiniWinExePath, $nginxExePath, $instanceDirPath, $confFilePath)
    & $tiniWinExePath --graceful-stop ('"' + $nginxExePath + '" -p "' + $instanceDirPath + '" -c "' + $confFilePath + '" -s quit') --stop-timeout 5s -- $nginxExePath -p $instanceDirPath -c $confFilePath
    exit $LASTEXITCODE
  } -ArgumentList $tiniWinExe, $localNginxExe, $InstanceDir, $ConfPath
}

function Finish-NginxTiniJob {
  param([Parameter(Mandatory = $true)]$Job)
  Wait-Job $Job | Out-Null
  $output = Receive-Job $Job -Keep
  $code = $Job.ChildJobs[0].JobStateInfo.Reason.HResult
  Remove-Job $Job -Force
  return $output
}

function Invoke-NginxHealthyProof {
  param(
    [Parameter(Mandatory = $true)][string]$InstanceDir,
    [Parameter(Mandatory = $true)][int]$Port
  )
  $pidFile = Join-Path $InstanceDir 'logs\nginx.pid'
  $conf = Join-Path $InstanceDir 'conf\nginx.conf'
  $url = 'http://127.0.0.1:' + $Port + '/health'

  $job = Start-NginxTiniJob -InstanceDir $InstanceDir -ConfPath $conf
  Wait-ForFile -Path $pidFile -TimeoutSeconds 15
  $masterPid = Read-PidFile $pidFile
  Assert-ProcessExists -TargetPid $masterPid -Message "nginx healthy pid $masterPid was not observed running"
  $response = Wait-ForHttpStatus -Url $url -ExpectedStatus 200 -TimeoutSeconds 15
  if ([string]$response.Content -notmatch 'ok') {
    throw 'nginx healthy /health response did not contain ok'
  }
  & $localNginxExe -p $InstanceDir -c $conf -s quit | Out-Null
  Wait-ForProcessGone -TargetPid $masterPid -TimeoutSeconds 15
  Wait-Job $job | Out-Null
  Receive-Job $job -Keep | Out-Null
  Remove-Job $job -Force
  Write-Host "nginx healthy pid=$masterPid served /health and exited cleanly after quit"
}

function Invoke-NginxNoHealthProof {
  param(
    [Parameter(Mandatory = $true)][string]$InstanceDir,
    [Parameter(Mandatory = $true)][int]$Port
  )
  $pidFile = Join-Path $InstanceDir 'logs\nginx.pid'
  $conf = Join-Path $InstanceDir 'conf\nginx.conf'
  $rootUrl = 'http://127.0.0.1:' + $Port + '/'
  $healthUrl = 'http://127.0.0.1:' + $Port + '/health'

  $job = Start-NginxTiniJob -InstanceDir $InstanceDir -ConfPath $conf
  Wait-ForFile -Path $pidFile -TimeoutSeconds 15
  $masterPid = Read-PidFile $pidFile
  Assert-ProcessExists -TargetPid $masterPid -Message "nginx no-health pid $masterPid was not observed running"
  $null = Wait-ForHttpStatus -Url $rootUrl -ExpectedStatus 200 -TimeoutSeconds 15
  try {
    $healthResponse = Invoke-WebRequest -UseBasicParsing -Uri $healthUrl -TimeoutSec 2
    if ([int]$healthResponse.StatusCode -eq 200) {
      throw 'nginx no-health unexpectedly returned 200 for /health'
    }
  } catch {
    $status = if ($_.Exception.Response) { [int]$_.Exception.Response.StatusCode } else { -1 }
    if ($status -ne 404) {
      throw "nginx no-health expected 404 for /health, got $status"
    }
  }
  & $localNginxExe -p $InstanceDir -c $conf -s quit | Out-Null
  Wait-ForProcessGone -TargetPid $masterPid -TimeoutSeconds 15
  Wait-Job $job | Out-Null
  Receive-Job $job -Keep | Out-Null
  Remove-Job $job -Force
  Write-Host "nginx no-health pid=$masterPid served / and returned 404 for /health as expected"
}

function Invoke-NginxInvalidConfigProof {
  param(
    [Parameter(Mandatory = $true)][string]$InstanceDir
  )
  $conf = Join-Path $InstanceDir 'conf\nginx.conf'
  $argsLine = '-- ' + (Quote-CommandPath $localNginxExe) + ' -p "' + $InstanceDir + '" -c "' + $conf + '"'
  & $tiniWinExe -- $localNginxExe -p $InstanceDir -c $conf
  $code = $LASTEXITCODE
  if ($code -eq 0) {
    throw 'nginx invalid-config unexpectedly exited 0'
  }
  Write-Host "nginx invalid-config failed fast with exit code=$code"
}

.\scripts\build.ps1
.\scripts\build-testapps.ps1
.\scripts\build-sample-projects.ps1

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tini-win-proof-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force $tempDir | Out-Null

Write-Host ""
Write-Host "== Case 1: simple exit with pid file =="
$simplePidFile = Join-Path $tempDir 'simple-exit.pid'
& .\bin\tini-win.exe -- .\bin\testapps\simple-exit.exe --pid-file $simplePidFile --sleep-ms 150
if ($LASTEXITCODE -ne 0) { throw "simple-exit case failed with code $LASTEXITCODE" }
Wait-ForFile -Path $simplePidFile
Write-Host "simple-exit pid=$(Read-PidFile $simplePidFile) code=$LASTEXITCODE"

Write-Host ""
Write-Host "== Case 2: remap exit code 143->0 =="
& .\bin\tini-win.exe --remap-exit 143:0 -- cmd /c exit 143
if ($LASTEXITCODE -ne 0) { throw "remap case failed with code $LASTEXITCODE" }
Write-Host "remap case code=$LASTEXITCODE"

Write-Host ""
Write-Host "== Case 3: graceful-stop child via tini-win =="
$signalFile = Join-Path $tempDir 'graceful-stop.signal'
$gracefulPidFile = Join-Path $tempDir 'graceful-stop.pid'
$gracefulChildExe = Join-Path $testAppsDir 'graceful-stop.exe'
$gracefulSignalCommand = '& ' + (Quote-CommandPath $gracefulChildExe) + ' --send --signal-file "' + $signalFile + '"'
Invoke-GracefulProof -Label 'graceful-stop child' -AppCommand (Quote-CommandPath $gracefulChildExe) -SignalCommand $gracefulSignalCommand -PidFile $gracefulPidFile -SignalFile $signalFile

Write-Host ""
Write-Host "== Case 4: spawn-child tree cleanup via tini-win =="
$spawnParentPidFile = Join-Path $tempDir 'spawn-parent.pid'
$spawnChildPidFile = Join-Path $tempDir 'spawn-child.pid'
$spawnChildExe = Join-Path $testAppsDir 'spawn-child.exe'
Invoke-SpawnCleanupProof -Label 'spawn-child' -AppCommand (Quote-CommandPath $spawnChildExe) -ParentPidFile $spawnParentPidFile -ChildPidFile $spawnChildPidFile

Write-Host ""
Write-Host "== Case 5: ignore-stop forced cleanup via tini-win =="
$ignorePidFile = Join-Path $tempDir 'ignore-stop.pid'
$ignoreStopExe = Join-Path $testAppsDir 'ignore-stop.exe'
Invoke-IgnoreStopProof -Label 'ignore-stop' -AppCommand (Quote-CommandPath $ignoreStopExe) -PidFile $ignorePidFile

Write-Host ""
Write-Host "== Case 6: Go sample project graceful-stop =="
$goSignalFile = Join-Path $tempDir 'go-sample.signal'
$goPidFile = Join-Path $tempDir 'go-sample.pid'
$goSignalCommand = '& ' + (Quote-CommandPath $goSampleExe) + ' --mode graceful-stop --send --signal-file "' + $goSignalFile + '"'
Invoke-GracefulProof -Label 'go sample graceful-stop' -AppCommand (Quote-CommandPath $goSampleExe) -SignalCommand $goSignalCommand -PidFile $goPidFile -SignalFile $goSignalFile -RunArgs '--mode graceful-stop'

Write-Host ""
Write-Host "== Case 7: Java sample project spawn-child cleanup =="
$javaParentPidFile = Join-Path $tempDir 'java-sample-parent.pid'
$javaChildPidFile = Join-Path $tempDir 'java-sample-child.pid'
Invoke-SpawnCleanupProof -Label 'java sample spawn-child' -AppCommand (Quote-CommandPath $javaSampleCmd) -ParentPidFile $javaParentPidFile -ChildPidFile $javaChildPidFile -RunArgs '--mode spawn-child'

Write-Host ""
Write-Host "== Case 8: Java sample project ignore-stop cleanup =="
$javaIgnorePidFile = Join-Path $tempDir 'java-sample-ignore.pid'
Invoke-IgnoreStopProof -Label 'java sample ignore-stop' -AppCommand (Quote-CommandPath $javaSampleCmd) -PidFile $javaIgnorePidFile -RunArgs '--mode ignore-stop'

Write-Host ""
Write-Host "== Case 9: integration tests =="
go test .\internal\app .\internal\runner -v

Write-Host ""
Write-Host "== Case 10: nginx local fixture availability =="
if (Test-Path $localNginxExe) {
  Write-Host "local nginx fixture found at $localNginxExe"
} else {
  throw 'local nginx fixture missing'
}

Write-Host ""
Write-Host "== Case 11: nginx healthy scenario =="
$nginxHealthyDir = Join-Path $tempDir 'nginx-healthy'
Render-NginxScenario -Scenario 'healthy' -Port 18080 -OutputDir $nginxHealthyDir
Invoke-NginxHealthyProof -InstanceDir $nginxHealthyDir -Port 18080

Write-Host ""
Write-Host "== Case 12: nginx no-health scenario =="
$nginxNoHealthDir = Join-Path $tempDir 'nginx-no-health'
Render-NginxScenario -Scenario 'no-health' -Port 18081 -OutputDir $nginxNoHealthDir
Invoke-NginxNoHealthProof -InstanceDir $nginxNoHealthDir -Port 18081

Write-Host ""
Write-Host "== Case 13: nginx invalid-config scenario =="
$nginxInvalidDir = Join-Path $tempDir 'nginx-invalid'
Render-NginxScenario -Scenario 'invalid-config' -Port 18082 -OutputDir $nginxInvalidDir
Invoke-NginxInvalidConfigProof -InstanceDir $nginxInvalidDir
