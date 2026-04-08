# tini-win Spec

## Product definition

`tini-win` is a standalone Windows-native tiny parent process for running one target program safely.

## Behavioral contract

`tini-win` must:
1. launch one child program
2. supervise that child program on Windows
3. support graceful stop when configured
4. support forced cleanup fallback when graceful stop fails
5. return a meaningful exit status

## Non-goals

`tini-win` is not:
- a multi-service orchestrator
- a workflow engine
- a deployment platform
- a generic dashboard/service manager

## MVP CLI model

```text
tini-win [OPTIONS] -- PROGRAM [ARGS...]
```

## Expected later options
- `--graceful-stop <cmd>`
- `--stop-timeout <seconds>`
- `--kill-tree`
- `--remap-exit <from>:<to>`
- `-v`

## Core implementation direction

- use Windows-native process creation
- use Job Objects for managed process-tree behavior
- attempt graceful stop first when configured
- force terminate tree if timeout expires
- keep the tool small and focused
