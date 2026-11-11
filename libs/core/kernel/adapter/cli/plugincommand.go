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
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PluginCommands is the commands the enabled plugins declare, and what readies the plugins to run
// one over a vault.
type PluginCommands struct {
	List plugincommand.List
	Open func(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error)
}

// declaredCommand is a command a plugin manifest declares, run by that plugin's backend.
type declaredCommand struct {
	plugins func() PluginCommands
	entry   plugin.CommandEntry
}

func (c declaredCommand) Name() string { return c.entry.Command.ID }

func (c declaredCommand) Synopsis() string {
	return fmt.Sprintf("  numen-cli %s\n      %s", getPluginCommandSynopsis(c.plugins().List, c.entry.Command), c.entry.Command.Title)
}

func getPluginCommandSynopsis(list plugincommand.List, command manifest.Command) string {
	words := []string{command.ID, "<vault>"}
	if list.HasTargetVariable(command) {
		words = append(words, "<file>")
	}
	for _, argument := range command.Arguments {
		word := "--" + argument.Name
		if argument.Type != manifest.CommandArgumentBoolean {
			word += " <" + argument.Type + ">"
		}
		if !argument.IsRequired {
			word = "[" + word + "]"
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

func (c declaredCommand) Run(ctx context.Context, term terminal, args []string) error {
	declared := c.entry.Command
	commands := c.plugins()
	list := commands.List
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

	call := plugin.CommandCall{PluginID: c.entry.PluginID, Command: declared.ID, Arguments: arguments}
	if list.HasTargetVariable(declared) {
		call.Path = positional[1]
	}
	result, err := list.RunInVault(ctx, commands.Open, positional[0], call)
	if err != nil {
		return err
	}
	return writeResult(term.out, result)
}

func newArgumentFlags(declared manifest.Command) (*flag.FlagSet, map[string]string) {
	fs := flag.NewFlagSet(declared.ID, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	arguments := map[string]string{}
	for _, argument := range declared.Arguments {
		set := func(value string) error {
			arguments[argument.Name] = value
			return nil
		}
		if argument.Type == manifest.CommandArgumentBoolean {
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
