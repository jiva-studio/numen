// Package cli is a driving adapter: it turns arguments into calls on the core
// and results into text. A graphical shell will be another such adapter, and
// the core knows about neither.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

const usage = `numen — notes with typed links and spaced repetition

usage:
  numen vault add <path> [--name <name>]   give a folder an identity and remember it
  numen vault list                         show the vaults this installation knows
  numen scan <vault>                       bring the index up to date with a vault
  numen search <vault> <query>             full-text search within one vault

A vault is named by its name, its path, or its identity.

options:
  --index <path>        where the index lives (default: platform cache directory)
  --registry <path>     where the vault list lives (default: platform config directory)
  --service-dir <name>  the folder a vault keeps its identity in (default: .numen)
`

// Main runs the command line and returns a process exit code.
func Main(ctx context.Context, out, errOut io.Writer, args []string) int {
	if err := Run(ctx, out, args); err != nil {
		fmt.Fprintln(errOut, "numen:", err)
		return 1
	}
	return 0
}

// Run is Main with its output injected and errors returned, so that what the
// user sees is testable rather than only observable by hand.
func Run(ctx context.Context, out io.Writer, args []string) error {
	var cfg container.Config
	fs := flag.NewFlagSet("numen", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	fs.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	fs.StringVar(&cfg.ServiceDir, "service-dir", filesystem.DefaultServiceDir, "vault service folder")

	// Deliberately a plain parse: it stops at the first argument that is not a
	// flag, which is the command. Anything after that belongs to the command and
	// is parsed by it — otherwise `vault add <path> --name x` would be read as a
	// global flag that does not exist.
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprint(out, usage)
		return errors.New("no command given")
	}

	switch rest[0] {
	case "vault":
		return vaultCommand(ctx, out, cfg, rest[1:])
	case "scan":
		return scanCommand(ctx, out, cfg, rest[1:])
	case "search":
		return searchCommand(ctx, out, cfg, rest[1:])
	case "help", "-h", "--help":
		fmt.Fprint(out, usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

// parseInterspersed collects positional arguments wherever they appear.
//
// The standard flag package stops at the first non-flag argument, which makes
// `vault add <path> --name x` silently drop the flag: the order people type is
// not the order the parser expects.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}
