package textversion

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// MenuEntry is one entry of a file's menu: a command a plugin declares, offered
// over the file or over a version of it. Title and Icon are the command's
// declaration; Icon is a Lucide icon name. Version is the text version the
// command runs over, and is empty for a command over the file. IsLive marks
// Version as the version displayed and indexed now.
type MenuEntry struct {
	PluginID   string
	PluginName string
	Command    string
	Title      string
	Icon       string
	Version    domain.TextVersionID
	IsLive     bool
}

// FileMenu is the menu of a vault file: the commands the enabled plugins
// declare that apply to the file as it is now. A command that names content
// types applies to a file of one of them; a command with conditions applies
// to each version of the file in whose context they all hold.
type FileMenu struct {
	versions *Versions
	commands pluginport.PluginCommands
	types    contenttypeport.ContentTypeTable
	readers  vaultport.VaultReaders
}

// NewFileMenu is the menu the plugin commands make of the files the readers open,
// where the table gives the type of a file.
func NewFileMenu(
	versions *Versions, commands pluginport.PluginCommands, types contenttypeport.ContentTypeTable, readers vaultport.VaultReaders,
) (*FileMenu, error) {
	if versions == nil || commands == nil || types == nil || readers == nil {
		return nil, errors.New("a file menu needs text versions, plugin commands, a content type table and vault readers")
	}
	return &FileMenu{versions: versions, commands: commands, types: types, readers: readers}, nil
}

// GetMenu is what the file's menu offers. A file is read and hashed only when
// it has versions, to tell which is live.
func (u *FileMenu) GetMenu(ctx context.Context, v domain.Vault, path string) ([]MenuEntry, error) {
	versions, err := u.versions.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.findLiveVersion(ctx, v, path, versions)
	if err != nil {
		return nil, err
	}
	values := pluginport.GetTargetValues(path, string(u.types.GetContentType(path)), "")
	var entries []MenuEntry
	for _, one := range u.commands.ListCommands() {
		command := one.Command
		condition, err := pluginport.ParseCondition(command.When)
		if err != nil {
			continue
		}
		entry := MenuEntry{PluginID: one.PluginID, PluginName: one.PluginName, Command: command.ID, Title: command.Title, Icon: command.Icon}
		switch {
		case condition.HasItems():
			for _, item := range listVersionItems(condition, values, versions, live, hasLive) {
				entry.Version, entry.IsLive = item.Version, item.IsLive
				entries = append(entries, entry)
			}
		case condition.NamesTarget() && !slices.Contains(condition.Names(), "target.noteType") &&
			!hasRequiredArgument(command) && condition.Evaluate(values) == pluginport.ConditionTrue:
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

// findLiveVersion is the live version of the file among its versions. The file is read and hashed
// only when it has versions.
func (u *FileMenu) findLiveVersion(ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion) (domain.TextVersion, bool, error) {
	if len(versions) == 0 {
		return domain.TextVersion{}, false, nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.TextVersion{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	return u.versions.findLive(ctx, v, versions, domain.HashContent(raw))
}

// hasRequiredArgument reports whether the command cannot run without a value a menu does not give.
func hasRequiredArgument(command pluginport.PluginCommand) bool {
	for _, argument := range command.Arguments {
		if argument.IsRequired {
			return true
		}
	}
	return false
}

// VersionItem is a text version of a file that a command with items offers to run over.
type VersionItem struct {
	Version domain.TextVersionID
	Kind    string
	IsLive  bool
}

// getVersionValues gives the version variables of a when condition for one version of a file.
func getVersionValues(target map[string]any, version domain.TextVersion, isLive bool) map[string]any {
	values := maps.Clone(target)
	values["version.kind"] = string(version.Kind)
	values["version.isLive"] = isLive
	return values
}

// listVersionItems is the versions of the file over which the condition holds, in their order.
func listVersionItems(condition pluginport.PluginCondition, target map[string]any, versions []domain.TextVersion, live domain.TextVersion, hasLive bool) []VersionItem {
	var items []VersionItem
	for _, version := range versions {
		isLive := hasLive && version.GetID() == live.GetID()
		if condition.Evaluate(getVersionValues(target, version, isLive)) == pluginport.ConditionTrue {
			items = append(items, VersionItem{Version: version.GetID(), Kind: string(version.Kind), IsLive: isLive})
		}
	}
	return items
}

// ListVersionItems lists the text versions of the file at the path over which the when condition of
// the command holds; a command whose condition names no version.* variable has none.
func (u *FileMenu) ListVersionItems(ctx context.Context, v domain.Vault, path string, entry pluginport.PluginCommandEntry) ([]VersionItem, error) {
	condition, err := pluginport.ParseCondition(entry.Command.When)
	if err != nil || !condition.HasItems() {
		return nil, err
	}
	versions, err := u.versions.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.findLiveVersion(ctx, v, path, versions)
	if err != nil {
		return nil, err
	}
	values := pluginport.GetTargetValues(path, string(u.types.GetContentType(path)), "")
	return listVersionItems(condition, values, versions, live, hasLive), nil
}

// RunVersionItem runs the command of the entry over the version of the file at the path, which it gets
// as its argument version-id. A version the command does not offer over the file now, or a command
// without items, is ErrInvalidVersion.
func (u *FileMenu) RunVersionItem(
	ctx context.Context, v domain.Vault, path string, entry pluginport.PluginCommandEntry, version domain.TextVersionID,
) (pluginport.PluginCommandResult, error) {
	items, err := u.ListVersionItems(ctx, v, path, entry)
	if err != nil {
		return pluginport.PluginCommandResult{}, err
	}
	if !slices.ContainsFunc(items, func(item VersionItem) bool { return item.Version == version }) {
		return pluginport.PluginCommandResult{}, fmt.Errorf("%w: %s of %s offers no version %q over %s",
			ErrInvalidVersion, entry.Command.ID, entry.PluginID, version, path)
	}
	return u.commands.RunCommand(ctx, pluginport.PluginCommandCall{
		PluginID: entry.PluginID, Command: entry.Command.ID, Path: path,
		Arguments: map[string]string{pluginport.CommandArgumentVersion: string(version)},
	})
}

// RunMenuEntry runs the command of an entry the file's menu offers now, over
// the entry's version. An entry the menu does not offer is ErrInvalidVersion.
func (u *FileMenu) RunMenuEntry(ctx context.Context, v domain.Vault, path string, entry MenuEntry) error {
	entries, err := u.GetMenu(ctx, v, path)
	if err != nil {
		return err
	}
	isInMenu := slices.ContainsFunc(entries, func(candidate MenuEntry) bool {
		return candidate.PluginID == entry.PluginID && candidate.Command == entry.Command && candidate.Version == entry.Version
	})
	if !isInMenu {
		return fmt.Errorf("%w: %s of %s is not offered over %s", ErrInvalidVersion, entry.Command, entry.PluginID, path)
	}
	call := pluginport.PluginCommandCall{PluginID: entry.PluginID, Command: entry.Command, Path: path}
	if entry.Version != "" {
		call.Arguments = map[string]string{pluginport.CommandArgumentVersion: string(entry.Version)}
	}
	_, err = u.commands.RunCommand(ctx, call)
	return err
}

// The errors of a menu entry the file's menu does not hold, for a driving adapter.
var (
	ErrNoText            = textport.ErrNoText
	ErrNotTextPlugin     = textport.ErrNotTextPlugin
	ErrInvalidVersion    = textport.ErrInvalidVersion
	ErrSourceNotAccepted = textport.ErrSourceNotAccepted
)
