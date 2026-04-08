package app

import (
	"testing"
	"time"
)

func TestParseArgs_Minimal(t *testing.T) {
	cfg, err := ParseArgs([]string{"--", "cmd", "/c", "exit 0"})
	if err != nil {
		t.Fatalf("ParseArgs error: %v", err)
	}
	if len(cfg.Command) != 3 {
		t.Fatalf("expected command length 3, got %d", len(cfg.Command))
	}
	if cfg.StopTimeout != 15*time.Second {
		t.Fatalf("expected default timeout 15s, got %v", cfg.StopTimeout)
	}
	if !cfg.KillTree {
		t.Fatalf("expected killTree true by default")
	}
}

func TestParseArgs_Remap(t *testing.T) {
	cfg, err := ParseArgs([]string{"--remap-exit", "143:0,137:0", "--", "cmd", "/c", "exit 0"})
	if err != nil {
		t.Fatalf("ParseArgs error: %v", err)
	}
	if cfg.RemapExitCode[143] != 0 || cfg.RemapExitCode[137] != 0 {
		t.Fatalf("unexpected remap map: %#v", cfg.RemapExitCode)
	}
}

func TestParseArgs_MissingSeparator(t *testing.T) {
	_, err := ParseArgs([]string{"cmd", "/c", "exit 0"})
	if err == nil {
		t.Fatalf("expected error for missing -- separator")
	}
}

func TestParseArgs_InvalidStopTimeout(t *testing.T) {
	_, err := ParseArgs([]string{"--stop-timeout", "nope", "--", "cmd", "/c", "exit 0"})
	if err == nil {
		t.Fatalf("expected invalid stop-timeout error")
	}
}

func TestParseArgs_InvalidRemapPair(t *testing.T) {
	_, err := ParseArgs([]string{"--remap-exit", "143", "--", "cmd", "/c", "exit 0"})
	if err == nil {
		t.Fatalf("expected invalid remap pair error")
	}
}

func TestParseArgs_MissingCommandAfterSeparator(t *testing.T) {
	_, err := ParseArgs([]string{"--"})
	if err == nil {
		t.Fatalf("expected missing command after separator error")
	}
}
