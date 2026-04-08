package winjob

// Package winjob will wrap the Windows Job Object primitives used by tini-win.
//
// Planned responsibilities:
//   - create/open job objects
//   - assign process handles to job objects
//   - configure job limits/termination behavior
//   - terminate the managed process tree when required
