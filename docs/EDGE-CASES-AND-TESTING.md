# Edge Cases and Testing

## Purpose

Document the edge cases `tini-win` must handle and define the dedicated test apps used to prove each behavior.

## Testing strategy

Do not rely on theory alone. Prove behavior with small purpose-built test programs, then app-like sample projects, then real-world targets like nginx.

## Test app matrix

### 1. `testapps/simple-exit`
#### Purpose
Prove the baseline case:
- child starts
- child exits normally
- exit status is propagated

#### What to test
- child exit code passthrough
- normal stdout/stderr passthrough
- no false forced-stop behavior

---

### 2. `testapps/spawn-child`
#### Purpose
Prove child-tree behavior:
- parent process starts
- parent spawns child/grandchild
- tree must be cleaned up correctly

#### What to test
- `tini-win` can terminate the relevant process tree
- descendants do not remain behind after forced cleanup
- observed process state matches expected tree-lifecycle behavior

---

### 3. `testapps/ignore-stop`
#### Purpose
Simulate a process that does not stop cleanly on a naive request.

#### What to test
- graceful-stop timeout is honored
- fallback forced cleanup happens
- final exit reason is classified correctly

---

### 4. `testapps/graceful-stop`
#### Purpose
Simulate a process that supports a proper explicit graceful shutdown command.

#### What to test
- stop command executes successfully
- process exits without forced kill
- final status reflects graceful stop success

---

### 5. `testapps/relaunch-orphan`
#### Purpose
Simulate a parent that quickly spawns a replacement child and exits.

#### What to test
- replacement/orphan child does not remain behind after the wrapped parent exits
- the cleanup model still holds when the original parent lifecycle is intentionally short

---

### 6. `testapps/brokered-child`
#### Purpose
Simulate work being spawned by an already-running external broker instead of directly by the wrapped process.

#### What to test
- whether broker-spawned work is outside the wrapped job tree
- whether that work can survive wrapped-client shutdown
- whether the gap is clearly characterized and documented

---

### 7. `testapps/breakaway-child`
#### Purpose
Attempt to simulate a child intentionally escaping the parent Job Object.

#### What to test
- whether breakaway creation succeeds in the current Windows/job configuration
- whether an escaped child survives normal wrapper cleanup
- if breakaway creation is blocked, document that as an environment-specific characterization result rather than a false proof of containment

---

## App-like validation targets

### Go sample project
Use after the tiny `testapps/` pass.

#### What to prove
- graceful-stop works in a more app-like executable
- PID files and signal-file behavior still work through `tini-win`

### Java sample project
Use after the Go sample passes.

#### What to prove
- JVM-based process start/stop behaves correctly under `tini-win`
- spawned-child cleanup still works
- forced cleanup still works on a longer-lived runtime
- normal Java child creation paths (`ProcessBuilder`, `Runtime.exec`) behave like ordinary managed children under the job model
- shell-indirected and batch-wrapper launch paths are still cleaned up correctly
- relaunch-orphan behavior is understood for Java launch paths
- broker/client launch behavior is characterized for Java as well

#### Important scope note
This now covers several Java-specific launch mechanisms, but it is still not a proof of a true in-JVM breakaway path with explicit Windows breakaway creation flags. On Windows, Java is not exercising Unix-style `fork()` semantics here. The current Java sample proves realistic JVM launch behavior and brokered-launch characterization, not every possible helper-launch mechanism.

## Real-world validation target

### nginx on Windows
Use after the smaller proof targets pass.

#### What to prove
- launch works
- graceful stop command works
- forced wrapper-only teardown kills nginx master/workers without needing `nginx -s quit`
- probe behavior can be varied by config scenario
- startup failure is surfaced cleanly on invalid config
- no relevant worker processes remain unexpectedly

---

## Required proof areas

### A. Child launch and wait
- spawn one process
- wait for it
- capture exit code reliably

### B. Process-tree cleanup
- managed child descendants are terminated when expected
- naive main-pid stop is not the only mechanism relied upon

### C. Graceful stop + timeout + fallback
- graceful stop command can be run
- timeout is enforced
- fallback kill occurs when required

### D. Parent death behavior
- determine what happens if `tini-win` itself is terminated unexpectedly
- verify child-tree outcome is understood and documented

### E. Escape-path characterization
- relaunch/orphan behavior is understood
- brokered-child escape paths are characterized
- breakaway-child behavior is characterized, including the default blocked case and the successful opt-in breakaway case when the job explicitly allows it
- external scheduler/WMI launch paths are characterized as out-of-job-process creation paths

### F. Handle / output behavior
- stdout/stderr remain usable
- inherited-stdio child behavior is understood
- `tini-win` reports enough lifecycle info without becoming noisy

### G. Console control behavior
- apps that rely on console control events can be characterized explicitly
- ctrl-break delivery behavior is understood separately from plain termination

## Test phases

### Phase 1
Use only the dedicated `testapps/` fixtures.

### Phase 2
Use the repo-local Go and Java sample projects.

### Phase 3
Use repo-local nginx scenarios on Windows.
- `healthy`
- `no-health`
- `invalid-config`

### Phase 4
Use characterization fixtures for escape and restart behavior.
- `relaunch-orphan`
- `brokered-child`
- `breakaway-child`
- inherited-stdio hold-open fixture
- console ctrl-break fixture
- Task Scheduler external launch characterization
- WMI external launch characterization
- restart/port-rebind server test

### Phase 5
Optionally test one heavier real JVM-based service with more complex descendant behavior.

## Acceptance baseline

`tini-win` is considered viable when it can:
- pass the dedicated test apps
- pass the repo-local Go and Java sample project proofs
- manage graceful stop and forced cleanup correctly
- prove nginx lifecycle control on Windows, including scenario-driven config variation, well enough for real use
- clearly document the remaining limits around brokered work, breakaway attempts, scheduler/WMI/service-style external launch paths, and other escape paths that are outside ordinary job-tree containment

## Current characterized results

Observed in the current proof set:
- ordinary descendants in the same Job Object are cleaned up correctly
- nginx master/workers launched under `tini-win` are cleaned up by wrapper/job teardown alone in the forced-kill proof, without relying on nginx's own quit command
- relaunch-orphan patterns are cleaned up correctly in the current runner model
- Java launch variants tested so far behave like normal managed children unless they explicitly hand work off to an external broker
- inherited stdio / handle-hold behavior did not block cleanup in the current fixture
- ctrl-break aware console apps can be exercised and observed separately from plain forced termination
- brokered-child is a confirmed escape gap
- Task Scheduler and WMI launches are confirmed external-launch escape paths
- explicit breakaway creation is blocked by default in this environment (`Access is denied`)
- explicit breakaway creation is positively proven when the job is created with breakaway explicitly allowed (`JOB_OBJECT_LIMIT_BREAKAWAY_OK` / `--allow-breakaway`)

Important interpretation:
- the successful breakaway proof is an opt-in weakening of containment for characterization/testing
- it proves the escape path is real in this environment
- it does not mean the default `tini-win` posture should allow breakaway

Still not fully proven:
- service-control-manager / COM / other privileged broker launch paths beyond the scheduler/WMI coverage already added

## Example matrix

This is the concrete example-by-example inventory of what is currently exercised.

### Dedicated `testapps/` fixtures

| Example | Purpose | Expected / observed result |
| --- | --- | --- |
| `simple-exit` | Baseline one-child launch/wait/exit passthrough | Exits cleanly, PID file is written, exit status propagates |
| `spawn-child` | Parent spawns normal child in the same job tree | Parent + child are both cleaned up by wrapper/job teardown |
| `ignore-stop` | Long-running process that does not stop cleanly on its own | Graceful wait times out, forced cleanup happens |
| `graceful-stop` | Process with explicit graceful stop command | Signal command succeeds and process exits cleanly |
| `relaunch-orphan` | Parent spawns replacement child and exits quickly | Replacement child is still cleaned up under the normal runner model |
| `brokered-child` | Already-running external broker spawns work on behalf of wrapped client | Broker-spawned child survives wrapped client shutdown, confirmed escape gap |
| `breakaway-child` default | Child tries explicit Windows breakaway from default job | Breakaway attempt is blocked in default mode in this environment (`Access is denied`) |
| `breakaway-child` with `--allow-breakaway` | Child tries explicit Windows breakaway when job explicitly allows it | Successful breakaway escape is positively proven in this environment |
| `port-rebind-server` | Service restart / port reuse behavior | Port is reusable after shutdown and restart test passes |
| `stdio-hold-open` | Child inherits stdout/stderr and keeps handles active | Cleanup still succeeds in the current proof |
| `console-trap` | Console control-event aware process | Ctrl-break delivery is observed and exits cleanly |

### Go sample example

| Example | Purpose | Expected / observed result |
| --- | --- | --- |
| Go `graceful-stop` sample | More app-like graceful shutdown behavior | Graceful stop works correctly through `tini-win` |

### Java sample examples

| Example | Purpose | Expected / observed result |
| --- | --- | --- |
| Java `spawn-child` | Normal JVM child launch via managed path | Cleanup succeeds |
| Java `ignore-stop` | Long-running JVM process requiring forced cleanup | Forced cleanup succeeds |
| Java `spawn-child-shell` | `ProcessBuilder` shell-indirected launch | Cleanup succeeds |
| Java `runtime-exec-array` | `Runtime.exec(String[])` launch path | Cleanup succeeds |
| Java `runtime-exec-string` | `Runtime.exec(String)` launch path | Cleanup succeeds |
| Java `batch-wrapper-child` | Java launches through a batch wrapper | Cleanup succeeds |
| Java `relaunch-orphan` | JVM process spawns replacement child and exits | Replacement child is cleaned up correctly |
| Java `broker` / `client` | External broker pattern driven from Java | Broker-spawned child survives wrapped client shutdown, confirmed escape gap |

### nginx examples

| Example | Purpose | Expected / observed result |
| --- | --- | --- |
| nginx `healthy` | Valid config with `/health` endpoint | Starts, serves `/health`, and exits cleanly via quit |
| nginx `forced wrapper-only teardown` | Start nginx master/workers and stop only through wrapper/job teardown | nginx master/workers die without sending `nginx -s quit` |
| nginx `no-health` | Valid config without `/health` handler | Serves `/`, returns 404 on `/health`, exits cleanly via quit |
| nginx `invalid-config` | Invalid startup config | Fails fast with nginx error |

### External-launch characterizations

| Example | Purpose | Expected / observed result |
| --- | --- | --- |
| Task Scheduler launch | Work created by Windows Task Scheduler | Starts outside ordinary `tini-win` job containment, confirmed external escape class |
| WMI / `Win32_Process.Create` launch | Work created by WMI process creation | Starts outside ordinary `tini-win` job containment, confirmed external escape class |
