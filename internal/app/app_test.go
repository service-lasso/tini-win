package app

import (
	"bytes"
	"strings"
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

func TestWantsHelp(t *testing.T) {
	if !WantsHelp([]string{"--help"}) || !WantsHelp([]string{"-h"}) {
		t.Fatalf("expected help flags to be recognized")
	}
	if WantsHelp([]string{"--", "--help"}) {
		t.Fatalf("did not expect help after separator to be treated as app help")
	}
}

func TestWantsVersion(t *testing.T) {
	if !WantsVersion([]string{"--version"}) {
		t.Fatalf("expected version flag to be recognized")
	}
	if WantsVersion([]string{"--", "--version"}) {
		t.Fatalf("did not expect version after separator to be treated as app version")
	}
}

func TestWriteHelp(t *testing.T) {
	var buf bytes.Buffer
	WriteHelp(&buf)
	out := buf.String()
	for _, needle := range []string{"Usage:", "--version", "--help", "--graceful-stop"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("expected help output to contain %q, got %q", needle, out)
		}
	}
}
