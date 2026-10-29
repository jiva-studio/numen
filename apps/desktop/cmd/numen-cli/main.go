// Command numen-cli is the command line onto a vault.
//
// This file is the entry point and nothing else. The command line is one
// driving adapter; the window is another, and neither is the product.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/apps/desktop/internal/plugincli"
	"github.com/jiva-studio/numen/libs/core/adapter/cli"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "plugin" {
		os.Exit(plugincli.Run(os.Args[2:], os.Stdout, os.Stderr))
	}

	os.Exit(run())
}

func run() (code int) {
	// The settings are read here only: nothing further down can access the
	// machine's own settings file.
	cfg := platform.GetConfig()
	// A run whose plugins did not stop reports it, and fails if nothing else did.
	defer func() {
		if err := cfg.StopPlugins(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "numen-cli: stop plugins:", err)
			if code == 0 {
				code = 1
			}
		}
	}()

	selected, err := cfg.Settings()
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-cli:", err)
		return 1
	}
	cfg = cfg.SetSettings(selected)
	return cli.Main(context.Background(), os.Stdout, os.Stderr, os.Args[1:], makeDeps(cfg))
}
