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
  numen-cli vault add <path> [--name <name>]   add a folder to the vault list
  numen-cli vault list                         show the vaults this installation knows
  numen-cli vault rename <vault> <new name>    change the name of a vault
  numen-cli vault remove <vault>               take a vault off the list, leaving its folder
  numen-cli vault erase <vault> [--yes]        remove it and move its folder to the trash
  numen-cli vault open <vault>                 set the vault the next window opens
  numen-cli scan <vault> [--rebuild-index]     update the index of a vault
                                               --rebuild-index reads every file again and deletes the vectors
                                               of every model except the one in use
  numen-cli search <vault> <query>             full-text search within one vault
  numen-cli links <vault> <note>               what a note points at, and what points at it
  numen-cli problems <vault> [<check>...]      list notes with broken links, invalid frontmatter or ambiguous links

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

	if cmd, ok := getCLICommands()[rest[0]]; ok {
		return formatVaultError(cmd(ctx, out, errOut, deps, rest[1:]))
	}
	return formatVaultError(runCommandOfPlugins(ctx, out, deps, rest))
}

type cliCommand func(ctx context.Context, out, errOut io.Writer, deps Deps, args []string) error

func getCLICommands() map[string]cliCommand {
	help := func(_ context.Context, out, _ io.Writer, deps Deps, _ []string) error {
		return writeUsage(out, deps)
	}
	return map[string]cliCommand{
		"vault": func(ctx context.Context, out, _ io.Writer, deps Deps, args []string) error {
			return runVaultCommand(ctx, out, deps, args)
		},
		"scan": func(ctx context.Context, out, _ io.Writer, deps Deps, args []string) error {
			return runScanCommand(ctx, out, deps, args)
		},
		"search": runSearchCommand,
		"links": func(ctx context.Context, out, _ io.Writer, deps Deps, args []string) error {
			return runLinksCommand(ctx, out, deps, args)
		},
		"problems": func(ctx context.Context, out, _ io.Writer, deps Deps, args []string) error {
			return runProblemsCommand(ctx, out, deps, args)
		},
		"help":   help,
		"-h":     help,
		"--help": help,
	}
}

func runCommandOfPlugins(ctx context.Context, out io.Writer, deps Deps, rest []string) error {
	if handlers := listSharedHandlers(deps, rest[0]); len(handlers) > 0 {
		return runSharedCommand(ctx, out, deps, handlers, rest[1:])
	}
	if entry, isDeclared := findPluginCommand(deps, rest[0]); isDeclared {
		return runPluginCommand(ctx, out, deps, entry, rest[1:])
	}
	return fmt.Errorf("unknown command %q", rest[0])
}

func writeUsage(out io.Writer, deps Deps) error {
	r := newReporter(out)
	r.Print(usage)
	writePluginCommandUsage(r, deps)
	writeSharedCommandUsage(r, deps)
	return r.Err()
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
