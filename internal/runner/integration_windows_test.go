//go:build windows

package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
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
	tempDir := repoTempDir(t)
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
	tempDir := repoTempDir(t)
	signalFile := filepath.Join(tempDir, "stop.signal")
	pidFile := filepath.Join(tempDir, "graceful-stop.pid")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:      []string{exe, "--signal-file", signalFile, "--pid-file", pidFile},
			GracefulStop: fmt.Sprintf("\"%s\" --send --signal-file \"%s\"", exe, signalFile),
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
	tempDir := repoTempDir(t)
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
	tempDir := repoTempDir(t)
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

func TestRunContext_RelaunchOrphanChildGetsCleanedUp(t *testing.T) {
	exe := buildTestApp(t, "relaunch-orphan")
	tempDir := repoTempDir(t)
	parentPIDFile := filepath.Join(tempDir, "relaunch-parent.pid")
	childPIDFile := filepath.Join(tempDir, "relaunch-child.pid")

	var out, errb bytes.Buffer
	err := RunContext(context.Background(), Config{
		Command:       []string{exe, "--duration", "30", "--pid-file", parentPIDFile, "--child-pid-file", childPIDFile},
		StopTimeout:   2 * time.Second,
		KillTree:      true,
		Verbose:       true,
		RemapExitCode: map[int]int{},
	}, &out, &errb)
	if err != nil {
		t.Fatalf("expected parent to exit cleanly, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
	}

	childPID := waitForPIDFile(t, childPIDFile, 5*time.Second)
	waitForProcessState(t, childPID, false, 5*time.Second)
}

func TestRunContext_BrokeredChildCanEscapeTree(t *testing.T) {
	brokerExe := buildTestApp(t, "brokered-child")
	clientExe := brokerExe
	tempDir := repoTempDir(t)
	requestFile := filepath.Join(tempDir, "broker.request")
	stopFile := filepath.Join(tempDir, "broker.stop")
	brokerPIDFile := filepath.Join(tempDir, "broker.pid")
	brokerChildPIDFile := filepath.Join(tempDir, "broker.child.pid")
	clientPIDFile := filepath.Join(tempDir, "client.pid")

	brokerCmd := exec.Command(brokerExe, "--mode", "broker", "--request-file", requestFile, "--stop-file", stopFile, "--pid-file", brokerPIDFile, "--child-pid-file", brokerChildPIDFile, "--duration", "30")
	var brokerOut, brokerErr bytes.Buffer
	brokerCmd.Stdout = &brokerOut
	brokerCmd.Stderr = &brokerErr
	if err := brokerCmd.Start(); err != nil {
		t.Fatalf("start broker: %v", err)
	}
	defer func() {
		_ = os.WriteFile(stopFile, []byte("stop"), 0o644)
		_ = brokerCmd.Process.Kill()
		_, _ = brokerCmd.Process.Wait()
	}()
	waitForPIDFile(t, brokerPIDFile, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:       []string{clientExe, "--mode", "client", "--request-file", requestFile, "--pid-file", clientPIDFile},
			StopTimeout:   500 * time.Millisecond,
			KillTree:      true,
			Verbose:       true,
			RemapExitCode: map[int]int{137: 0},
		}, &out, &errb)
	}()

	clientPID := waitForPIDFile(t, clientPIDFile, 5*time.Second)
	waitForProcessState(t, clientPID, true, 3*time.Second)
	childPID := waitForPIDFile(t, brokerChildPIDFile, 8*time.Second)
	waitForProcessState(t, childPID, true, 3*time.Second)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected wrapped client shutdown to succeed, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for brokered child client shutdown\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}

	waitForProcessState(t, clientPID, false, 5*time.Second)
	if !processExists(childPID) {
		t.Fatalf("expected broker-spawned child pid %d to survive wrapped client stop as a gap characterization\nstdout:\n%s\nstderr:\n%s\nbroker stdout:\n%s\nbroker stderr:\n%s", childPID, out.String(), errb.String(), brokerOut.String(), brokerErr.String())
	}

	_ = os.WriteFile(stopFile, []byte("stop"), 0o644)
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(childPID), "/T", "/F").Run()
	waitForProcessState(t, childPID, false, 5*time.Second)
}

func TestRunContext_BreakawayChildCharacterization(t *testing.T) {
	exe := buildTestApp(t, "breakaway-child")
	tempDir := repoTempDir(t)
	parentPIDFile := filepath.Join(tempDir, "breakaway-parent.pid")
	childPIDFile := filepath.Join(tempDir, "breakaway-child.pid")
	statusFile := filepath.Join(tempDir, "breakaway.status")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:       []string{exe, "--duration", "30", "--pid-file", parentPIDFile, "--child-pid-file", childPIDFile, "--status-file", statusFile},
			StopTimeout:   500 * time.Millisecond,
			KillTree:      true,
			Verbose:       true,
			RemapExitCode: map[int]int{137: 0},
		}, &out, &errb)
	}()

	waitForPIDFile(t, parentPIDFile, 5*time.Second)
	waitForFileExists(t, statusFile, 5*time.Second)
	statusBytes, _ := os.ReadFile(statusFile)
	status := strings.TrimSpace(string(statusBytes))
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected breakaway characterization wrapper shutdown to succeed, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for breakaway characterization\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}

	if strings.HasPrefix(status, "spawn-error:") {
		return
	}
	childPID := waitForPIDFile(t, childPIDFile, 5*time.Second)
	waitForProcessState(t, childPID, false, 5*time.Second)
}

func TestRunContext_BreakawayChildEscapesWhenAllowed(t *testing.T) {
	exe := buildTestApp(t, "breakaway-child")
	tempDir := repoTempDir(t)
	parentPIDFile := filepath.Join(tempDir, "breakaway-allowed-parent.pid")
	childPIDFile := filepath.Join(tempDir, "breakaway-allowed-child.pid")
	statusFile := filepath.Join(tempDir, "breakaway-allowed.status")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:        []string{exe, "--duration", "30", "--pid-file", parentPIDFile, "--child-pid-file", childPIDFile, "--status-file", statusFile},
			StopTimeout:    500 * time.Millisecond,
			KillTree:       true,
			AllowBreakaway: true,
			Verbose:        true,
			RemapExitCode:  map[int]int{137: 0},
		}, &out, &errb)
	}()

	waitForPIDFile(t, parentPIDFile, 5*time.Second)
	waitForFileExists(t, statusFile, 5*time.Second)
	statusBytes, _ := os.ReadFile(statusFile)
	status := strings.TrimSpace(string(statusBytes))
	if strings.HasPrefix(status, "spawn-error:") {
		cancel()
		<-errCh
		t.Fatalf("expected successful breakaway spawn when allow-breakaway is enabled, got %q\nstdout:\n%s\nstderr:\n%s", status, out.String(), errb.String())
	}
	childPID := waitForPIDFile(t, childPIDFile, 5*time.Second)
	waitForProcessState(t, childPID, true, 3*time.Second)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected breakaway-allowed wrapper shutdown to succeed, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for breakaway-allowed characterization\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}

	if !processExists(childPID) {
		t.Fatalf("expected breakaway child pid %d to survive when allow-breakaway is enabled\nstdout:\n%s\nstderr:\n%s", childPID, out.String(), errb.String())
	}
	_ = exec.Command("taskkill", "/PID", strconv.Itoa(childPID), "/T", "/F").Run()
	waitForProcessState(t, childPID, false, 5*time.Second)
}

func TestRunContext_StdoutStderrPassthrough(t *testing.T) {
	exe := buildTestApp(t, "stdout-stderr")
	tempDir := repoTempDir(t)
	pidFile := filepath.Join(tempDir, "stdout-stderr.pid")

	var out, errb bytes.Buffer
	err := RunContext(context.Background(), Config{
		Command:     []string{exe, "--pid-file", pidFile, "--stdout", "alpha-out", "--stderr", "beta-err"},
		StopTimeout: 2 * time.Second,
		KillTree:    true,
	}, &out, &errb)
	if err != nil {
		t.Fatalf("expected stdout-stderr success, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "alpha-out") {
		t.Fatalf("expected stdout passthrough, got:\n%s", out.String())
	}
	if !strings.Contains(errb.String(), "beta-err") {
		t.Fatalf("expected stderr passthrough, got:\n%s", errb.String())
	}
}

func TestRunContext_TailFileMirrorsToStdout(t *testing.T) {
	exe := buildTestApp(t, "file-log")
	tempDir := repoTempDir(t)
	logFile := filepath.Join(tempDir, "app.log")
	pidFile := filepath.Join(tempDir, "file-log.pid")

	var out, errb bytes.Buffer
	err := RunContext(context.Background(), Config{
		Command:     []string{exe, "--log-file", logFile, "--pid-file", pidFile, "--lines", "4", "--interval-ms", "100"},
		TailFiles:   []string{logFile},
		StopTimeout: 2 * time.Second,
		KillTree:    true,
		Verbose:     true,
	}, &out, &errb)
	if err != nil {
		t.Fatalf("expected file-log success, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "file-log line") || !strings.Contains(out.String(), "file-log line 4") {
		t.Fatalf("expected stdout to contain tailed file-log output, got:\n%s", out.String())
	}
}

func TestRunContext_PortRebindServerRestartsCleanly(t *testing.T) {
	exe := buildTestApp(t, "port-rebind-server")
	tempDir := repoTempDir(t)
	signalFile := filepath.Join(tempDir, "server.signal")
	pidFile := filepath.Join(tempDir, "server.pid")
	port := 18190
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out, errb bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunContext(ctx, Config{
			Command:      []string{exe, "--port", strconv.Itoa(port), "--signal-file", signalFile, "--pid-file", pidFile},
			GracefulStop: fmt.Sprintf("\"%s\" --send --signal-file \"%s\"", exe, signalFile),
			StopTimeout:  3 * time.Second,
			KillTree:     true,
			Verbose:      true,
		}, &out, &errb)
	}()

	pid := waitForPIDFile(t, pidFile, 5*time.Second)
	waitForProcessState(t, pid, true, 3*time.Second)
	waitForHTTPStatus(t, url, 200, 8*time.Second)
	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected graceful server shutdown success, got %v\nstdout:\n%s\nstderr:\n%s", err, out.String(), errb.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout waiting for graceful server shutdown\nstdout:\n%s\nstderr:\n%s", out.String(), errb.String())
	}
	waitForProcessState(t, pid, false, 5*time.Second)

	var out2, errb2 bytes.Buffer
	err := RunContext(context.Background(), Config{
		Command:     []string{exe, "--port", strconv.Itoa(port), "--signal-file", filepath.Join(tempDir, "server2.signal"), "--pid-file", filepath.Join(tempDir, "server2.pid"), "--send"},
		StopTimeout: 2 * time.Second,
		KillTree:    true,
		Verbose:     true,
	}, &out2, &errb2)
	var ee *ExitCodeError
	if err != nil && !errors.As(err, &ee) {
		t.Fatalf("unexpected restart probe error: %v\nstdout:\n%s\nstderr:\n%s", err, out2.String(), errb2.String())
	}
	listener, listenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if listenErr != nil {
		t.Fatalf("expected port %d to be reusable after shutdown, listen failed: %v", port, listenErr)
	}
	_ = listener.Close()
}

func buildTestApp(t *testing.T, name string) string {
	t.Helper()
	root := findRepoRoot(t)
	out := filepath.Join(repoTempDir(t), name+".exe")
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

func repoTempDir(t *testing.T) string {
	t.Helper()
	root := findRepoRoot(t)
	base := filepath.Join(root, "tmp", "tests")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir repo temp base: %v", err)
	}
	dir, err := os.MkdirTemp(base, sanitizeTestName(t.Name())+"-")
	if err != nil {
		t.Fatalf("create repo temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", " ", "-")
	return replacer.Replace(strings.ToLower(name))
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

func waitForFileExists(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for file %s", path)
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

func waitForHTTPStatus(t *testing.T, url string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == want {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for http %d from %s", want, url)
}
