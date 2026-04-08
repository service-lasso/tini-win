package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	sleepMs := flag.Int("sleep-ms", 0, "optional milliseconds to wait before exiting")
	flag.Parse()

	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	fmt.Println("simple-exit: starting")
	if *sleepMs > 0 {
		time.Sleep(time.Duration(*sleepMs) * time.Millisecond)
	}
	fmt.Println("simple-exit: exiting 0")
	os.Exit(0)
}
