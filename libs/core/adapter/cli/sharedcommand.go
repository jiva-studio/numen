package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// pluginFlag is the flag that picks the plugin whose handler of a shared command runs.
const pluginFlag = "plugin"

func listSharedHandlers(deps Deps, id string) []plugincommand.Entry {
	if deps.PluginCommands == nil || !plugincommand.IsSharedCommand(id) {
		return nil
	}
	return slices.DeleteFunc(listRunnableHandlers(deps), func(one plugincommand.Entry) bool {
		return one.Command.ID != id
	})
}

func listRunnableHandlers(deps Deps) []plugincommand.Entry {
	list := deps.PluginCommands().List
	handlers := slices.DeleteFunc(list.ExecuteHandlers(), func(one plugincommand.Entry) bool {
		return !list.HasBackend(one.PluginID) || plugincommand.HasItems(one.Command)
	})
	return slices.SortedStableFunc(slices.Values(handlers), func(a, b plugincommand.Entry) int {
		return strings.Compare(a.PluginID, b.PluginID)
	})
}

func isOverFile(handlers []plugincommand.Entry) bool {
	return slices.ContainsFunc(handlers, func(one plugincommand.Entry) bool { return plugincommand.HasTargetVariable(one.Command) })
}

func getSharedCommandSynopsis(handlers []plugincommand.Entry) string {
	declared := handlers[0].Command
	for _, one := range handlers {
		if plugincommand.HasTargetVariable(one.Command) {
			declared.When = one.Command.When
		}
	}
	return getPluginCommandSynopsis(declared) + " [--" + pluginFlag + " <id>]"
}

func writeSharedCommandUsage(r *reporter, deps Deps) {
	if deps.PluginCommands == nil {
		return
	}
	handlers := listRunnableHandlers(deps)
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
		r.Printf("  numen-cli %s\n", getSharedCommandSynopsis(of))
		for _, one := range of {
			r.Printf("      %s (%s)\n", one.Command.Title, one.PluginID)
		}
	}
}

func runSharedCommand(
	ctx context.Context, out io.Writer, deps Deps, handlers []plugincommand.Entry, args []string,
) (err error) {
	usage := errors.New("usage: numen-cli " + getSharedCommandSynopsis(handlers))
	fs, arguments := newArgumentFlags(handlers[0].Command)
	picked := ""
	fs.StringVar(&picked, pluginFlag, "", "the plugin whose handler runs")
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return fmt.Errorf("%w: %w", usage, err)
	}
	wanted := 1
	if isOverFile(handlers) {
		wanted = 2
	}
	if len(positional) != wanted {
		return usage
	}
	v, err := findVault(deps, positional[0])
	if err != nil {
		return err
	}
	path := ""
	if wanted == 2 {
		path = positional[1]
	}
	run, closeRun, err := deps.PluginCommands().Open(ctx, v)
	if err != nil {
		return err
	}
	defer closeInto(&err, closeRun)

	handler, err := pickHandler(handlers, picked, path, func(one plugincommand.Entry) bool {
		return run.IsAvailable(ctx, v, one, path)
	})
	if err != nil {
		return err
	}
	result, err := run.Execute(ctx, v, plugincommand.Call{
		PluginID: handler.PluginID, Command: handler.Command.ID, Path: path, Arguments: arguments,
	})
	if err != nil {
		return err
	}
	return writeResult(out, result)
}

func pickHandler(
	handlers []plugincommand.Entry, picked, path string, isAvailable func(plugincommand.Entry) bool,
) (plugincommand.Entry, error) {
	id := handlers[0].Command.ID
	if picked != "" {
		index := slices.IndexFunc(handlers, func(one plugincommand.Entry) bool { return one.PluginID == picked })
		if index < 0 {
			return plugincommand.Entry{}, fmt.Errorf("%w: %s handles no %s", plugincommand.ErrUnknownCommand, picked, id)
		}
		return handlers[index], nil
	}
	available := slices.DeleteFunc(slices.Clone(handlers), func(one plugincommand.Entry) bool { return !isAvailable(one) })
	switch len(available) {
	case 0:
		return plugincommand.Entry{}, fmt.Errorf("%w: no plugin handles %s over %q", plugincommand.ErrBadArguments, id, path)
	case 1:
		return available[0], nil
	}
	plugins := make([]string, 0, len(available))
	for _, one := range available {
		plugins = append(plugins, one.PluginID)
	}
	return plugincommand.Entry{}, fmt.Errorf("several plugins handle %s over %q; pick one with --%s: %s",
		id, path, pluginFlag, strings.Join(plugins, ", "))
}
