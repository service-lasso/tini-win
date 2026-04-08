package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type waitResult struct {
	err  error
	code int
}

func Run(cfg Config, stdout, stderr io.Writer) error {
	return RunContext(context.Background(), cfg, stdout, stderr)
}

func RunContext(ctx context.Context, cfg Config, stdout, stderr io.Writer) error {
	if len(cfg.Command) == 0 {
		return errors.New("missing child command")
	}
	if cfg.StopTimeout <= 0 {
		cfg.StopTimeout = 15 * time.Second
	}

	if cfg.Verbose {
		fmt.Fprintf(stdout, "[tini-win] starting child: %s\n", strings.Join(cfg.Command, " "))
	}

	child := exec.Command(cfg.Command[0], cfg.Command[1:]...)
	child.Stdout = stdout
	child.Stderr = stderr
	setChildProcessAttrs(child)

	if err := child.Start(); err != nil {
		return fmt.Errorf("start child: %w", err)
	}

	job, err := createAndAssignJobObject(uint32(child.Process.Pid), cfg.KillTree)
	if err != nil {
		_ = child.Process.Kill()
		_ = child.Wait()
		return fmt.Errorf("assign job object: %w", err)
	}
	defer closeJobObject(job)

	waitCh := make(chan waitResult, 1)
	go func() {
		err := child.Wait()
		code := 0
		if err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				code = ee.ExitCode()
			}
		}
		waitCh <- waitResult{err: err, code: code}
	}()

	forcedStop := false
	var stopOnce sync.Once
	stopChild := func(reason string) {
		stopOnce.Do(func() {
			if cfg.Verbose {
				fmt.Fprintf(stderr, "[tini-win] stop requested: %s\n", reason)
			}

			if cfg.GracefulStop != "" {
				if cfg.Verbose {
					fmt.Fprintf(stderr, "[tini-win] running graceful stop command: %s\n", cfg.GracefulStop)
				}
				if err := runGracefulStop(cfg.GracefulStop, cfg.StopTimeout, stdout, stderr); err != nil && cfg.Verbose {
					fmt.Fprintf(stderr, "[tini-win] graceful stop command failed: %v\n", err)
				}
			}

			select {
			case res := <-waitCh:
				waitCh <- res
				return
			case <-time.After(cfg.StopTimeout):
				forcedStop = true
				if cfg.Verbose {
					fmt.Fprintln(stderr, "[tini-win] graceful stop timeout reached, forcing tree termination")
				}
				if cfg.KillTree {
					_ = terminateJobObject(job, 1)
				} else {
					_ = child.Process.Kill()
				}
			}
		})
	}

	for {
		select {
		case <-ctx.Done():
			stopChild("context canceled")
			return finalizeWaitResult(<-waitCh, forcedStop, cfg, stdout)
		case res := <-waitCh:
			return finalizeWaitResult(res, forcedStop, cfg, stdout)
		}
	}
}

func finalizeWaitResult(res waitResult, forcedStop bool, cfg Config, stdout io.Writer) error {
	if res.err != nil {
		var ee *exec.ExitError
		if !errors.As(res.err, &ee) {
			return fmt.Errorf("wait child: %w", res.err)
		}
	}

	code := res.code
	if forcedStop {
		code = 137
	}
	if cfg.Verbose {
		fmt.Fprintf(stdout, "[tini-win] child wait result: raw_code=%d forced_stop=%t\n", res.code, forcedStop)
	}
	if mapped, ok := cfg.RemapExitCode[code]; ok {
		code = mapped
	}
	if code == 0 {
		if cfg.Verbose {
			fmt.Fprintln(stdout, "[tini-win] child exited cleanly")
		}
		return nil
	}
	return &ExitCodeError{Code: code}
}

func runGracefulStop(command string, timeout time.Duration, stdout, stderr io.Writer) error {
	args, err := splitCommandLine(command)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("empty graceful stop command")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func splitCommandLine(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuotes := false

	flush := func() {
		if current.Len() > 0 {
			args = append(args, current.String())
			current.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '"':
			inQuotes = !inQuotes
		case ' ', '\t':
			if inQuotes {
				current.WriteByte(ch)
			} else {
				flush()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if inQuotes {
		return nil, errors.New("unterminated quote in graceful stop command")
	}
	flush()
	return args, nil
}
