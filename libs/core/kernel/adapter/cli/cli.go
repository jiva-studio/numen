// Package cli is a driving adapter: it converts arguments into calls on the core
// and results into text. A graphical shell would be another such adapter; the core
// depends on neither.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

const usageHead = `numen-cli — notes with typed links

usage:
`

const usageTail = `
A vault is named by its name, its path, or its identity.

options:
  --index <path>        where the index lives (default: platform cache directory)
  --registry <path>     where the vault list lives (default: platform config directory)
  --service-dir <name>  the folder a vault keeps its identity in (default: .numen)
`

// Main runs the command line and returns a process exit code. open supplies the
// installation this run works with, once the locations given on the command line
// have been read from the arguments: the application assembles an installation, and
// open requests one.
func Main(ctx context.Context, out, errOut io.Writer, args []string,
	open func(Locations) Deps,
) int {
	if err := Run(ctx, out, errOut, args, open); err != nil {
		if _, writeErr := fmt.Fprintln(errOut, "numen-cli:", err); writeErr != nil {
			return 2
		}
		return 1
	}
	return 0
}

// Run is Main with its output injected and errors returned, so the output is
// testable. errOut carries what a command reports besides its result.
func Run(ctx context.Context, out, errOut io.Writer, args []string,
	open func(Locations) Deps,
) error {
	var where Locations
	fs := flag.NewFlagSet("numen-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&where.Index, "index", "", "path to the index database")
	fs.StringVar(&where.Registry, "registry", "", "path to the vault list")
	// Empty is the folder this installation writes into a vault by default,
	// which is the answer for every other location here too.
	fs.StringVar(&where.ServiceDir, "service-dir", "", "vault service folder")
	fs.BoolVar(&where.SafeMode, "safe-mode", false, "disable plugins for emergency recovery")

	// A plain parse: it stops at the first argument that is not a flag, which
	// is the command. Anything after that belongs to the command and is parsed
	// by it, `--name` in `vault add <path> --name x` included.
	if err := fs.Parse(args); err != nil {
		return err
	}
	deps := open(where)

	rest := fs.Args()
	commands := newCommandRegistry(deps)
	if len(rest) == 0 {
		if err := writeUsage(out, commands); err != nil {
			return err
		}
		return errors.New("no command given")
	}

	one, isKnown := commands.find(rest[0])
	if !isKnown {
		return fmt.Errorf("unknown command %q", rest[0])
	}
	return formatVaultError(one.Run(ctx, terminal{out: out, errOut: errOut}, rest[1:]))
}

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
