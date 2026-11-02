package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// PluginCommands is the commands the enabled plugins declare, and what readies the plugins to run
// one over a vault.
type PluginCommands struct {
	List plugincommand.List
	// Open lets the plugins work the vault and returns what runs a command there, with what gives
	// back what it opened.
	Open func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error)
}

// findPluginCommand returns the command a subcommand names, and false where no plugin declares one.
func findPluginCommand(deps Deps, name string) (plugincommand.Entry, bool) {
	if deps.PluginCommands == nil {
		return plugincommand.Entry{}, false
	}
	return deps.PluginCommands().List.Find(name)
}

// writePluginCommandUsage writes a line for each command the enabled plugins declare.
func writePluginCommandUsage(r *reporter, deps Deps) {
	if deps.PluginCommands == nil {
		return
	}
	entries := deps.PluginCommands().List.Execute()
	if len(entries) == 0 {
		return
	}
	r.Println("\ncommands of the plugins:")
	for _, entry := range entries {
		r.Printf("  numen-cli %s\n      %s\n", getPluginCommandSynopsis(entry.Command), entry.Command.Title)
	}
}

// getPluginCommandSynopsis is how the command is typed: its id, the vault, the file where it runs
// over one, and a flag for each argument, an optional one in brackets.
func getPluginCommandSynopsis(command plugincommand.Command) string {
	words := []string{command.ID, "<vault>"}
	if command.HasFiles() {
		words = append(words, "<file>")
	}
	for _, argument := range command.Arguments {
		word := "--" + argument.Name
		if argument.Type != plugincommand.ArgumentBoolean {
			word += " <" + argument.Type + ">"
		}
		if !argument.IsRequired {
			word = "[" + word + "]"
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

// runPluginCommand runs a command that a plugin declares over a vault, with the
// arguments given as flags, and writes the plugin's output.
func runPluginCommand(
	ctx context.Context, out io.Writer, deps Deps, entry plugincommand.Entry, args []string,
) (err error) {
	declared := entry.Command
	usage := errors.New("usage: numen-cli " + getPluginCommandSynopsis(declared))
	fs := flag.NewFlagSet(declared.ID, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	arguments := map[string]string{}
	for _, argument := range declared.Arguments {
		set := func(value string) error {
			arguments[argument.Name] = value
			return nil
		}
		if argument.Type == plugincommand.ArgumentBoolean {
			fs.BoolFunc(argument.Name, argument.Title, set)
		} else {
			fs.Func(argument.Name, argument.Title, set)
		}
	}
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", usage, err)
	}
	wanted := 1
	if declared.HasFiles() {
		wanted = 2
	}
	if len(positional) != wanted {
		return usage
	}
	v, err := findVault(deps, positional[0])
	if err != nil {
		return err
	}
	run, closeRun, err := deps.PluginCommands().Open(ctx, v)
	if err != nil {
		return err
	}
	defer closeInto(&err, closeRun)

	call := plugincommand.Call{PluginID: entry.PluginID, Command: declared.ID, Arguments: arguments}
	if declared.HasFiles() {
		call.Path = positional[1]
	}
	result, err := run.Execute(ctx, v, call)
	if err != nil {
		return err
	}
	r := newReporter(out)
	if result.Message != "" {
		r.Println(result.Message)
	}
	if result.OpenedPath != "" {
		r.Println(result.OpenedPath)
	}
	return r.Err()
}
