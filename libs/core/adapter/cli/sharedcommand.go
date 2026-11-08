package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// pluginFlag is the flag that picks the plugin whose handler of a shared command runs.
const pluginFlag = "plugin"

func listSharedHandlers(deps Deps, id string) []plugincommand.Entry {
	if deps.PluginCommands == nil {
		return nil
	}
	return deps.PluginCommands().List.ListSharedHandlers(id)
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

	handler, err := run.FindHandler(ctx, v, handlers, picked, path)
	var several *plugincommand.SeveralHandlersError
	if errors.As(err, &several) {
		return fmt.Errorf("%w; pick one with --%s", err, pluginFlag)
	}
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
