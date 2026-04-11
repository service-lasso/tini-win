//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	logFile := flag.String("log-file", "", "file to append log lines to")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	lines := flag.Int("lines", 3, "number of lines to write")
	intervalMs := flag.Int("interval-ms", 200, "delay between lines")
	flag.Parse()

	if *logFile == "" {
		fmt.Fprintln(os.Stderr, "--log-file is required")
		os.Exit(2)
	}
	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}
	_ = os.MkdirAll(filepath.Dir(*logFile), 0o755)

	f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open log file:", err)
		os.Exit(1)
	}
	defer f.Close()

	for i := 1; i <= *lines; i++ {
		fmt.Fprintf(f, "file-log line %d\n", i)
		_ = f.Sync()
		time.Sleep(time.Duration(*intervalMs) * time.Millisecond)
	}
}
