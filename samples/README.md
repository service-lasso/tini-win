# sample projects

These are slightly more app-like lifecycle fixtures than the tiny `testapps/` programs, and they are built into the normal `prove-mvp.ps1` flow.

## Go sample
- Path: `samples/go/edgecase-app`
- Output: `bin/samples/go/edgecase-go.exe`

## Java sample
- Path: `samples/java/edgecase-app`
- Output: `bin/samples/java/edgecase-app.jar`
- Wrapper: `bin/samples/java/edgecase-app.cmd`

## Build

```powershell
.\scripts\build-sample-projects.ps1
```

Outputs:
- `bin/samples/go/edgecase-go.exe`
- `bin/samples/java/edgecase-app.jar`
- `bin/samples/java/edgecase-app.cmd`

## Supported modes
Both samples support:
- `--mode simple-exit`
- `--mode graceful-stop`
- `--mode ignore-stop`
- `--mode spawn-child`

Shared flags:
- `--pid-file <path>`
- `--child-pid-file <path>` for `spawn-child`
- `--signal-file <path>` for `graceful-stop`
- `--send` for `graceful-stop`
- `--sleep-ms <n>` for `simple-exit`
- `--duration <n>` for `spawn-child`
- `--exit-code <n>` for `simple-exit`

## Proof coverage currently wired
- Go sample: graceful-stop
- Java sample: spawn-child cleanup
- Java sample: ignore-stop cleanup

The sample apps exist to simulate app-level lifecycle behavior while still being deterministic enough for local proof runs.
