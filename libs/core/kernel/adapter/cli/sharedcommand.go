package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// pluginFlag is the flag that picks the plugin whose handler of a shared command runs.
const pluginFlag = "plugin"

func isOverFile(list plugincommand.List, handlers []plugin.CommandEntry) bool {
	return slices.ContainsFunc(handlers, func(one plugin.CommandEntry) bool { return list.HasTargetVariable(one.Command) })
}

func getSharedCommandSynopsis(list plugincommand.List, handlers []plugin.CommandEntry) string {
	declared := handlers[0].Command
	for _, one := range handlers {
		if list.HasTargetVariable(one.Command) {
			declared.When = one.Command.When
		}
	}
	return getPluginCommandSynopsis(list, declared) + " [--" + pluginFlag + " <id>]"
}

// sharedCommand is a command that several plugins may handle; the --plugin flag picks the handler.
type sharedCommand struct {
	plugins  func() PluginCommands
	handlers []plugin.CommandEntry
}

func (c sharedCommand) Name() string { return c.handlers[0].Command.ID }

func (c sharedCommand) Synopsis() string {
	lines := []string{"  numen-cli " + getSharedCommandSynopsis(c.plugins().List, c.handlers)}
	for _, one := range c.handlers {
		lines = append(lines, fmt.Sprintf("      %s (%s)", one.Command.Title, one.PluginID))
	}
	return strings.Join(lines, "\n")
}

func (c sharedCommand) Run(ctx context.Context, term terminal, args []string) error {
	handlers := c.handlers
	commands := c.plugins()
	list := commands.List
	usage := errors.New("usage: numen-cli " + getSharedCommandSynopsis(list, handlers))
	fs, arguments := newArgumentFlags(handlers[0].Command)
	picked := ""
	fs.StringVar(&picked, pluginFlag, "", "the plugin whose handler runs")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", usage, err)
	}
	wanted := 1
	if isOverFile(list, handlers) {
		wanted = 2
	}
	if len(positional) != wanted {
		return usage
	}
	path := ""
	if wanted == 2 {
		path = positional[1]
	}
	result, err := commands.ByName.ExecuteSharedByName(ctx, commands.Open, positional[0], handlers, picked, plugin.CommandCall{
		Path: path, Arguments: arguments,
	})
	if errors.As(err, new(*plugincommand.SeveralHandlersError)) {
		return fmt.Errorf("%w; pick one with --%s", err, pluginFlag)
	}
	if err != nil {
		return err
	}
	return writeResult(term.out, result)
}
