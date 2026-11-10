package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginCommands is the commands the enabled plugins declare, and what readies the plugins to run
// one over a vault.
type PluginCommands struct {
	List plugincommand.List
	Open func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error)
}

func findPluginCommand(deps Deps, name string) (plugin.CommandEntry, bool) {
	if deps.PluginCommands == nil {
		return plugin.CommandEntry{}, false
	}
	return deps.PluginCommands().List.Find(name)
}

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
		r.Printf("  numen-cli %s\n      %s\n", getPluginCommandSynopsis(deps.PluginCommands().List, entry.Command), entry.Command.Title)
	}
}

func getPluginCommandSynopsis(list plugincommand.List, command plugincommand.Command) string {
	words := []string{command.ID, "<vault>"}
	if list.HasTargetVariable(command) {
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

func runPluginCommand(
	ctx context.Context, out io.Writer, deps Deps, entry plugin.CommandEntry, args []string,
) (err error) {
	declared := entry.Command
	list := deps.PluginCommands().List
	usage := errors.New("usage: numen-cli " + getPluginCommandSynopsis(list, declared))
	fs, arguments := newArgumentFlags(declared)
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", usage, err)
	}
	wanted := 1
	if list.HasTargetVariable(declared) {
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

	call := plugin.CommandCall{PluginID: entry.PluginID, Command: declared.ID, Arguments: arguments}
	if list.HasTargetVariable(declared) {
		call.Path = positional[1]
	}
	result, err := run.Execute(ctx, v, call)
	if err != nil {
		return err
	}
	return writeResult(out, result)
}

func newArgumentFlags(declared plugincommand.Command) (*flag.FlagSet, map[string]string) {
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
	return fs, arguments
}

func writeResult(out io.Writer, result plugin.CommandResult) error {
	r := newReporter(out)
	if result.Message != "" {
		r.Println(result.Message)
	}
	if result.OpenedPath != "" {
		r.Println(result.OpenedPath)
	}
	return r.Err()
}
