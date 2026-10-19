// Command numen-cli is the command line onto a vault.
//
// This file is the entry point and nothing else. The command line is one
// driving adapter; the window is another, and neither is the product.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/platform"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/plugincli"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "plugin" {
		os.Exit(plugincli.Run(os.Args[2:], os.Stdout, os.Stderr))
	}

	os.Exit(run())
}

func run() (code int) {
	// The settings are read here and nowhere below: an entry point says what it
	// found, and nothing further down can reach the machine's own file.
	cfg := platform.GetConfig()
	// A run whose plugins did not stop says so, and fails if nothing else did.
	defer func() {
		if err := cfg.StopPlugins(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "numen-cli: stop plugins:", err)
			if code == 0 {
				code = 1
			}
		}
	}()

	chosen, err := cfg.Settings()
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-cli:", err)
		return 1
	}
	cfg = cfg.SetSettings(chosen)
	return cli.Main(context.Background(), os.Stdout, os.Stderr, os.Args[1:], makeDeps(cfg))
}
