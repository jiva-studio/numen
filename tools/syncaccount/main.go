// Command syncaccount works on the accounts of a real sync server for the acceptance scenarios: it makes an account the way
// three computers of one person do and writes the keys the application under test needs, lists the devices an account holds
// and looks for a text in what the server stores. The server is the test stack of tests/server or the stand of the owner.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"
)

const (
	commandTimeout = 2 * time.Minute
	usage          = "usage: syncaccount seed|list-devices|find-text -help"
)

func main() { os.Exit(execute()) }

func execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "syncaccount:", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	switch args[0] {
	case "seed":
		return runSeed(ctx, args[1:], out)
	case "list-devices":
		return runListDevices(ctx, args[1:], out)
	case "find-text":
		return runFindText(ctx, args[1:], out)
	}
	return fmt.Errorf("unknown command %q: %s", args[0], usage)
}

func closeQuietly(closer io.Closer) {
	if err := closer.Close(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "close:", err)
	}
}
