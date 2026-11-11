package plugincommand

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// RunCommand runs the command a call names: Delete text over the version its item names, a command with
// items over the version its item names, and any other command in its plugin.
type RunCommand struct {
	list     List
	run      Run
	versions VersionItems
}

// Execute runs the call over the vault. A call that is not a command, matches none, or names a version
// the file does not offer is ErrUnknownCommand or ErrBadArguments, and nothing is run.
func (u RunCommand) Execute(ctx context.Context, v domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	if isDeleteText(call.PluginID, call.Command) {
		return plugin.CommandResult{}, u.deleteVersion(ctx, v, call)
	}
	entry, found := findEntry(u.list, call.PluginID, call.Command)
	switch {
	case !found || !hasItems(u.list.conditions, entry.Command):
		return u.run.Execute(ctx, v, call)
	case call.ItemID == "":
		return plugin.CommandResult{}, fmt.Errorf("%w: %s runs over an item it offers", ErrBadArguments, call.Command)
	case u.versions == nil:
		return plugin.CommandResult{}, errNoVersions
	}
	result, err := u.versions.RunVersionItem(ctx, v, call.Path, entry, domain.TextVersionID(call.ItemID))
	if errors.Is(err, text.ErrInvalidVersion) {
		err = fmt.Errorf("%w: %w", ErrBadArguments, err)
	}
	return result, err
}

func (u RunCommand) deleteVersion(ctx context.Context, v domain.Vault, call plugin.CommandCall) error {
	switch {
	case call.ItemID == "":
		return fmt.Errorf("%w: %s deletes the version its item names", ErrBadArguments, call.Command)
	case u.versions == nil:
		return errNoVersions
	}
	err := u.versions.DeleteVersion(ctx, v, call.Path, domain.TextVersionID(call.ItemID))
	if errors.Is(err, text.ErrInvalidVersion) {
		return fmt.Errorf("%w: %w", ErrBadArguments, err)
	}
	return err
}
