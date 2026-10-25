// Package plugincli provides developer tooling for scaffolding, validating,
// and packaging Numen plugins.
package plugincli

import (
	"flag"
	"fmt"
	"io"
)

const usage = `numen plugin — plugin developer tooling

usage:
  numen plugin validate <path>
  numen plugin package <path> [--out=<archive>]
`

// Run executes a plugin subcommand and returns the exit status code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "package":
		return runPackage(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown plugin command: %q\n\n%s", args[0], usage)
		return 1
	}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return 1
	}

	if len(positional) == 0 {
		fmt.Fprintln(stderr, "error: plugin path is required")
		fmt.Fprintln(stderr, "usage: numen plugin validate <path>")
		return 1
	}

	targetPath := positional[0]
	issues, err := Validate(targetPath)
	if err != nil {
		fmt.Fprintf(stderr, "validation failed: %v\n", err)
		return 1
	}

	if len(issues) > 0 {
		fmt.Fprintln(stderr, "validation errors:")
		for _, issue := range issues {
			fmt.Fprintf(stderr, "  - %s\n", issue)
		}
		return 1
	}

	fmt.Fprintf(stdout, "plugin at %s is valid\n", targetPath)
	return 0
}

func runPackage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var outArchive string
	fs.StringVar(&outArchive, "out", "", "destination path for archive")

	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return 1
	}

	if len(positional) == 0 {
		fmt.Fprintln(stderr, "error: plugin path is required")
		fmt.Fprintln(stderr, "usage: numen plugin package <path> [--out=<archive>]")
		return 1
	}

	targetPath := positional[0]
	archivePath, err := Package(targetPath, outArchive, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "package error: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "packaged plugin: %s\n", archivePath)
	return 0
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
