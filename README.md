# tini-win

`tini-win` is a standalone Windows-native tiny process babysitter.

## Goal

Run one target program safely on Windows with a small parent process that can:
- launch the child
- supervise it
- support graceful stop commands
- force-clean the process tree on timeout/failure
- return meaningful exit status

## Current status

Runnable MVP with repo-local proof fixtures.

Implemented now:
- CLI parsing with `--` separator
- child process spawn/wait
- exit-code passthrough + remap (`--remap-exit`)
- Windows Job Object assignment
- optional process-tree kill behavior
- optional graceful-stop command + timeout fallback
- signal-aware CLI cancellation via `signal.NotifyContext`
- unit + integration tests for parser/runner core
- purpose-built Go test apps for lifecycle edge cases
- repo-local Go and Java sample projects for more app-like lifecycle proof
- repo-local nginx fixture + scenario configs (`healthy`, `no-health`, `invalid-config`)

## Project layout

- `cmd/tini-win/` - CLI entrypoint
- `internal/app/` - argument parsing / app wiring
- `internal/runner/` - child process lifecycle model
- `internal/winjob/` - Windows Job Object integration
- `testapps/` - small purpose-built Go programs used to prove lifecycle behavior
- `samples/` - repo-local Go and Java sample apps for more realistic lifecycle proof
- `tests/nginx/` - repo-local nginx fixture + scenario templates
- `docs/` - spec and testing docs
- `scripts/` - build/test/proof helpers

## Build

```powershell
.\scripts\build.ps1
```

Output:
- `.\bin\tini-win.exe`

## Run tests

```powershell
.\scripts\test.ps1
```

## Build app-like sample projects

```powershell
.\scripts\build-sample-projects.ps1
```

Output:
- `./bin/samples/go/edgecase-go.exe`
- `./bin/samples/java/edgecase-app.jar`
- `./bin/samples/java/edgecase-app.cmd`

## Build sample test apps

```powershell
.\scripts\build-testapps.ps1
```

Output:
- `.\bin\testapps\simple-exit.exe`
- `.\bin\testapps\spawn-child.exe`
- `.\bin\testapps\ignore-stop.exe`
- `.\bin\testapps\graceful-stop.exe`

## Quick manual checks

### 1) Basic passthrough exit
```powershell
.\bin\tini-win.exe -- .\bin\testapps\simple-exit.exe
$LASTEXITCODE
```

### 2) Exit-code remap
```powershell
.\bin\tini-win.exe --remap-exit 143:0 -- cmd /c exit 143
$LASTEXITCODE
```

### 3) Kill-tree fallback check (manual)
```powershell
.\bin\tini-win.exe --stop-timeout 3s -- .\bin\testapps\ignore-stop.exe
```
(then interrupt process and verify child cleanup behavior)

## Proof flow

```powershell
.\scripts\prove-mvp.ps1
```

Current proof coverage includes:
- small lifecycle test apps
- Go sample project cases
- Java sample project cases
- repo-local nginx scenarios

## Example

```powershell
.\bin\tini-win.exe --graceful-stop "nginx.exe -s quit" --stop-timeout 15s --kill-tree -- nginx.exe -p C:\instance\nginx -c C:\instance\nginx\conf\nginx.conf
```

## Notes

- This is a Windows-native tiny supervisor, not a full service manager.
- It is intentionally focused on one-child lifecycle control.
- The nginx proof path uses a PowerShell job launch because `Start-Process` can lose the `--` separator and misroute child flags like `-p` into `tini-win` parsing.
- See `docs/SPEC.md`, `docs/EDGE-CASES-AND-TESTING.md`, and `samples/README.md` for scope and validation details.
