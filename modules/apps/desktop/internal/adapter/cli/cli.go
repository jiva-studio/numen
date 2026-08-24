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
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

const usage = `numen-cli — notes with typed links and spaced repetition

usage:
  numen-cli vault add <path> [--name <name>]   give a folder an identity and remember it
  numen-cli vault list                         show the vaults this installation knows
  numen-cli vault rename <vault> <new name>    call a vault something else
  numen-cli vault forget <vault>               take a vault off the list, leaving its folder
  numen-cli vault erase <vault> [--yes]        forget it, and put its folder in the trash
  numen-cli vault open <vault>                 the vault the next window opens
  numen-cli scan <vault> [--rebuild-index]      bring the index up to date with a vault
  numen-cli recognise <vault> <file>          read a scanned document with a model
  numen-cli proofread <vault> <file>          put a document's reading right with a model
  numen-cli search <vault> <query>             full-text search within one vault
  numen-cli links <vault> <note>               what a note points at, and what points at it
  numen-cli problems <vault> [<check>...]      what the vault holds that was not guessed at

A vault is named by its name, its path, or its identity.

options:
  --index <path>        where the index lives (default: platform cache directory)
  --registry <path>     where the vault list lives (default: platform config directory)
  --service-dir <name>  the folder a vault keeps its identity in (default: .numen)
  --note-extensions <list>  which files are notes (default: .md)
`

// Main runs the command line and returns a process exit code.
func Main(ctx context.Context, out, errOut io.Writer, args []string, indexing settings.Indexing) int {
	if err := Run(ctx, out, args, indexing); err != nil {
		fmt.Fprintln(errOut, "numen-cli:", err)
		return 1
	}
	return 0
}

// Run is Main with its output injected and errors returned, so what the person
// sees is testable.
func Run(ctx context.Context, out io.Writer, args []string, indexing settings.Indexing) error {
	cfg := container.Config{}.Indexing(indexing)
	fs := flag.NewFlagSet("numen-cli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.IndexPath, "index", "", "path to the index database")
	fs.StringVar(&cfg.RegistryPath, "registry", "", "path to the vault list")
	fs.StringVar(&cfg.ServiceDir, "service-dir", filesystem.DefaultServiceDir, "vault service folder")
	extensions := fs.String("note-extensions", strings.Join(filesystem.DefaultExtensions, ","),
		"comma-separated file extensions treated as notes")

	// A plain parse: it stops at the first argument that is not a flag, which
	// is the command. Anything after that belongs to the command and is parsed
	// by it, `--name` in `vault add <path> --name x` included.
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg.Extensions = strings.Split(*extensions, ",")

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
	case "recognise":
		return recogniseCommand(ctx, out, cfg, rest[1:])
	case "proofread":
		return proofreadCommand(ctx, out, cfg, rest[1:])
	case "search":
		return searchCommand(ctx, out, cfg, rest[1:])
	case "links":
		return linksCommand(ctx, out, cfg, rest[1:])
	case "problems":
		return problemsCommand(ctx, out, cfg, rest[1:])
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
