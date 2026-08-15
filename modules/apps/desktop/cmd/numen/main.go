// Command numen is the desktop application.
//
// This file is the entry point and nothing else. The command line is one
// driving adapter; a graphical shell will be another, and neither is the
// product.
package main

import (
	"context"
	"os"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/cli"
)

func main() {
	os.Exit(cli.Main(context.Background(), os.Stdout, os.Stderr, os.Args[1:]))
}
