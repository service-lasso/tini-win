package runner

// Package runner will hold the child-process lifecycle model for tini-win.
//
// Planned responsibilities:
//   - spawn one child process
//   - wait for child lifecycle events
//   - execute optional graceful-stop command
//   - enforce timeout
//   - fall back to forced termination
//   - report final exit reason/status
