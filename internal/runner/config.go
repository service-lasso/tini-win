package runner

import "time"

// Config controls one tini-win execution.
type Config struct {
	Command       []string
	GracefulStop  string
	StopTimeout   time.Duration
	KillTree      bool
	Verbose       bool
	RemapExitCode map[int]int
}

// ExitCodeError indicates a process completed with a specific exit code.
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return "process exited"
}
