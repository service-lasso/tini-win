package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/service-lasso/tini-win/internal/runner"
)

var Version = "dev"

func WantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func WantsVersion(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--version" {
			return true
		}
	}
	return false
}

func WriteHelp(w io.Writer) {
	fmt.Fprint(w, "tini-win - Windows-native tiny process babysitter\n\n")
	fmt.Fprint(w, "Usage:\n")
	fmt.Fprint(w, "  tini-win [OPTIONS] -- PROGRAM [ARGS...]\n\n")
	fmt.Fprint(w, "Options:\n")
	fmt.Fprint(w, "  -h, --help             Show help\n")
	fmt.Fprint(w, "      --version          Show version\n")
	fmt.Fprint(w, "      --graceful-stop    Command to run for graceful shutdown\n")
	fmt.Fprint(w, "      --stop-timeout     Graceful stop timeout (default 15s)\n")
	fmt.Fprint(w, "      --kill-tree        Kill process tree on forced stop (default true)\n")
	fmt.Fprint(w, "      --allow-breakaway  Allow explicit child breakaway from the job object\n")
	fmt.Fprint(w, "      --tail-file        Tail a log file to stdout (repeatable)\n")
	fmt.Fprint(w, "      --remap-exit       Remap exit codes, e.g. 143:0,137:0\n")
	fmt.Fprint(w, "  -v                     Verbose logs\n")
}

func Run(args []string, stdout, stderr io.Writer) error {
	cfg, err := ParseArgs(args)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return runner.RunContext(ctx, cfg, stdout, stderr)
}

func ParseArgs(args []string) (runner.Config, error) {
	var cfg runner.Config
	cfg.StopTimeout = 15 * time.Second
	cfg.KillTree = true
	cfg.RemapExitCode = map[int]int{}

	idx := indexOf(args, "--")
	if idx < 0 || idx == len(args)-1 {
		return cfg, errors.New("usage: tini-win [options] -- program [args...]")
	}

	fs := flag.NewFlagSet("tini-win", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var stopTimeout string
	var remap string
	var tailFiles multiValueFlag
	fs.StringVar(&cfg.GracefulStop, "graceful-stop", "", "graceful stop command")
	fs.StringVar(&stopTimeout, "stop-timeout", "15s", "graceful stop timeout")
	fs.BoolVar(&cfg.KillTree, "kill-tree", true, "kill process tree on forced stop")
	fs.BoolVar(&cfg.AllowBreakaway, "allow-breakaway", false, "allow explicit child breakaway from the job object")
	fs.Var(&tailFiles, "tail-file", "tail a log file to stdout (repeatable)")
	fs.BoolVar(&cfg.Verbose, "v", false, "verbose logs")
	fs.StringVar(&remap, "remap-exit", "", "remap exit codes, e.g. 143:0,137:0")

	if err := fs.Parse(args[:idx]); err != nil {
		return cfg, err
	}

	d, err := time.ParseDuration(stopTimeout)
	if err != nil {
		return cfg, fmt.Errorf("invalid --stop-timeout: %w", err)
	}
	cfg.StopTimeout = d

	m, err := parseRemap(remap)
	if err != nil {
		return cfg, err
	}
	cfg.RemapExitCode = m
	cfg.TailFiles = append(cfg.TailFiles, tailFiles...)
	cfg.Command = append([]string{}, args[idx+1:]...)

	return cfg, nil
}

type multiValueFlag []string

func (m *multiValueFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiValueFlag) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("tail-file must not be empty")
	}
	*m = append(*m, value)
	return nil
}

func parseRemap(v string) (map[int]int, error) {
	out := map[int]int{}
	if strings.TrimSpace(v) == "" {
		return out, nil
	}
	parts := strings.Split(v, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.Split(p, ":")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid remap pair %q", p)
		}
		from, err := strconv.Atoi(strings.TrimSpace(kv[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid remap from %q: %w", kv[0], err)
		}
		to, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid remap to %q: %w", kv[1], err)
		}
		out[from] = to
	}
	return out, nil
}

func indexOf(arr []string, needle string) int {
	for i, v := range arr {
		if v == needle {
			return i
		}
	}
	return -1
}
