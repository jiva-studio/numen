// Command numen-cli is the command line onto a vault.
//
// This file is the entry point and nothing else. The command line is one
// driving adapter; the window is another, and neither is the product.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

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

func run() int {
	// The settings are read here and nowhere below: an entry point says what it
	// found, and nothing further down can reach the machine's own file.
	cfg := platform.GetConfig()
	if cfg.PluginHost != nil {
		defer func() {
			if host, ok := cfg.PluginHost.(interface {
				TeardownAll(context.Context, time.Duration) error
			}); ok {
				_ = host.TeardownAll(context.Background(), 2*time.Second)
			}
		}()
	}

	chosen, err := cfg.Settings()
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-cli:", err)
		return 1
	}
	cfg = cfg.SetSettings(chosen)
	return cli.Main(context.Background(), os.Stdout, os.Stderr, os.Args[1:], makeDeps(cfg))
}
