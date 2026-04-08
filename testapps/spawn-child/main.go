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
	duration := flag.Int("duration", 30, "seconds to stay alive")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	childPIDFile := flag.String("child-pid-file", "", "optional file to write spawned child pid")
	flag.Parse()

	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	fmt.Println("spawn-child: parent starting")
	cmd := exec.Command("cmd", "/c", "ping 127.0.0.1 -n "+strconv.Itoa(*duration)+" >nul")
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to start child:", err)
		os.Exit(2)
	}
	fmt.Printf("spawn-child: spawned pid=%d\n", cmd.Process.Pid)
	if *childPIDFile != "" {
		_ = os.WriteFile(*childPIDFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
	}
	time.Sleep(time.Duration(*duration) * time.Second)
}
