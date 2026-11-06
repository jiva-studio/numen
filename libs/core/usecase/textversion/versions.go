package textversion

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Versions is the text versions of vault files: what plugins report and what is
// live.
type Versions struct {
	repos   textport.VersionRepositories
	files   textport.TextFileStores
	plugins textport.TextPlugins
	readers vaultport.VaultReaders
	clock   machineport.Clock

	mu       sync.Mutex
	listener textport.VersionListener
}

// NewVersions returns the versions use case. It takes the version repositories,
// the file stores, the enabled plugins that may record versions, the vault
// readers that read a note's type, and the clock that dates new versions. No
// listener is notified of changes until SetListener is called.
func NewVersions(
	repos textport.VersionRepositories,
	files textport.TextFileStores,
	plugins textport.TextPlugins,
	readers vaultport.VaultReaders,
	clock machineport.Clock,
) (*Versions, error) {
	if repos == nil || files == nil || plugins == nil || readers == nil || clock == nil {
		return nil, errors.New("text versions need a repository, files, plugins, vault readers and a clock")
	}
	return &Versions{
		repos: repos, files: files, plugins: plugins, readers: readers, clock: clock,
		listener: noListener{},
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

// reportChanged notifies the listener that the versions of a file changed.
func (u *Versions) reportChanged(v domain.Vault, path string) {
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
	Path       string
	Key        string
	Source     domain.TextVersionID
	Kind       domain.TextKind
	SourceHash string
	File       string
}

// PutVersion records a version of the plugin, or replaces the record of the
// one it has under the key. The core reads the version's file and records the
// hash of its bytes. A replaced version keeps the time it was first made.
func (u *Versions) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params VersionPut) (domain.TextVersion, error) {
	if !u.plugins.IsEnabled(pluginID) {
		return domain.TextVersion{}, fmt.Errorf("%w: %s", textport.ErrNotTextPlugin, pluginID)
	}
	return u.saveVersion(ctx, v, domain.TextVersion{
		Path: params.Path, PluginID: pluginID, Key: params.Key, Source: params.Source,
		Kind: params.Kind, File: params.File, SourceHash: params.SourceHash,
	})
}

// saveVersion checks a version against its file and the other versions of its
// path, and records it.
func (u *Versions) saveVersion(ctx context.Context, v domain.Vault, version domain.TextVersion) (domain.TextVersion, error) {
	if err := checkVersionFields(version); err != nil {
		return domain.TextVersion{}, err
	}
	repo, err := u.repos.Open(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	versions, err := repo.GetVersions(ctx, version.Path)
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
		return domain.TextVersion{}, fmt.Errorf("%w: file %s of %s: %w", textport.ErrInvalidVersion, version.File, version.PluginID, err)
	}
	version.Hash = file.Hash
	version.CreateTime = u.clock.Now()
	if i := indexOf(versions, version.GetID()); i >= 0 {
		version.CreateTime = versions[i].CreateTime
	}
	if err := repo.SaveVersion(ctx, version); err != nil {
		return domain.TextVersion{}, fmt.Errorf("save version %s of %s: %w", version.GetID(), version.Path, err)
	}
	u.reportChanged(v, version.Path)
	return version, nil
}

// checkVersionFields reports a version that lacks a path, key, kind or file.
func checkVersionFields(version domain.TextVersion) error {
	if version.Path == "" || version.Key == "" || version.Kind == "" || version.File == "" {
		return fmt.Errorf("%w: path %q, key %q, kind %q, file %q",
			textport.ErrInvalidVersion, version.Path, version.Key, version.Kind, version.File)
	}
	return nil
}

// checkVersionSource reports a source that is the version itself or is no version of the path.
func checkVersionSource(version domain.TextVersion, versions []domain.TextVersion) error {
	if version.Source == version.GetID() || (version.Source != "" && !hasVersion(versions, version.Source)) {
		return fmt.Errorf("%w: %s is no other version of %s", textport.ErrInvalidVersion, version.Source, version.Path)
	}
	return nil
}

// RemoveVersion removes the record of a version of the plugin.
func (u *Versions) RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error {
	repo, err := u.repos.Open(v)
	if err != nil {
		return err
	}
	if err := repo.RemoveVersion(ctx, path, domain.NewTextVersionID(pluginID, key)); err != nil {
		return fmt.Errorf("remove version %s/%s of %s: %w", pluginID, key, path, err)
	}
	u.reportChanged(v, path)
	return nil
}

// Listing is the versions of a file and the one that is live. Live is empty
// where none is.
type Listing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// ListVersions returns the versions of a file and the live one, evaluated for
// the file content identified by fileHash.
func (u *Versions) ListVersions(ctx context.Context, v domain.Vault, path, fileHash string) (Listing, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return Listing{}, err
	}
	listing := Listing{Versions: versions}
	live, ok, err := u.findLive(ctx, v, versions, fileHash)
	if err != nil {
		return Listing{}, err
	}
	if ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

// findLive is the live version among a file's versions: the newest one with a
// valid chain whose file still has the hash it was recorded with.
func (u *Versions) findLive(ctx context.Context, v domain.Vault, versions []domain.TextVersion, fileHash string) (domain.TextVersion, bool, error) {
	if len(versions) == 0 {
		return domain.TextVersion{}, false, nil
	}
	files, err := u.files.Open(v)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	live, ok := domain.FindLiveVersion(versions, fileHash, func(version domain.TextVersion) bool {
		file, err := files.ReadTextFile(ctx, version.PluginID, version.File)
		return err == nil && file.Hash == version.Hash
	})
	return live, ok, nil
}

// DeletePluginVersions removes the records of every version a plugin made in a
// vault, after its data was deleted, and reports each file whose versions changed.
func (u *Versions) DeletePluginVersions(ctx context.Context, v domain.Vault, pluginID string) error {
	repo, err := u.repos.Open(v)
	if err != nil {
		return err
	}
	removed, err := repo.RemovePluginVersions(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("remove the versions of %s: %w", pluginID, err)
	}
	var paths []string
	for _, one := range removed {
		if !slices.Contains(paths, one.Path) {
			paths = append(paths, one.Path)
			u.reportChanged(v, one.Path)
		}
	}
	return nil
}

// TextEdit is a user's edit of a version of a file: the edited segments, derived
// from Source in the view of PluginID.
type TextEdit struct {
	PluginID string
	Path     string
	Source   domain.TextVersionID
	Segments []domain.TextSegment
}

// editPrefix starts the key of a version created by editing another version.
const editPrefix = "edit-"

// CreateEdit records a user's edit as a new version derived from its source.
// The core writes the segments into the folder of the plugin the user edits
// in. The new version is the newest, and live while its chain is valid.
func (u *Versions) CreateEdit(ctx context.Context, v domain.Vault, edit TextEdit) (domain.TextVersion, error) {
	versions, err := u.getVersions(ctx, v, edit.Path)
	if err != nil {
		return domain.TextVersion{}, err
	}
	i := indexOf(versions, edit.Source)
	if i < 0 {
		return domain.TextVersion{}, fmt.Errorf("%w: version %s of %s", textport.ErrNoText, edit.Source, edit.Path)
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
		Kind: source.Kind, File: name, SourceHash: source.Hash,
	})
}

// fileExtension ends the name of a version file the core writes.
const fileExtension = ".jsonl"

// IsTextSource reports whether the index keeps the text of the file: it has text versions.
func (u *Versions) IsTextSource(ctx context.Context, v domain.Vault, path string) (bool, error) {
	versions, err := u.getVersions(ctx, v, path)
	return len(versions) > 0, err
}

func (u *Versions) getVersions(ctx context.Context, v domain.Vault, path string) ([]domain.TextVersion, error) {
	repo, err := u.repos.Open(v)
	if err != nil {
		return nil, err
	}
	return repo.GetVersions(ctx, path)
}

func indexOf(versions []domain.TextVersion, id domain.TextVersionID) int {
	return slices.IndexFunc(versions, func(one domain.TextVersion) bool { return one.GetID() == id })
}

func hasVersion(versions []domain.TextVersion, id domain.TextVersionID) bool {
	return indexOf(versions, id) >= 0
}
