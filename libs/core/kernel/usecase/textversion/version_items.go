package textversion

import (
	"context"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// VersionItems is the text versions of a vault file that the commands of the
// enabled plugins whose condition names version.* offer as their items, and
// the run of such a command over one of them.
type VersionItems struct {
	texts      itemTexts
	versions   versionRemover
	run        pluginport.CommandRun
	types      contenttypeport.ContentTypeTable
	conditions *when.Conditions
}

// itemTexts finds the versions of a file and which one is live.
type itemTexts interface {
	FindLive(ctx context.Context, v domain.Vault, path string) ([]domain.TextVersion, domain.TextVersion, bool, error)
}

// versionRemover deletes a version of a file.
type versionRemover interface {
	DeleteVersion(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) error
}

// NewVersionItems returns the version items of the vault files, run with run and using the table
// to find the type of a file.
func NewVersionItems(texts *Texts, versions *Versions, run pluginport.CommandRun, types contenttypeport.ContentTypeTable) *VersionItems {
	return &VersionItems{texts: texts, versions: versions, run: run, types: types, conditions: when.NewConditions()}
}

func listVersionItems(condition when.Condition, target plugin.TargetValues, versions []domain.TextVersion, live domain.TextVersion, hasLive bool) []text.VersionItem {
	var items []text.VersionItem
	for _, version := range versions {
		isLive := hasLive && version.GetID() == live.GetID()
		if condition.Evaluate(target.WithVersion(string(version.Kind), isLive).Get()) == when.True {
			items = append(items, text.VersionItem{
				Version: version.GetID(), Kind: string(version.Kind), IsLive: isLive,
				PluginID: version.PluginID, CreateTime: version.CreateTime,
			})
		}
	}
	return items
}

// ListVersionItems lists the text versions of the file at the path over which the when condition of
// the command holds; a command whose condition names no version.* variable has none.
func (u *VersionItems) ListVersionItems(ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry) ([]text.VersionItem, error) {
	condition, err := u.conditions.Parse(entry.Command.When)
	if err != nil || !condition.HasItems() {
		return nil, err
	}
	versions, live, hasLive, err := u.texts.FindLive(ctx, v, path)
	if err != nil {
		return nil, err
	}
	values := plugin.NewTargetValues(path, false, u.types.GetContentType(path))
	return listVersionItems(condition, values, versions, live, hasLive), nil
}

// ListEveryVersion lists every text version of the file at the path, as the items of the application
// command text.DeleteCommand.
func (u *VersionItems) ListEveryVersion(ctx context.Context, v domain.Vault, path string) ([]text.VersionItem, error) {
	versions, live, hasLive, err := u.texts.FindLive(ctx, v, path)
	if err != nil {
		return nil, err
	}
	items := make([]text.VersionItem, 0, len(versions))
	for _, version := range versions {
		isLive := hasLive && version.GetID() == live.GetID()
		items = append(items, text.VersionItem{
			Version: version.GetID(), Kind: string(version.Kind), IsLive: isLive,
			PluginID: version.PluginID, CreateTime: version.CreateTime,
		})
	}
	return items, nil
}

// DeleteVersion deletes the version of the file at the path, its record and its file; see
// Versions.DeleteVersion.
func (u *VersionItems) DeleteVersion(ctx context.Context, v domain.Vault, path string, version domain.TextVersionID) error {
	return u.versions.DeleteVersion(ctx, v, path, version)
}

// RunVersionItem runs the command of the entry over the version of the file at the path, which it gets
// as its argument version-id. A version the command does not offer over the file now, or a command
// without items, is ErrInvalidVersion.
func (u *VersionItems) RunVersionItem(
	ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry, version domain.TextVersionID,
) (plugin.CommandResult, error) {
	items, err := u.ListVersionItems(ctx, v, path, entry)
	if err != nil {
		return plugin.CommandResult{}, err
	}
	if !slices.ContainsFunc(items, func(item text.VersionItem) bool { return item.Version == version }) {
		return plugin.CommandResult{}, fmt.Errorf("%w: %s of %s offers no version %q over %s",
			text.ErrInvalidVersion, entry.Command.ID, entry.PluginID, version, path)
	}
	return u.run.Execute(ctx, v, plugin.CommandCall{
		PluginID: entry.PluginID, Command: entry.Command.ID, Path: path,
		Arguments: map[string]string{manifest.CommandArgumentVersion: string(version)},
	})
}

// The errors of the version items, for a driving adapter.
var ()
