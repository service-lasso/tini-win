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
	mode := flag.String("mode", "simple-exit", "simple-exit|graceful-stop|ignore-stop|spawn-child")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	childPIDFile := flag.String("child-pid-file", "", "optional file to write spawned child pid")
	signalFile := flag.String("signal-file", "", "signal file path for graceful-stop mode")
	sleepMs := flag.Int("sleep-ms", 0, "delay before exit for simple-exit mode")
	duration := flag.Int("duration", 30, "seconds to stay alive for spawn-child mode")
	exitCode := flag.Int("exit-code", 0, "exit code for simple-exit mode")
	send := flag.Bool("send", false, "send graceful-stop signal")
	flag.Parse()

	writePID(*pidFile, os.Getpid())

	switch *mode {
	case "simple-exit":
		fmt.Println("go-edgecase: simple-exit starting")
		if *sleepMs > 0 {
			time.Sleep(time.Duration(*sleepMs) * time.Millisecond)
		}
		fmt.Printf("go-edgecase: exiting %d\n", *exitCode)
		os.Exit(*exitCode)
	case "graceful-stop":
		if *signalFile == "" {
			fail("--signal-file is required for graceful-stop mode", 2)
		}
		if *send {
			if err := os.WriteFile(*signalFile, []byte("stop"), 0o644); err != nil {
				fail("failed to write signal file: "+err.Error(), 3)
			}
			fmt.Println("go-edgecase: graceful signal sent")
			return
		}
		fmt.Println("go-edgecase: graceful-stop running")
		for {
			if _, err := os.Stat(*signalFile); err == nil {
				fmt.Println("go-edgecase: graceful-stop detected signal, exiting 0")
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
	case "ignore-stop":
		fmt.Println("go-edgecase: ignore-stop running")
		for {
			time.Sleep(2 * time.Second)
			fmt.Println("go-edgecase: still alive")
		}
	case "spawn-child":
		fmt.Println("go-edgecase: spawn-child starting")
		cmd := exec.Command("cmd", "/c", "ping 127.0.0.1 -n "+strconv.Itoa(*duration)+" >nul")
		if err := cmd.Start(); err != nil {
			fail("failed to start child: "+err.Error(), 4)
		}
		writePID(*childPIDFile, cmd.Process.Pid)
		fmt.Printf("go-edgecase: spawned pid=%d\n", cmd.Process.Pid)
		time.Sleep(time.Duration(*duration) * time.Second)
	default:
		fail("unknown mode: "+*mode, 2)
	}
}

func writePID(path string, pid int) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func fail(msg string, code int) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
