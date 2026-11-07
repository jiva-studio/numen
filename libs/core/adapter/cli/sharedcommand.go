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

// listSharedHandlers is the handlers of the shared command of the id that a plugin backend runs over
// a file, in the order of the plugins' ids; none where the id is no shared command. A handler that
// offers items is left out: the command line names no item.
func listSharedHandlers(deps Deps, id string) []plugincommand.Entry {
	if deps.PluginCommands == nil || !plugincommand.IsSharedCommand(id) {
		return nil
	}
	return slices.DeleteFunc(listRunnableHandlers(deps), func(one plugincommand.Entry) bool {
		return one.Command.ID != id
	})
}

// listRunnableHandlers is every handler of a shared command that a plugin backend runs and that
// offers no items.
func listRunnableHandlers(deps Deps) []plugincommand.Entry {
	list := deps.PluginCommands().List
	handlers := slices.DeleteFunc(list.ExecuteHandlers(), func(one plugincommand.Entry) bool {
		return !list.HasBackend(one.PluginID) || plugincommand.HasItems(one.Command)
	})
	return slices.SortedStableFunc(slices.Values(handlers), func(a, b plugincommand.Entry) int {
		return strings.Compare(a.PluginID, b.PluginID)
	})
}

// isOverFile reports whether a handler of the shared command runs over a file, so a run names one.
func isOverFile(handlers []plugincommand.Entry) bool {
	return slices.ContainsFunc(handlers, func(one plugincommand.Entry) bool { return plugincommand.NamesTarget(one.Command) })
}

// getSharedCommandSynopsis is how a shared command is typed: its id, the vault, the file where a
// handler runs over one, a flag for each argument, and the flag that picks the plugin.
func getSharedCommandSynopsis(handlers []plugincommand.Entry) string {
	declared := handlers[0].Command
	for _, one := range handlers {
		if plugincommand.NamesTarget(one.Command) {
			declared.When = one.Command.When
		}
	}
	return getPluginCommandSynopsis(declared) + " [--" + pluginFlag + " <id>]"
}

// writeSharedCommandUsage writes a line for each shared command the enabled plugins handle, and the
// title and plugin of each of its handlers.
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

// runSharedCommand runs a shared command over a vault through the handler of the plugin the flag
// picks, or else the one handler whose condition holds over the file, with the arguments given as
// flags, and writes the plugin's output.
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

	if picked != "" {
		handlers = slices.DeleteFunc(handlers, func(one plugincommand.Entry) bool { return one.PluginID != picked })
	} else {
		handlers = slices.DeleteFunc(handlers, func(one plugincommand.Entry) bool { return !run.IsAvailable(ctx, v, one, path) })
	}
	switch {
	case len(handlers) == 0 && picked != "":
		return fmt.Errorf("%w: %s handles no %s", plugincommand.ErrUnknownCommand, picked, fs.Name())
	case len(handlers) == 0:
		return fmt.Errorf("%w: no plugin handles %s over %q", plugincommand.ErrBadArguments, fs.Name(), path)
	case len(handlers) > 1:
		plugins := make([]string, 0, len(handlers))
		for _, one := range handlers {
			plugins = append(plugins, one.PluginID)
		}
		return fmt.Errorf("several plugins handle %s over %q; pick one with --%s: %s",
			fs.Name(), path, pluginFlag, strings.Join(plugins, ", "))
	}
	result, err := run.Execute(ctx, v, plugincommand.Call{
		PluginID: handlers[0].PluginID, Command: handlers[0].Command.ID, Path: path, Arguments: arguments,
	})
	if err != nil {
		return err
	}
	return writeResult(out, result)
}
