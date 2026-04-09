package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/service-lasso/tini-win/internal/app"
	"github.com/service-lasso/tini-win/internal/runner"
)

var version = "dev"

func main() {
	app.Version = version

	if app.WantsHelp(os.Args[1:]) {
		app.WriteHelp(os.Stdout)
		return
	}
	if app.WantsVersion(os.Args[1:]) {
		fmt.Fprintln(os.Stdout, version)
		return
	}

	if err := app.Run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		var ee *runner.ExitCodeError
		if errors.As(err, &ee) {
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
