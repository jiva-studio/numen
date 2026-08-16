// Command numen-cli is the command line onto a vault.
//
// This file is the entry point and nothing else. The command line is one
// driving adapter; the window is another, and neither is the product.
package main

import (
	"context"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/cli"
)

func main() {
	os.Exit(cli.Main(context.Background(), os.Stdout, os.Stderr, os.Args[1:]))
}
