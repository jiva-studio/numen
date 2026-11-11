package plugincommand

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// VersionItems lists the text versions of a file a command with items offers, and runs the command over
// one of them; it lists every version of a file and deletes one for the application command Delete text.
type VersionItems interface {
	ListVersionItems(ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry) ([]text.VersionItem, error)
	RunVersionItem(ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry, version domain.TextVersionID) (plugin.CommandResult, error)
	ListEveryVersion(ctx context.Context, v domain.Vault, path string) ([]text.VersionItem, error)
	DeleteVersion(ctx context.Context, v domain.Vault, path string, version domain.TextVersionID) error
}

// errNoVersions is a command with items run where the window has no text versions.
var errNoVersions = errors.New("the window has no text versions")

func isDeleteText(pluginID, command string) bool {
	return pluginID == "" && command == text.DeleteCommand
}

func findEntry(list List, pluginID, command string) (plugin.CommandEntry, bool) {
	var entries []plugin.CommandEntry
	if manifest.IsSharedCommand(command) {
		entries = list.ExecuteHandlers()
	} else {
		entries = list.ExecuteAll()
	}
	for _, one := range entries {
		if one.PluginID == pluginID && one.Command.ID == command {
			return one, true
		}
	}
	return plugin.CommandEntry{}, false
}
