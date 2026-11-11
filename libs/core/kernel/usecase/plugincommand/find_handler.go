package plugincommand

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// SeveralHandlersError is a shared command run over a path where the handlers of several plugins hold
// and none was picked.
type SeveralHandlersError struct {
	Command   string
	Path      string
	PluginIDs []string
}

// Error names the command, the path and the plugins.
func (e *SeveralHandlersError) Error() string {
	return fmt.Sprintf("several plugins handle %s over %q: %s", e.Command, e.Path, strings.Join(e.PluginIDs, ", "))
}

// ListBackendHandlers lists the handlers of shared commands that a plugin backend runs with no item to
// pick, in the order of the plugins' ids.
func (u List) ListBackendHandlers() []plugin.CommandEntry {
	handlers := slices.DeleteFunc(u.ExecuteHandlers(), func(one plugin.CommandEntry) bool {
		return !hasBackend(u.commands, one.PluginID) || hasItems(u.conditions, one.Command)
	})
	return slices.SortedStableFunc(slices.Values(handlers), func(a, b plugin.CommandEntry) int {
		return strings.Compare(a.PluginID, b.PluginID)
	})
}

// ListSharedHandlers lists the handlers of the shared command of the id as ListBackendHandlers does;
// none where the id is no shared command.
func (u List) ListSharedHandlers(id string) []plugin.CommandEntry {
	if !manifest.IsSharedCommand(id) {
		return nil
	}
	return slices.DeleteFunc(u.ListBackendHandlers(), func(one plugin.CommandEntry) bool { return one.Command.ID != id })
}

// FindHandler picks the handler of the plugin of the id among the handlers, or, with no id, the one
// handler whose condition holds over the path. A plugin among none is ErrUnknownCommand, no handler
// that holds ErrBadArguments, and several a *SeveralHandlersError.
func (u Run) FindHandler(ctx context.Context, v domain.Vault, handlers []plugin.CommandEntry, pluginID, path string) (plugin.CommandEntry, error) {
	if len(handlers) == 0 {
		return plugin.CommandEntry{}, ErrUnknownCommand
	}
	id := handlers[0].Command.ID
	if pluginID != "" {
		index := slices.IndexFunc(handlers, func(one plugin.CommandEntry) bool { return one.PluginID == pluginID })
		if index < 0 {
			return plugin.CommandEntry{}, fmt.Errorf("%w: %s handles no %s", ErrUnknownCommand, pluginID, id)
		}
		return handlers[index], nil
	}
	available := slices.DeleteFunc(slices.Clone(handlers), func(one plugin.CommandEntry) bool { return !u.IsAvailable(ctx, v, one, path) })
	switch len(available) {
	case 0:
		return plugin.CommandEntry{}, fmt.Errorf("%w: no plugin handles %s over %q", ErrBadArguments, id, path)
	case 1:
		return available[0], nil
	}
	several := &SeveralHandlersError{Command: id, Path: path}
	for _, one := range available {
		several.PluginIDs = append(several.PluginIDs, one.PluginID)
	}
	return plugin.CommandEntry{}, several
}
