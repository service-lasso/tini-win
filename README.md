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
- characterization coverage for relaunch/orphan, brokered-child escape, breakaway attempt behavior, inherited-stdio cleanup, console control-event handling, external scheduler/WMI launch, and restart/port-rebind reuse

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
- Java sample project cases, including Java-specific launch paths (`ProcessBuilder`, `Runtime.exec`, batch-wrapper launch, relaunch-orphan, broker/client characterization)
- repo-local nginx scenarios, including forced wrapper-only teardown without `nginx -s quit`
- relaunch/orphan cleanup proof
- brokered-child escape characterization
- breakaway-child characterization attempt
- inherited-stdio cleanup proof
- console ctrl-break graceful-stop proof
- external scheduler launch characterization
- external WMI launch characterization
- restart/port-rebind integration proof

## Current observed findings

What the current proof set shows:
- normal descendants in the same Job Object clean up correctly
- relaunch-orphan cleanup works under the current runner model
- Java normal launch paths also clean up correctly under the same job-tree model
- inherited stdio alone did not break cleanup in the current proof
- ctrl-break aware apps can be exercised explicitly with the console fixture
- brokered work is a real escape path, both in the dedicated native broker fixture and the Java broker/client fixture
- Task Scheduler and WMI launches are real out-of-job-process creation paths and should be treated as outside ordinary `tini-win` containment
- explicit breakaway creation is blocked under the default job configuration in this environment (`Access is denied`)
- explicit breakaway creation is now also positively proven in this environment when `tini-win` is started with `--allow-breakaway`, which enables `JOB_OBJECT_LIMIT_BREAKAWAY_OK` for characterization/testing
- nginx master/worker processes launched under `tini-win` are now explicitly proven to die from wrapper/job teardown alone, without telling nginx to quit, in the forced-kill proof path

## Release pipeline

Local packaging:

```powershell
.\scripts\package-release.ps1 -Version 2026.4.9-86a7a68 -Platform windows-amd64 -OutputDir .\dist
```

GitHub Actions workflow:
- `.github/workflows/release.yml`

Current release behavior:
- runs on every push to `master`
- also supports manual `workflow_dispatch` with an optional version override
- runs tests on `windows-latest`
- builds a Windows release binary
- packages `tini-win.exe` + `README.md` + `LICENSE` into a zip
- writes a `.sha256` checksum file
- uploads artifacts on workflow runs
- creates/updates a GitHub Release automatically for each `master` build

## Versioning

Canonical release versioning is:
- `yyyy.m.d-<shortsha>`

Example:
- `2026.4.9-86a7a68`

Why this shape:
- readable date-based version
- unique on every build
- commit is always embedded for traceability
- consistent across release labels, artifacts, and tags

The release tag format is:
- `vyyyy.m.d-<shortsha>`

You can still manually override the version in `workflow_dispatch` if needed, but the intended override shape is the same canonical format: `yyyy.m.d-<shortsha>`.

## Example

```powershell
.\bin\tini-win.exe --graceful-stop "nginx.exe -s quit" --stop-timeout 15s --kill-tree -- nginx.exe -p C:\instance\nginx -c C:\instance\nginx\conf\nginx.conf
```

## Notes

- This is a Windows-native tiny supervisor, not a full service manager.
- It is intentionally focused on one-child lifecycle control.
- A `breakaway child` means a spawned process that escapes the parent Job Object and may survive cleanup that kills the normal job tree.
- `--allow-breakaway` is now available as an explicit opt-in characterization/testing mode. It weakens containment on purpose so successful breakaway behavior can be proven when needed; it should not be treated as the normal safe default.
- Current Java proof coverage now includes normal JVM lifecycle plus Java-specific launch paths like `ProcessBuilder`, `Runtime.exec(String[])`, `Runtime.exec(String)`, batch-wrapper launch, relaunch-orphan, and broker/client characterization.
- On Windows, Java does not use Unix `fork()` semantics in the usual sense. `ProcessBuilder` or `Runtime.exec(...)` normally create a child process, which should remain in the same Job Object unless launched through some external broker/escape mechanism.
- External launch paths like Task Scheduler and WMI are now explicitly characterized as out-of-job-process creation paths, and they do produce independently running work outside ordinary `tini-win` containment.
- Console-control-event behavior is now characterized with a ctrl-break-aware fixture so apps that rely on control events can be tested explicitly.
- The remaining Java-specific gap is true in-JVM breakaway creation with explicit Windows breakaway flags. That is not yet proven here.
- The broader remaining limitation is simple: `tini-win` is strong for ordinary descendants in the wrapped job tree, but it should not claim universal containment across all Windows launch mechanisms.
- The nginx proof path uses a PowerShell job launch because `Start-Process` can lose the `--` separator and misroute child flags like `-p` into `tini-win` parsing.
- License: Apache 2.0 (`LICENSE`).
- See `docs/SPEC.md`, `docs/EDGE-CASES-AND-TESTING.md`, and `samples/README.md` for scope and validation details.
- `docs/EDGE-CASES-AND-TESTING.md` now includes an explicit example-by-example matrix covering every current proof fixture and its observed result.
