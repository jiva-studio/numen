package textversion

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// FileMenu is the text versions of a vault file that the commands of the
// enabled plugins whose condition names version.* offer as their items, and
// the run of such a command over one of them.
type FileMenu struct {
	versions *Versions
	run      CommandRun
	types    contenttypeport.ContentTypeTable
}

// NewFileMenu returns the menu of plugin commands for the vault files, using the
// table to find the type of a file.
func NewFileMenu(versions *Versions, run CommandRun, types contenttypeport.ContentTypeTable) (*FileMenu, error) {
	if versions == nil || run == nil || types == nil {
		return nil, errors.New("file menu: text versions, the run of commands and content type table are required")
	}
	return &FileMenu{versions: versions, run: run, types: types}, nil
}

// CommandRun runs a command of a plugin checked against its declaration: its backend, its condition
// and its arguments.
type CommandRun interface {
	Execute(ctx context.Context, v domain.Vault, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error)
}

func (u *FileMenu) findLiveVersion(ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion) (domain.TextVersion, bool, error) {
	if len(versions) == 0 {
		return domain.TextVersion{}, false, nil
	}
	contentHash, err := u.versions.GetContentHash(ctx, v, path)
	if err != nil {
		return domain.TextVersion{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	return u.versions.findLive(ctx, v, path, versions, contentHash)
}

// VersionItem is a text version of a file that a command with items offers to run over.
type VersionItem struct {
	Version domain.TextVersionID
	Kind    string
	IsLive  bool
}

func listVersionItems(condition pluginport.PluginCondition, target pluginport.TargetValues, versions []domain.TextVersion, live domain.TextVersion, hasLive bool) []VersionItem {
	var items []VersionItem
	for _, version := range versions {
		isLive := hasLive && version.GetID() == live.GetID()
		if condition.Evaluate(target.WithVersion(string(version.Kind), isLive).Get()) == pluginport.ConditionTrue {
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
	values := pluginport.NewTargetValues(path, false, u.types.GetContentType(path))
	return listVersionItems(condition, values, versions, live, hasLive), nil
}

// ListEveryVersion lists every text version of the file at the path, as the items of the application
// command DeleteCommand.
func (u *FileMenu) ListEveryVersion(ctx context.Context, v domain.Vault, path string) ([]VersionItem, error) {
	versions, err := u.versions.getVersions(ctx, v, path)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.findLiveVersion(ctx, v, path, versions)
	if err != nil {
		return nil, err
	}
	items := make([]VersionItem, 0, len(versions))
	for _, version := range versions {
		isLive := hasLive && version.GetID() == live.GetID()
		items = append(items, VersionItem{Version: version.GetID(), Kind: string(version.Kind), IsLive: isLive})
	}
	return items, nil
}

// DeleteVersion deletes the version of the file at the path, its record and its file; see
// Versions.DeleteVersion.
func (u *FileMenu) DeleteVersion(ctx context.Context, v domain.Vault, path string, version domain.TextVersionID) error {
	return u.versions.DeleteVersion(ctx, v, path, version)
}

// DeleteCommand is the application command that deletes a text version of a file. The core serves
// it with no plugin: its items are every version of the file.
const DeleteCommand = "app:version.delete"

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
	return u.run.Execute(ctx, v, pluginport.PluginCommandCall{
		PluginID: entry.PluginID, Command: entry.Command.ID, Path: path,
		Arguments: map[string]string{pluginport.CommandArgumentVersion: string(version)},
	})
}

// The errors of a menu entry the file's menu does not hold, for a driving adapter.
var (
	ErrNoText            = textport.ErrNoText
	ErrNotTextPlugin     = textport.ErrNotTextPlugin
	ErrInvalidVersion    = textport.ErrInvalidVersion
	ErrSourceNotAccepted = textport.ErrSourceNotAccepted
)
