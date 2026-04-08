package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	signalFile := flag.String("signal-file", "", "path to signal file")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	send := flag.Bool("send", false, "send graceful stop signal")
	flag.Parse()

	if *signalFile == "" {
		fmt.Fprintln(os.Stderr, "--signal-file is required")
		os.Exit(2)
	}

	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	if *send {
		f, err := os.Create(*signalFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to write signal file:", err)
			os.Exit(3)
		}
		_ = f.Close()
		fmt.Println("graceful-stop: signal sent")
		os.Exit(0)
	}

	fmt.Println("graceful-stop: running")
	for {
		if _, err := os.Stat(*signalFile); err == nil {
			fmt.Println("graceful-stop: detected signal file, exiting 0")
			os.Exit(0)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
