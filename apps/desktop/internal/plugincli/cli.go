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

// output writes to a stream and keeps the first write error.
type output struct {
	w   io.Writer
	err error
}

func (o *output) printf(format string, args ...any) {
	if o.err != nil {
		return
	}
	_, o.err = fmt.Fprintf(o.w, format, args...)
}

func (o *output) println(args ...any) {
	if o.err != nil {
		return
	}
	_, o.err = fmt.Fprintln(o.w, args...)
}

// finish returns code, or 1 when code reports success and a stream failed to take the output.
func finish(code int, outs ...*output) int {
	if code != 0 {
		return code
	}
	for _, o := range outs {
		if o.err != nil {
			return 1
		}
	}
	return 0
}

// Run executes a plugin subcommand and returns the exit status code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return finish(0, printUsage(stdout))
	}

	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "package":
		return runPackage(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		return finish(0, printUsage(stdout))
	default:
		out := &output{w: stderr}
		out.printf("unknown plugin command: %q\n\n%s", args[0], usage)
		return 1
	}
}

func printUsage(w io.Writer) *output {
	out := &output{w: w}
	out.printf("%s", usage)
	return out
}

func runValidate(args []string, stdoutWriter, stderrWriter io.Writer) int {
	stdout, stderr := &output{w: stdoutWriter}, &output{w: stderrWriter}
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderrWriter)

	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return 1
	}

	if len(positional) == 0 {
		stderr.println("error: plugin path is required")
		stderr.println("usage: numen plugin validate <path>")
		return finish(1, stderr)
	}

	targetPath := positional[0]
	issues, err := Validate(targetPath)
	if err != nil {
		stderr.printf("validation failed: %v\n", err)
		return 1
	}

	if len(issues) > 0 {
		stderr.println("validation errors:")
		for _, issue := range issues {
			stderr.printf("  - %s\n", issue)
		}
		return 1
	}

	stdout.printf("plugin at %s is valid\n", targetPath)
	return finish(0, stdout)
}

func runPackage(args []string, stdoutWriter, stderrWriter io.Writer) int {
	stdout, stderr := &output{w: stdoutWriter}, &output{w: stderrWriter}
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	fs.SetOutput(stderrWriter)

	var outArchive string
	fs.StringVar(&outArchive, "out", "", "destination path for archive")

	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return 1
	}

	if len(positional) == 0 {
		stderr.println("error: plugin path is required")
		stderr.println("usage: numen plugin package <path> [--out=<archive>]")
		return 1
	}

	targetPath := positional[0]
	archivePath, err := Package(targetPath, outArchive, stdoutWriter)
	if err != nil {
		stderr.printf("package error: %v\n", err)
		return 1
	}

	stdout.printf("packaged plugin: %s\n", archivePath)
	return finish(0, stdout)
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
