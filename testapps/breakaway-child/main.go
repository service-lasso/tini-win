//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const createBreakawayFromJob = 0x01000000

func main() {
	duration := flag.Int("duration", 30, "seconds to keep the spawned child alive")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	childPIDFile := flag.String("child-pid-file", "", "optional file to write spawned child pid")
	statusFile := flag.String("status-file", "", "optional file to record breakaway spawn result")
	flag.Parse()

	writePID(*pidFile, os.Getpid())
	cmd := exec.Command("cmd", "/c", "ping 127.0.0.1 -n "+strconv.Itoa(*duration)+" >nul")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createBreakawayFromJob | syscall.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		writeStatus(*statusFile, "spawn-error:"+err.Error())
		fmt.Println("breakaway-child: breakaway spawn failed:", err)
		time.Sleep(2 * time.Second)
		return
	}

	writePID(*childPIDFile, cmd.Process.Pid)
	writeStatus(*statusFile, "spawned")
	fmt.Printf("breakaway-child: spawned pid=%d with breakaway flags\n", cmd.Process.Pid)
	time.Sleep(time.Duration(*duration) * time.Second)
}

func writePID(path string, pid int) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func writeStatus(path, value string) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(value), 0o644)
}
