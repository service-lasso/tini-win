//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procFreeConsole           = kernel32.NewProc("FreeConsole")
	procAttachConsole         = kernel32.NewProc("AttachConsole")
	procSetConsoleCtrlHandler = kernel32.NewProc("SetConsoleCtrlHandler")
	procGenerateCtrlEvent     = kernel32.NewProc("GenerateConsoleCtrlEvent")
)

const ctrlBreakEvent = 1

func main() {
	mode := flag.String("mode", "run", "run or send-break")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	eventFile := flag.String("event-file", "", "optional file written when a control event is received")
	targetPID := flag.Int("target-pid", 0, "target process group for send-break mode")
	duration := flag.Int("duration", 30, "seconds to wait before timing out in run mode")
	flag.Parse()

	switch *mode {
	case "run":
		writePID(*pidFile, os.Getpid())
		runTrap(*eventFile, *duration)
	case "send-break":
		if *targetPID <= 0 {
			fmt.Fprintln(os.Stderr, "--target-pid is required for send-break mode")
			os.Exit(2)
		}
		if err := sendCtrlBreak(*targetPID); err != nil {
			fmt.Fprintln(os.Stderr, "failed to send ctrl-break:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(2)
	}
}

func runTrap(eventFile string, duration int) {
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt)
	defer signal.Stop(ch)

	deadline := time.After(time.Duration(duration) * time.Second)
	for {
		select {
		case sig := <-ch:
			writeEvent(eventFile, sig.String())
			fmt.Println("console-trap: received control event", sig)
			return
		case <-deadline:
			writeEvent(eventFile, "timeout")
			fmt.Println("console-trap: timed out waiting for control event")
			return
		}
	}
}

func sendCtrlBreak(targetPID int) error {
	_, _, _ = procFreeConsole.Call()
	if r1, _, err := procAttachConsole.Call(uintptr(targetPID)); r1 == 0 {
		return err
	}
	defer procFreeConsole.Call()
	procSetConsoleCtrlHandler.Call(0, 1)
	if r1, _, err := procGenerateCtrlEvent.Call(ctrlBreakEvent, uintptr(targetPID)); r1 == 0 {
		return err
	}
	return nil
}

func writePID(path string, pid int) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func writeEvent(path, value string) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(value), 0o644)
}
