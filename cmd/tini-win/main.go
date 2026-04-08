package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/service-lasso/tini-win/internal/app"
	"github.com/service-lasso/tini-win/internal/runner"
)

func main() {
	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var ee *runner.ExitCodeError
		if errors.As(err, &ee) {
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
