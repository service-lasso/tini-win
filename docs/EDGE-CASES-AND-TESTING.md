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
- breakaway-child behavior is characterized, including environment-dependent failure to create a breakaway child
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
