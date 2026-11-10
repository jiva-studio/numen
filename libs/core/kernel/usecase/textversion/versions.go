package textversion

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// NoteReader reads the type and the frontmatter of a note from its current file.
type NoteReader interface {
	Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteType, map[string]any, error)
}

// Versions is the text versions of vault files: what plugins report and what is
// live.
type Versions struct {
	stores  textport.VersionStores
	files   textport.TextFileStores
	plugins textport.TextPlugins
	readers vaultport.VaultReaders
	notes   NoteReader
	clock   machineport.Clock

	mu       sync.Mutex
	listener textport.VersionListener
	lives    map[liveKey]domain.TextVersionID
}

// NewVersions returns the versions use case. It takes the version stores,
// the file stores, the enabled plugins that may record versions, the vault
// readers, the reader of a note's properties, and the clock that dates new versions. No
// listener is notified of changes until SetListener is called.
func NewVersions(
	stores textport.VersionStores,
	files textport.TextFileStores,
	plugins textport.TextPlugins,
	readers vaultport.VaultReaders,
	notes NoteReader,
	clock machineport.Clock,
) (*Versions, error) {
	if stores == nil || files == nil || plugins == nil || readers == nil || notes == nil || clock == nil {
		return nil, errors.New("text versions need version stores, files, plugins, vault readers, a note reader and a clock")
	}
	return &Versions{
		stores: stores, files: files, plugins: plugins, readers: readers, notes: notes, clock: clock,
		listener: noListener{}, lives: map[liveKey]domain.TextVersionID{},
	}, nil
}

// SetListener sets the listener of version changes, the index of the displayed
// vault. Nil disables notifications.
func (u *Versions) SetListener(listener textport.VersionListener) {
	if listener == nil {
		listener = noListener{}
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.listener = listener
}

func (u *Versions) reportChanged(ctx context.Context, v domain.Vault, path string) {
	if _, err := u.updateLive(ctx, v, path); err != nil {
		u.mu.Lock()
		delete(u.lives, liveKey{vault: v.ID, path: path})
		u.mu.Unlock()
	}
	u.notifyListener(v, path)
}

func (u *Versions) notifyListener(v domain.Vault, path string) {
	u.mu.Lock()
	listener := u.listener
	u.mu.Unlock()
	listener.ReportVersionsChanged(v, path)
}

type noListener struct{}

func (noListener) ReportVersionsChanged(domain.Vault, string) {}

// VersionPut is what a plugin reports of one of its versions. File is the name
// of the version's file in the plugin's folder, written before the put.
type VersionPut struct {
	Path           string
	Key            string
	Source         domain.TextVersionID
	SourceProperty domain.TextSourceProperty
	Kind           domain.TextKind
	InputHash      string
	File           string
}

// PutVersion records a version of the plugin, or replaces the record of the
// one it has under the key. The core reads the version's file and records the
// hash of its bytes. A replaced version keeps the time it was first made.
func (u *Versions) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params VersionPut) (domain.TextVersion, error) {
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
	u.reportChanged(ctx, v, version.Path)
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

// RemoveVersion removes the record of a version of the plugin.
func (u *Versions) RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error {
	writer, err := u.stores.OpenVersionWriter(v)
	if err != nil {
		return err
	}
	if err := writer.RemoveVersion(ctx, path, domain.NewTextVersionID(pluginID, key)); err != nil {
		return fmt.Errorf("remove version %s/%s of %s: %w", pluginID, key, path, err)
	}
	u.reportChanged(ctx, v, path)
	return nil
}

// DeleteVersion deletes the version of the file at the path, whichever plugin made it: its record and
// its file in the folder of that plugin. The listener is notified that the file's versions changed. A version
// the file does not have is ErrInvalidVersion.
func (u *Versions) DeleteVersion(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) error {
	versions, err := u.getVersions(ctx, v, path)
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
	u.reportChanged(ctx, v, path)
	return nil
}

// Listing is the versions of a file and the one that is live. Live is empty
// where none is.
type Listing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// ListVersions returns the versions of a file and the live one, evaluated for
// the file content identified by contentHash.
func (u *Versions) ListVersions(ctx context.Context, v domain.Vault, path, contentHash string) (Listing, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return Listing{}, err
	}
	listing := Listing{Versions: versions}
	live, ok, err := u.findLive(ctx, v, path, versions, contentHash)
	if err != nil {
		return Listing{}, err
	}
	if ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

func (u *Versions) findLive(
	ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion, contentHash string,
) (domain.TextVersion, bool, error) {
	if len(versions) == 0 {
		return domain.TextVersion{}, false, nil
	}
	properties, err := u.readSourceProperties(ctx, v, path, versions)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	files, err := u.files.Open(v)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	live, ok := domain.FindLiveVersion(versions, contentHash, properties, func(version domain.TextVersion) bool {
		file, err := files.ReadTextFile(ctx, version.PluginID, version.File)
		return err == nil && file.TextHash == version.TextHash
	})
	return live, ok, nil
}

func (u *Versions) readSourceProperties(
	ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion,
) (map[string]any, error) {
	if !slices.ContainsFunc(versions, domain.TextVersion.IsFromProperty) {
		return nil, nil
	}
	_, properties, err := u.notes.Execute(ctx, v, path)
	if err != nil {
		return nil, fmt.Errorf("read the properties of %s: %w", path, err)
	}
	return properties, nil
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
			u.reportChanged(ctx, v, one.Path)
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
	versions, err := u.getVersions(ctx, v, edit.Path)
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

// ListTargetVersions lists the versions of the file at the path as the when variable target.versions
// holds them, each marked valid where its source chain still matches the file, and the live one. The
// file is read for its content hash only where it has versions.
func (u *Versions) ListTargetVersions(ctx context.Context, v domain.Vault, path string) ([]plugin.TargetVersion, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil || len(versions) == 0 {
		return []plugin.TargetVersion{}, err
	}
	contentHash, err := u.GetContentHash(ctx, v, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	properties, err := u.readSourceProperties(ctx, v, path, versions)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.findLive(ctx, v, path, versions, contentHash)
	if err != nil {
		return nil, err
	}
	out := make([]plugin.TargetVersion, 0, len(versions))
	for _, one := range versions {
		out = append(out, plugin.TargetVersion{
			Kind: string(one.Kind), PluginID: one.PluginID, IsLive: hasLive && one.GetID() == live.GetID(),
			IsValid: domain.IsChainValid(versions, one.GetID(), contentHash, properties),
		})
	}
	return out, nil
}

// IsTextSource reports whether the index keeps the text of the file: it has text versions.
func (u *Versions) IsTextSource(ctx context.Context, v domain.Vault, path string) (bool, error) {
	versions, err := u.getVersions(ctx, v, path)
	return len(versions) > 0, err
}

func (u *Versions) getVersions(ctx context.Context, v domain.Vault, path string) ([]domain.TextVersion, error) {
	reader, err := u.stores.OpenVersionReader(v)
	if err != nil {
		return nil, err
	}
	return reader.GetVersions(ctx, path)
}

func indexOf(versions []domain.TextVersion, id domain.TextVersionID) int {
	return slices.IndexFunc(versions, func(one domain.TextVersion) bool { return one.GetID() == id })
}

func hasVersion(versions []domain.TextVersion, id domain.TextVersionID) bool {
	return indexOf(versions, id) >= 0
}
