// Command numen-plugin is the command line tool for plugin development.
package main

import (
	"os"

	"github.com/jiva-studio/numen/apps/desktop/internal/plugincli"
)

func main() {
	os.Exit(plugincli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
