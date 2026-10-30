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

const usage = `numen-cli — notes with typed links

usage:
  numen-cli vault add <path> [--name <name>]   give a folder an identity and remember it
  numen-cli vault list                         show the vaults this installation knows
  numen-cli vault rename <vault> <new name>    call a vault something else
  numen-cli vault remove <vault>               take a vault off the list, leaving its folder
  numen-cli vault erase <vault> [--yes]        remove it, and put its folder in the trash
  numen-cli vault open <vault>                 the vault the next window opens
  numen-cli scan <vault> [--rebuild-index]     bring the index up to date with a vault
                                               --rebuild-index reads every file again, forgets
                                               the vectors of every model but the one in use,
                                               and gives their space back
  numen-cli search <vault> <query>             full-text search within one vault
  numen-cli links <vault> <note>               what a note points at, and what points at it
  numen-cli problems <vault> [<check>...]      what the vault holds that was not guessed at

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
	if len(rest) == 0 {
		if err := writeUsage(out, deps); err != nil {
			return err
		}
		return errors.New("no command given")
	}

	switch rest[0] {
	case "vault":
		return runVaultCommand(ctx, out, deps, rest[1:])
	case "scan":
		return runScanCommand(ctx, out, deps, rest[1:])
	case "search":
		return runSearchCommand(ctx, out, errOut, deps, rest[1:])
	case "links":
		return runLinksCommand(ctx, out, deps, rest[1:])
	case "problems":
		return runProblemsCommand(ctx, out, deps, rest[1:])
	case "help", "-h", "--help":
		return writeUsage(out, deps)
	default:
		if entry, isDeclared := findPluginCommand(deps, rest[0]); isDeclared {
			return runPluginCommand(ctx, out, deps, entry, rest[1:])
		}
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

// writeUsage writes the usage text and the commands the enabled plugins declare.
func writeUsage(out io.Writer, deps Deps) error {
	r := newReporter(out)
	r.Print(usage)
	writePluginCommandUsage(r, deps)
	return r.Err()
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
