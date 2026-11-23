package plugincommand

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/commandrun"
)

// RunCommand runs the command a call names: Delete text over the version its item names, a command with
// items over the version its item names, and any other command in its plugin.
type RunCommand struct {
	list     List
	run      Runner
	versions VersionItems
}

// NewRunCommand makes the scenario that runs a command of the list through run. versions are the text
// versions that a command with items runs over; without them such a command runs nothing.
func NewRunCommand(list List, run Runner, versions VersionItems) RunCommand {
	if versions == nil {
		versions = noVersions{}
	}
	return RunCommand{list: list, run: run, versions: versions}
}

// Execute runs the call over the vault. A call that is not a command, matches none, or names a version
// the file does not offer is plugin.ErrUnknownCommand or plugin.ErrBadArguments, and nothing is run.
func (u RunCommand) Execute(ctx context.Context, v domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	if isDeleteText(call.PluginID, call.Command) {
		return plugin.CommandResult{}, u.deleteVersion(ctx, v, call)
	}
	entry, found := findEntry(u.list, call.PluginID, call.Command)
	switch {
	case !found || !commandrun.HasItems(u.list.conditions, entry.Command):
		return u.run.Run(ctx, v, call)
	case call.ItemID == "":
		return plugin.CommandResult{}, fmt.Errorf("%w: %s runs over an item it offers", plugin.ErrBadArguments, call.Command)
	}
	result, err := u.versions.RunVersionItem(ctx, v, call.Path, entry, domain.TextVersionID(call.ItemID))
	if errors.Is(err, text.ErrInvalidVersion) {
		err = fmt.Errorf("%w: %w", plugin.ErrBadArguments, err)
	}
	return result, err
}

func (u RunCommand) deleteVersion(ctx context.Context, v domain.Vault, call plugin.CommandCall) error {
	if call.ItemID == "" {
		return fmt.Errorf("%w: %s deletes the version its item names", plugin.ErrBadArguments, call.Command)
	}
	err := u.versions.DeleteVersion(ctx, v, call.Path, domain.TextVersionID(call.ItemID))
	if errors.Is(err, text.ErrInvalidVersion) {
		return fmt.Errorf("%w: %w", plugin.ErrBadArguments, err)
	}
	return err
}
