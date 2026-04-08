package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func main() {
	duration := flag.Int("duration", 30, "seconds the relaunched child should stay alive")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	childPIDFile := flag.String("child-pid-file", "", "optional file to write relaunched child pid")
	flag.Parse()

	writePID(*pidFile, os.Getpid())
	cmd := exec.Command("cmd", "/c", "ping 127.0.0.1 -n "+strconv.Itoa(*duration)+" >nul")
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to spawn orphan candidate:", err)
		os.Exit(2)
	}
	writePID(*childPIDFile, cmd.Process.Pid)
	fmt.Printf("relaunch-orphan: spawned pid=%d and exiting parent immediately\n", cmd.Process.Pid)
}

func writePID(path string, pid int) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}
