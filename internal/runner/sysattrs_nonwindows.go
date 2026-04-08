//go:build !windows

package runner

import "os/exec"

func setChildProcessAttrs(cmd *exec.Cmd) {}
