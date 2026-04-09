//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	mode := flag.String("mode", "parent", "parent or child")
	duration := flag.Int("duration", 30, "seconds to stay alive")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	childPIDFile := flag.String("child-pid-file", "", "optional file to write spawned child pid")
	flag.Parse()

	writePID(*pidFile, os.Getpid())

	switch *mode {
	case "parent":
		runParent(*duration, *childPIDFile)
	case "child":
		runChild(*duration)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(2)
	}
}

func runParent(duration int, childPIDFile string) {
	cmd := exec.Command(os.Args[0], "--mode", "child", "--duration", strconv.Itoa(duration))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to start stdio child:", err)
		os.Exit(1)
	}
	writePID(childPIDFile, cmd.Process.Pid)
	fmt.Printf("stdio-hold-open: parent spawned child pid=%d with inherited stdio\n", cmd.Process.Pid)
	time.Sleep(time.Duration(duration) * time.Second)
}

func runChild(duration int) {
	deadline := time.Now().Add(time.Duration(duration) * time.Second)
	for time.Now().Before(deadline) {
		fmt.Println("stdio-hold-open: child keeping stdio handles open")
		time.Sleep(1 * time.Second)
	}
}

func writePID(path string, pid int) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}
