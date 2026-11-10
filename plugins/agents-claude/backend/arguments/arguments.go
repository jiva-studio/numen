// Package arguments builds the command lines the Claude Code program is started with.
package arguments

import (
	"fmt"
	"slices"
	"strings"
)

// Options is what a run of the program is started from.
type Options struct {
	Instructions  string
	Model         string
	Reasoning     string
	Configuration string
	Session       string
	Turns         int
	Tools         []string
	MayReadHooks  bool
}

// defaultTurns is how many times the program may go round on one task where the task names no limit.
const defaultTurns = 100

// Command-line flags the program is started with.
const (
	flagOutputFormat = "--output-format"
	flagVerbose      = "--verbose"
	streamJSON       = "stream-json"
)

// GetModelListing returns the arguments that make the program read the initialize request of its
// control protocol on its input.
func GetModelListing() []string {
	return []string{
		"-p", "--input-format", streamJSON, flagOutputFormat, streamJSON, flagVerbose,
		"--setting-sources", "",
	}
}

// Build returns the arguments of a run: a single answer without tools when the options name no
// tool, otherwise a run that may use the named tools of the vault.
func Build(in Options) []string {
	if len(in.Tools) == 0 {
		return buildAnswerArguments(in)
	}
	turns := in.Turns
	if turns <= 0 {
		turns = defaultTurns
	}

	args := []string{
		"-p",
		flagOutputFormat, streamJSON,
		flagVerbose,
		"--include-partial-messages",
		"--strict-mcp-config",
		"--permission-mode", "dontAsk",
		"--max-turns", fmt.Sprint(turns),
	}

	if in.Configuration != "" {
		args = append(args, "--mcp-config", in.Configuration)
	}

	args = append(args, "--tools", "WebSearch")

	var allowed []string
	allowed = append(allowed, "WebSearch")
	for _, t := range in.Tools {
		name := t
		if !strings.HasPrefix(name, "mcp__numen__") {
			name = "mcp__numen__" + name
		}
		allowed = append(allowed, name)
	}
	slices.Sort(allowed)
	args = append(args, "--allowedTools", strings.Join(allowed, ","))

	// A vault's own hooks and settings are never read. The user's own hooks and
	// settings are read only when the setting allows it.
	sources := ""
	if in.MayReadHooks {
		sources = "user"
	}
	args = append(args, "--setting-sources", sources)

	return appendRunFlags(args, in)
}

func buildAnswerArguments(in Options) []string {
	args := []string{
		"-p",
		flagOutputFormat, streamJSON,
		flagVerbose,
		"--include-partial-messages",
		"--strict-mcp-config",
		"--permission-mode", "dontAsk",
		"--max-turns", "1",
		"--tools", "",
		"--setting-sources", "",
	}
	return appendRunFlags(args, in)
}

func appendRunFlags(args []string, in Options) []string {
	if in.Instructions != "" {
		args = append(args, "--append-system-prompt", in.Instructions)
	}
	if in.Model != "" {
		args = append(args, "--model", in.Model)
	}
	if in.Reasoning != "" {
		args = append(args, "--effort", in.Reasoning)
	}
	if in.Session != "" {
		args = append(args, "--resume", in.Session)
	}
	return args
}
