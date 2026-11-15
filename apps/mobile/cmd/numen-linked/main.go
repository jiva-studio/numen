// Command numen-linked serves the core with Go plugins linked into its own binary and served inside its process, the
// way the phone will run them: a phone starts no plugin process. It runs on a computer, and the plugins it links
// declare the editor window as their target, since none declares the phone yet. It opens the vault in the folder
// vault/ of the folder it is given, keeps the index and the settings beside it, and writes the address it serves on
// and the token every request carries as a bearer credential.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	if len(args) != 1 {
		writeLine(errOut, "usage: numen-linked <folder>")
		return 2
	}
	core, err := startCore(args[0], errOut)
	if err != nil {
		writeLine(errOut, "numen-linked:", err)
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	writeLine(out, core.url, core.token)
	<-ctx.Done()
	if err := core.Close(); err != nil {
		writeLine(errOut, "numen-linked:", err)
		return 1
	}
	return 0
}

func writeLine(to io.Writer, parts ...any) {
	if _, err := fmt.Fprintln(to, parts...); err != nil {
		return
	}
}
