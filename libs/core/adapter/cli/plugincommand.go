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
func writePluginCommandUsage(out io.Writer, deps Deps) {
	if deps.PluginCommands == nil {
		return
	}
	listed := deps.PluginCommands().List.Execute()
	if len(listed) == 0 {
		return
	}
	fmt.Fprintln(out, "\ncommands of the plugins:")
	for _, entry := range listed {
		fmt.Fprintf(out, "  numen-cli %s\n      %s\n", getPluginCommandSynopsis(entry.Command), entry.Command.Title)
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

// runPluginCommand runs a command a plugin declares over a vault, with the arguments given as flags,
// and writes what the plugin said.
func runPluginCommand(
	ctx context.Context, out io.Writer, deps Deps, entry plugincommand.Entry, args []string,
) error {
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
	defer closeIfOpen(closeRun)

	call := plugincommand.Call{PluginID: entry.PluginID, Command: declared.ID, Arguments: arguments}
	if declared.HasFiles() {
		call.Path = positional[1]
	}
	result, err := run.Execute(ctx, call)
	if err != nil {
		return err
	}
	if result.Message != "" {
		fmt.Fprintln(out, result.Message)
	}
	if result.OpenedPath != "" {
		fmt.Fprintln(out, result.OpenedPath)
	}
	return nil
}
