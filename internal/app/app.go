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
	fs.StringVar(&cfg.GracefulStop, "graceful-stop", "", "graceful stop command")
	fs.StringVar(&stopTimeout, "stop-timeout", "15s", "graceful stop timeout")
	fs.BoolVar(&cfg.KillTree, "kill-tree", true, "kill process tree on forced stop")
	fs.BoolVar(&cfg.AllowBreakaway, "allow-breakaway", false, "allow explicit child breakaway from the job object")
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
	cfg.Command = append([]string{}, args[idx+1:]...)

	return cfg, nil
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
