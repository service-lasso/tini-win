//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	stdoutText := flag.String("stdout", "stdout-line", "text to write to stdout")
	stderrText := flag.String("stderr", "stderr-line", "text to write to stderr")
	exitCode := flag.Int("exit-code", 0, "exit code")
	flag.Parse()

	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	fmt.Fprintln(os.Stdout, *stdoutText)
	fmt.Fprintln(os.Stderr, *stderrText)
	os.Exit(*exitCode)
}
