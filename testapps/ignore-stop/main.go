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
	flag.Parse()

	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	fmt.Println("ignore-stop: running")
	for {
		time.Sleep(2 * time.Second)
		fmt.Println("ignore-stop: still alive")
	}
}
