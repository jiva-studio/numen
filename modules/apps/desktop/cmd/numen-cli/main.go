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
	"github.com/jiva-studio/numen/modules/libs/core/adapter/cli"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

func main() {
	// The settings are read here and nowhere below: an entry point says what it
	// found, and nothing further down can reach the machine's own file.
	chosen, err := settings.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "numen-cli:", err)
		os.Exit(1)
	}
	cfg := platform.Config().Indexing(chosen.Indexing)
	cfg.Fetching = chosen.Importing
	os.Exit(cli.Main(context.Background(), os.Stdout, os.Stderr, os.Args[1:], deps(cfg)))
}
