package textversion

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

// NoteReader reads the type and the frontmatter of a note from its current file.
type NoteReader interface {
	Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteType, map[string]any, error)
}

// Versions records, replaces and removes the text versions of vault files, and keeps the files of
// those versions. Each change is reported to the Changes it was built with.
type Versions struct {
	stores  textport.VersionStores
	files   textport.TextFileStores
	plugins textport.TextPlugins
	clock   machineport.Clock
	changes versionReporter
}

// versionReporter receives the files whose versions changed.
type versionReporter interface {
	report(ctx context.Context, v domain.Vault, path string)
}

// NewVersions returns the versions use case over the version stores, the file stores, the enabled plugins
// that may record versions and the clock that dates new versions. changes receives every change.
func NewVersions(
	stores textport.VersionStores,
	files textport.TextFileStores,
	plugins textport.TextPlugins,
	clock machineport.Clock,
	changes *Changes,
) *Versions {
	return &Versions{stores: stores, files: files, plugins: plugins, clock: clock, changes: changes}
}

// PutVersion records a version of the plugin, or replaces the record of the
// one it has under the key. The core reads the version's file and records the
// hash of its bytes. A replaced version keeps the time it was first made.
func (u *Versions) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error) {
	if !u.plugins.IsEnabled(pluginID) {
		return domain.TextVersion{}, fmt.Errorf("%w: %s", text.ErrNotTextPlugin, pluginID)
	}
	return u.saveVersion(ctx, v, domain.TextVersion{
		Path: params.Path, PluginID: pluginID, Key: params.Key, Source: params.Source,
		SourceProperty: params.SourceProperty, Kind: params.Kind, File: params.File, InputHash: params.InputHash,
	})
}

func (u *Versions) saveVersion(ctx context.Context, v domain.Vault, version domain.TextVersion) (domain.TextVersion, error) {
	if err := checkVersionFields(version); err != nil {
		return domain.TextVersion{}, err
	}
	reader, err := u.stores.OpenVersionReader(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	versions, err := reader.GetVersions(ctx, version.Path)
	if err != nil {
		return domain.TextVersion{}, err
	}
	if err := checkVersionSource(version, versions); err != nil {
		return domain.TextVersion{}, err
	}
	files, err := u.files.Open(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	file, err := files.ReadTextFile(ctx, version.PluginID, version.File)
	if err != nil {
		return domain.TextVersion{}, fmt.Errorf("%w: file %s of %s: %w", text.ErrInvalidVersion, version.File, version.PluginID, err)
	}
	version.TextHash = file.TextHash
	version.CreateTime = u.clock.Now()
	if i := indexOf(versions, version.GetID()); i >= 0 {
		version.CreateTime = versions[i].CreateTime
	}
	writer, err := u.stores.OpenVersionWriter(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	if err := writer.SaveVersion(ctx, version); err != nil {
		return domain.TextVersion{}, fmt.Errorf("save version %s of %s: %w", version.GetID(), version.Path, err)
	}
	u.changes.report(ctx, v, version.Path)
	return version, nil
}

func checkVersionFields(version domain.TextVersion) error {
	if version.Path == "" || version.Key == "" || version.Kind == "" || version.File == "" {
		return fmt.Errorf("%w: path %q, key %q, kind %q, file %q",
			text.ErrInvalidVersion, version.Path, version.Key, version.Kind, version.File)
	}
	return nil
}

func checkVersionSource(version domain.TextVersion, versions []domain.TextVersion) error {
	if version.IsFromProperty() && (version.Source != "" || version.InputHash != "") {
		return fmt.Errorf("%w: a version made from the property %s of %s has another source",
			text.ErrInvalidVersion, version.SourceProperty.Key, version.Path)
	}
	if version.Source == version.GetID() || (version.Source != "" && !hasVersion(versions, version.Source)) {
		return fmt.Errorf("%w: %s is no other version of %s", text.ErrInvalidVersion, version.Source, version.Path)
	}
	return nil
}

// DeleteVersion deletes the version of the file at the path, whichever plugin made it: its record and
// its file in the folder of that plugin. The listener is notified that the file's versions changed. A version
// the file does not have is ErrInvalidVersion.
func (u *Versions) DeleteVersion(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) error {
	versions, err := getVersions(ctx, u.stores, v, path)
	if err != nil {
		return err
	}
	index := indexOf(versions, id)
	if index < 0 {
		return fmt.Errorf("%w: %s has no version %s", text.ErrInvalidVersion, path, id)
	}
	version := versions[index]
	writer, err := u.stores.OpenVersionWriter(v)
	if err != nil {
		return err
	}
	if err := writer.RemoveVersion(ctx, path, id); err != nil {
		return fmt.Errorf("remove version %s of %s: %w", id, path, err)
	}
	files, err := u.files.Open(v)
	if err != nil {
		return err
	}
	if err := files.DeleteTextFile(ctx, version.PluginID, version.File); err != nil {
		return fmt.Errorf("delete the file %s of %s: %w", version.File, version.PluginID, err)
	}
	u.changes.report(ctx, v, path)
	return nil
}

// DeletePluginVersions removes the records of every version a plugin made in a
// vault, after its data was deleted, and reports each file whose versions changed.
func (u *Versions) DeletePluginVersions(ctx context.Context, v domain.Vault, pluginID string) error {
	writer, err := u.stores.OpenVersionWriter(v)
	if err != nil {
		return err
	}
	removed, err := writer.RemovePluginVersions(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("remove the versions of %s: %w", pluginID, err)
	}
	var paths []string
	for _, one := range removed {
		if !slices.Contains(paths, one.Path) {
			paths = append(paths, one.Path)
			u.changes.report(ctx, v, one.Path)
		}
	}
	return nil
}

// editPrefix starts the key of a version created by editing another version.
const editPrefix = "edit-"

// CreateEdit records a user's edit as a new version derived from its source.
// The core writes the segments into the folder of the plugin the user edits
// in. The new version is the newest, and live while its chain is valid.
func (u *Versions) CreateEdit(ctx context.Context, v domain.Vault, edit text.Edit) (domain.TextVersion, error) {
	versions, err := getVersions(ctx, u.stores, v, edit.Path)
	if err != nil {
		return domain.TextVersion{}, err
	}
	i := indexOf(versions, edit.Source)
	if i < 0 {
		return domain.TextVersion{}, fmt.Errorf("%w: version %s of %s", text.ErrNoText, edit.Source, edit.Path)
	}
	source := versions[i]
	files, err := u.files.Open(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	key := editPrefix + strconv.FormatInt(u.clock.Now().UnixNano(), 36)
	name := key + fileExtension
	if _, err := files.WriteTextFile(ctx, edit.PluginID, name, edit.Segments); err != nil {
		return domain.TextVersion{}, fmt.Errorf("write edit of %s: %w", edit.Path, err)
	}
	return u.saveVersion(ctx, v, domain.TextVersion{
		Path: edit.Path, PluginID: edit.PluginID, Key: key, Source: source.GetID(),
		Kind: source.Kind, File: name, InputHash: source.TextHash,
	})
}

// fileExtension ends the name of a version file the core writes.
const fileExtension = ".jsonl"

func indexOf(versions []domain.TextVersion, id domain.TextVersionID) int {
	return slices.IndexFunc(versions, func(one domain.TextVersion) bool { return one.GetID() == id })
}

func hasVersion(versions []domain.TextVersion, id domain.TextVersionID) bool {
	return indexOf(versions, id) >= 0
}
