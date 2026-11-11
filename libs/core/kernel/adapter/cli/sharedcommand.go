package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// pluginFlag is the flag that picks the plugin whose handler of a shared command runs.
const pluginFlag = "plugin"

func listSharedHandlers(deps Deps, id string) []plugin.CommandEntry {
	if deps.PluginCommands == nil {
		return nil
	}
	return deps.PluginCommands().List.ListSharedHandlers(id)
}

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

func writeSharedCommandUsage(r *reporter, deps Deps) {
	if deps.PluginCommands == nil {
		return
	}
	handlers := deps.PluginCommands().List.ListBackendHandlers()
	var ids []string
	for _, one := range handlers {
		if !slices.Contains(ids, one.Command.ID) {
			ids = append(ids, one.Command.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	r.Println("\nshared commands, run by the handler of a plugin:")
	for _, id := range ids {
		of := listSharedHandlers(deps, id)
		r.Printf("  numen-cli %s\n", getSharedCommandSynopsis(deps.PluginCommands().List, of))
		for _, one := range of {
			r.Printf("      %s (%s)\n", one.Command.Title, one.PluginID)
		}
	}
}

func runSharedCommand(
	ctx context.Context, out io.Writer, deps Deps, handlers []plugin.CommandEntry, args []string,
) error {
	commands := deps.PluginCommands()
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
	result, err := list.RunSharedInVault(ctx, commands.Open, positional[0], handlers, picked, plugin.CommandCall{
		Path: path, Arguments: arguments,
	})
	if errors.As(err, new(*plugincommand.SeveralHandlersError)) {
		return fmt.Errorf("%w; pick one with --%s", err, pluginFlag)
	}
	if err != nil {
		return err
	}
	return writeResult(out, result)
}
