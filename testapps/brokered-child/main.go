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
	mode := flag.String("mode", "client", "broker|client")
	requestFile := flag.String("request-file", "", "path used to request broker spawn")
	stopFile := flag.String("stop-file", "", "path used to stop broker")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	childPIDFile := flag.String("child-pid-file", "", "optional file to write broker-spawned child pid")
	duration := flag.Int("duration", 30, "seconds broker-spawned child should stay alive")
	flag.Parse()

	writePID(*pidFile, os.Getpid())

	switch *mode {
	case "broker":
		runBroker(*requestFile, *stopFile, *childPIDFile, *duration)
	case "client":
		runClient(*requestFile)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(2)
	}
}

func runBroker(requestFile, stopFile, childPIDFile string, duration int) {
	if requestFile == "" || stopFile == "" {
		fmt.Fprintln(os.Stderr, "--request-file and --stop-file are required for broker mode")
		os.Exit(2)
	}
	fmt.Println("brokered-child: broker waiting for request")
	for {
		if _, err := os.Stat(stopFile); err == nil {
			fmt.Println("brokered-child: broker stopping")
			return
		}
		if _, err := os.Stat(requestFile); err == nil {
			cmd := exec.Command("cmd", "/c", "ping 127.0.0.1 -n "+strconv.Itoa(duration)+" >nul")
			if err := cmd.Start(); err != nil {
				fmt.Fprintln(os.Stderr, "broker failed to start child:", err)
				os.Exit(3)
			}
			writePID(childPIDFile, cmd.Process.Pid)
			fmt.Printf("brokered-child: broker spawned pid=%d\n", cmd.Process.Pid)
			for {
				if _, err := os.Stat(stopFile); err == nil {
					fmt.Println("brokered-child: broker stopping after spawn")
					return
				}
				time.Sleep(250 * time.Millisecond)
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func runClient(requestFile string) {
	if requestFile == "" {
		fmt.Fprintln(os.Stderr, "--request-file is required for client mode")
		os.Exit(2)
	}
	_ = os.WriteFile(requestFile, []byte("spawn"), 0o644)
	fmt.Println("brokered-child: client requested broker spawn")
	for {
		time.Sleep(2 * time.Second)
	}
}

func writePID(path string, pid int) {
	if path == "" {
		return
	}
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}
