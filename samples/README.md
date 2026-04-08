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
- `bin/samples/java/child-wrapper.cmd`

## Supported modes
Both samples support:
- `--mode simple-exit`
- `--mode graceful-stop`
- `--mode ignore-stop`
- `--mode spawn-child`

Java sample also supports:
- `--mode spawn-child-shell`
- `--mode runtime-exec-array`
- `--mode runtime-exec-string`
- `--mode batch-wrapper-child`
- `--mode relaunch-orphan`
- `--mode broker`
- `--mode client`

Shared flags:
- `--pid-file <path>`
- `--child-pid-file <path>` for child-launch modes
- `--signal-file <path>` for `graceful-stop`
- `--send` for `graceful-stop`
- `--sleep-ms <n>` for `simple-exit`
- `--duration <n>` for child-launch/broker modes
- `--exit-code <n>` for `simple-exit`
- `--request-file <path>` / `--stop-file <path>` for broker/client modes

## Proof coverage currently wired
- Go sample: graceful-stop
- Java sample: spawn-child cleanup
- Java sample: ignore-stop cleanup
- Java sample: `ProcessBuilder` shell launch cleanup
- Java sample: `Runtime.exec(String[])` cleanup
- Java sample: `Runtime.exec(String)` cleanup
- Java sample: batch-wrapper launch cleanup
- Java sample: relaunch-orphan cleanup
- Java sample: brokered-child characterization

## Java scope note
The Java sample now proves several Java-specific launch paths under `tini-win`.

That means:
- covered: normal Java launch, `ProcessBuilder`, `Runtime.exec(String[])`, `Runtime.exec(String)`, batch-wrapper launch, relaunch-orphan, and broker/client brokered launch characterization
- partly characterized: brokered child escape paths, where work started by an already-running broker can survive wrapped-client shutdown
- not yet covered: a true Java-specific Job Object breakaway path with explicit Windows breakaway creation flags from inside the JVM
- not the same as Unix `fork()`: on Windows, Java child creation here is effectively Windows process launch behavior, not classic fork semantics

The sample apps exist to simulate app-level lifecycle behavior while still being deterministic enough for local proof runs.
