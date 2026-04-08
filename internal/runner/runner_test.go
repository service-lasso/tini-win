package runner

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestRun_ExitCodeRemap(t *testing.T) {
	var out, errb bytes.Buffer
	cfg := Config{
		Command:       []string{"cmd", "/c", "exit 143"},
		StopTimeout:   2 * time.Second,
		KillTree:      true,
		RemapExitCode: map[int]int{143: 0},
	}
	err := Run(cfg, &out, &errb)
	if err != nil {
		t.Fatalf("expected remapped success, got error: %v", err)
	}
}

func TestRun_FailingExitCode(t *testing.T) {
	var out, errb bytes.Buffer
	cfg := Config{
		Command:       []string{"cmd", "/c", "exit 7"},
		StopTimeout:   2 * time.Second,
		KillTree:      true,
		RemapExitCode: map[int]int{},
	}
	err := Run(cfg, &out, &errb)
	var ee *ExitCodeError
	if !errors.As(err, &ee) {
		t.Fatalf("expected ExitCodeError, got %v", err)
	}
	if ee.Code != 7 {
		t.Fatalf("expected exit code 7, got %d", ee.Code)
	}
}
