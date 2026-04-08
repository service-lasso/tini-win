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

### E. Output behavior
- stdout/stderr remain usable
- `tini-win` reports enough lifecycle info without becoming noisy

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
Optionally test one heavier real JVM-based service with more complex descendant behavior.

## Acceptance baseline

`tini-win` is considered viable when it can:
- pass the dedicated test apps
- pass the repo-local Go and Java sample project proofs
- manage graceful stop and forced cleanup correctly
- prove nginx lifecycle control on Windows, including scenario-driven config variation, well enough for real use
