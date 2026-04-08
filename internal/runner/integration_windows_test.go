//go:build windows

package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRunContext_SimpleExitWritesPIDFile(t *testing.T) {
	exe := buildTestApp(t, "simple-exit")
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "simple-exit.pid")

	var out, errb bytes.Buffer
	err := RunContext(context.Background(), Config{
		Command:     []string{exe, "--pid-file", pidFile, "--sleep-ms", "150"},
		StopTimeout: 2 * time.Second,
		KillTree:    true,
		Verbose:     true,
	}, &out, &errb)
	if err != nil {
		t.Fatalf("expected simple exit success, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
	}

	pid := waitForPIDFile(t, pidFile, 2*time.Second)
	if pid <= 0 {
		t.Fatalf("expected valid pid, got %d", pid)
	}
}

func TestRunContext_GracefulStopCommand(t *testing.T) {
	exe := buildTestApp(t, "graceful-stop")
	tempDir := t.TempDir()
	signalFile := filepath.Join(tempDir, "stop.signal")
	pidFile := filepath.Join(tempDir, "graceful-stop.pid")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:      []string{exe, "--signal-file", signalFile, "--pid-file", pidFile},
			GracefulStop: "\"" + exe + "\" --send --signal-file \"" + signalFile + "\"",
			StopTimeout:  2 * time.Second,
			KillTree:     true,
			Verbose:      true,
		}, &out, &errb)
	}()

	pid := waitForPIDFile(t, pidFile, 5*time.Second)
	waitForProcessState(t, pid, true, 3*time.Second)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected graceful stop success, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for graceful stop\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}

	if _, err := os.Stat(signalFile); err != nil {
		t.Fatalf("expected signal file to be created, err=%v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
	}
	waitForProcessState(t, pid, false, 5*time.Second)
}

func TestRunContext_KillTreeTerminatesSpawnedChild(t *testing.T) {
	exe := buildTestApp(t, "spawn-child")
	tempDir := t.TempDir()
	parentPIDFile := filepath.Join(tempDir, "spawn-parent.pid")
	childPIDFile := filepath.Join(tempDir, "spawn-child.pid")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:       []string{exe, "--duration", "30", "--pid-file", parentPIDFile, "--child-pid-file", childPIDFile},
			StopTimeout:   500 * time.Millisecond,
			KillTree:      true,
			Verbose:       true,
			RemapExitCode: map[int]int{137: 0},
		}, &out, &errb)
	}()

	parentPID := waitForPIDFile(t, parentPIDFile, 5*time.Second)
	childPID := waitForPIDFile(t, childPIDFile, 5*time.Second)
	waitForProcessState(t, parentPID, true, 3*time.Second)
	waitForProcessState(t, childPID, true, 3*time.Second)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected forced-stop remapped success, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for forced stop\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}

	waitForProcessState(t, parentPID, false, 5*time.Second)
	waitForProcessState(t, childPID, false, 5*time.Second)
}

func TestRunContext_ForcedStopReturnsExitCodeErrorWithoutRemap(t *testing.T) {
	exe := buildTestApp(t, "ignore-stop")
	tempDir := t.TempDir()
	pidFile := filepath.Join(tempDir, "ignore-stop.pid")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:     []string{exe, "--pid-file", pidFile},
			StopTimeout: 500 * time.Millisecond,
			KillTree:    true,
			Verbose:     true,
		}, &out, &errb)
	}()

	pid := waitForPIDFile(t, pidFile, 5*time.Second)
	waitForProcessState(t, pid, true, 3*time.Second)
	time.Sleep(1200 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		var ee *ExitCodeError
		if !errors.As(err, &ee) {
			t.Fatalf("expected ExitCodeError, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
		if ee.Code != 137 {
			t.Fatalf("expected forced-stop code 137, got %d\nstdout:\n%s\nstderr:\n%s", ee.Code, out.String(), errb.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for forced-stop result\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}

	waitForProcessState(t, pid, false, 5*time.Second)
}

func buildTestApp(t *testing.T, name string) string {
	t.Helper()
	root := findRepoRoot(t)
	out := filepath.Join(t.TempDir(), name+".exe")
	src := filepath.Join(root, "testapps", name)
	cmd := exec.Command("go", "build", "-o", out, src)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build test app %s: %v\n%s", name, err, stderr.String())
	}
	return out
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd))
}

func waitForPIDFile(t *testing.T, path string, timeout time.Duration) int {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		b, err := os.ReadFile(path)
		if err == nil {
			pid, convErr := strconv.Atoi(strings.TrimSpace(string(b)))
			if convErr != nil {
				t.Fatalf("invalid pid file contents: %v", convErr)
			}
			return pid
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for pid file %s", path)
	return 0
}

func waitForProcessState(t *testing.T, pid int, wantExists bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if processExists(pid) == wantExists {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for pid %d exist=%t", pid, wantExists)
}

func processExists(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.Contains(out.String(), strconv.Itoa(pid))
}
