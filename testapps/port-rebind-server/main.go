package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	port := flag.Int("port", 18090, "port to bind")
	signalFile := flag.String("signal-file", "", "path to shutdown signal file")
	pidFile := flag.String("pid-file", "", "optional file to write this process pid")
	send := flag.Bool("send", false, "send graceful stop signal")
	flag.Parse()

	if *signalFile == "" {
		fmt.Fprintln(os.Stderr, "--signal-file is required")
		os.Exit(2)
	}

	if *send {
		if err := os.WriteFile(*signalFile, []byte("stop"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "failed to write signal file:", err)
			os.Exit(3)
		}
		fmt.Println("port-rebind-server: signal sent")
		return
	}

	if *pidFile != "" {
		_ = os.WriteFile(*pidFile, []byte(strconv.Itoa(os.Getpid())), 0o644)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{Addr: ":" + strconv.Itoa(*port), Handler: mux}
	go func() {
		for {
			if _, err := os.Stat(*signalFile); err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
				return
			}
			time.Sleep(150 * time.Millisecond)
		}
	}()

	fmt.Printf("port-rebind-server: listening on %d\n", *port)
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "listen failed:", err)
		os.Exit(4)
	}
}
