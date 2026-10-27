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
	"github.com/jiva-studio/numen/libs/core/usecase/file"
)

// Versions is the text versions of vault files: what plugins report, what is
// live, and what a file's menu offers.
type Versions struct {
	repos     textport.VersionRepositories
	files     textport.TextFileStores
	plugins   textport.TextPlugins
	providers textport.TextProviders
	readers   vaultport.VaultReaders
	clock     machineport.Clock

	mu        sync.Mutex
	listener  textport.VersionListener
	requested map[request]bool
}

// request is one plugin asked, on its own, for text of one content of a file.
type request struct {
	vault    domain.VaultID
	path     string
	fileHash string
	pluginID string
}

// NewVersions is the versions recorded in repos with their files in files, the
// plugins that make text and their providers, the vault files a note's type is
// read from, and the clock a new version is dated by. A change is reported to
// no one until SetListener names a listener.
func NewVersions(
	repos textport.VersionRepositories,
	files textport.TextFileStores,
	plugins textport.TextPlugins,
	providers textport.TextProviders,
	readers vaultport.VaultReaders,
	clock machineport.Clock,
) (*Versions, error) {
	if repos == nil || files == nil || plugins == nil || providers == nil || readers == nil || clock == nil {
		return nil, errors.New("text versions need a repository, files, plugins, providers, vault readers and a clock")
	}
	return &Versions{
		repos: repos, files: files, plugins: plugins, providers: providers, readers: readers, clock: clock,
		listener: noListener{}, requested: map[request]bool{},
	}, nil
}

// SetListener names the listener of version changes: the index of the vault
// the window shows. Nil reports to no one.
func (u *Versions) SetListener(listener textport.VersionListener) {
	if listener == nil {
		listener = noListener{}
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.listener = listener
}

// reportChanged tells the listener that the versions of a file changed.
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
	if _, ok := u.findPlugin(pluginID); !ok {
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
	if version.Path == "" || version.Key == "" || version.Kind == "" || version.File == "" {
		return domain.TextVersion{}, fmt.Errorf("%w: path %q, key %q, kind %q, file %q",
			textport.ErrInvalidVersion, version.Path, version.Key, version.Kind, version.File)
	}
	repo, err := u.repos.Open(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	versions, err := repo.GetVersions(ctx, version.Path)
	if err != nil {
		return domain.TextVersion{}, err
	}
	if version.Source == version.GetID() || (version.Source != "" && !hasVersion(versions, version.Source)) {
		return domain.TextVersion{}, fmt.Errorf("%w: %s is no other version of %s", textport.ErrInvalidVersion, version.Source, version.Path)
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
	version.CreateTime = u.clock()
	if i := indexOf(versions, version.GetID()); i >= 0 {
		version.CreateTime = versions[i].CreateTime
	}
	if err := repo.SaveVersion(ctx, version); err != nil {
		return domain.TextVersion{}, fmt.Errorf("save version %s of %s: %w", version.GetID(), version.Path, err)
	}
	u.reportChanged(v, version.Path)
	return version, nil
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

// ListVersions is the versions of a file and which of them is live, for the
// file as fileHash says it is now.
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

// DeleteVersion removes a version's file and its record. The versions made
// from it stay, and are not live.
func (u *Versions) DeleteVersion(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) error {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return err
	}
	i := indexOf(versions, id)
	if i < 0 {
		return fmt.Errorf("%w: version %s of %s", textport.ErrNoText, id, path)
	}
	version := versions[i]
	files, err := u.files.Open(v)
	if err != nil {
		return err
	}
	if err := files.DeleteTextFile(ctx, version.PluginID, version.File); err != nil {
		return fmt.Errorf("delete text %s of %s: %w", id, path, err)
	}
	return u.RemoveVersion(ctx, v, version.PluginID, path, version.Key)
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

// TextEdit is a person's edit of a version of a file: the segments as the
// person left them, made from Source in the view of PluginID.
type TextEdit struct {
	PluginID string
	Path     string
	Source   domain.TextVersionID
	Segments []domain.TextSegment
}

// editPrefix starts the key of a version a person made by editing another.
const editPrefix = "edit-"

// CreateEdit records a person's edit as a new version made from its source.
// The core writes the segments into the folder of the plugin the person edits
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
	key := editPrefix + strconv.FormatInt(u.clock().UnixNano(), 36)
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

// CreateText asks a plugin, for a person, for text of a file, or of one of its
// versions where source names one. The plugin has to take the source it is given.
func (u *Versions) CreateText(ctx context.Context, v domain.Vault, pluginID, path string, source domain.TextVersionID) error {
	return u.createText(ctx, v, textport.TextCreation{Vault: v.ID, Path: path, Source: source, IsUserStarted: true}, pluginID)
}

func (u *Versions) createText(ctx context.Context, v domain.Vault, req textport.TextCreation, pluginID string) error {
	path, source := req.Path, req.Source
	plugin, ok := u.findPlugin(pluginID)
	if !ok {
		return fmt.Errorf("%w: %s", textport.ErrNotTextPlugin, pluginID)
	}
	if source == "" && !u.isTaking(ctx, v, plugin, path) {
		return fmt.Errorf("%w: %s does not read %s", textport.ErrSourceNotAccepted, pluginID, path)
	}
	if source != "" {
		versions, err := u.getVersions(ctx, v, path)
		if err != nil {
			return err
		}
		i := indexOf(versions, source)
		if i < 0 {
			return fmt.Errorf("%w: version %s of %s", textport.ErrNoText, source, path)
		}
		if !slices.Contains(plugin.Kinds, versions[i].Kind) {
			return fmt.Errorf("%w: %s does not read %s text", textport.ErrSourceNotAccepted, pluginID, versions[i].Kind)
		}
	}
	provider, err := u.providers.OpenProvider(ctx, pluginID)
	if err != nil {
		return err
	}
	if err := provider.CreateText(ctx, req); err != nil {
		return fmt.Errorf("create text of %s with %s: %w", path, pluginID, err)
	}
	return nil
}

// RequestText asks each enabled plugin that runs on its own and reads the file
// for text of it, once for each content of the file. A file with a live
// version is asked for nothing.
func (u *Versions) RequestText(ctx context.Context, v domain.Vault, path, fileHash string) error {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return err
	}
	if _, ok, err := u.findLive(ctx, v, versions, fileHash); err != nil || ok {
		return err
	}
	var failures []error
	for _, plugin := range u.plugins.GetTextPlugins() {
		if !plugin.IsAutoRun || !plugin.Files.IsFileAccepted(path) {
			continue
		}
		key := request{vault: v.ID, path: path, fileHash: fileHash, pluginID: plugin.ID}
		if !u.markRequested(key) {
			continue
		}
		if err := u.createText(ctx, v, textport.TextCreation{Vault: v.ID, Path: path}, plugin.ID); err != nil {
			u.clearRequested(key)
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// IsTextSource reports whether the index keeps the text of the file: it has text
// versions, or an enabled plugin that runs on its own takes it.
func (u *Versions) IsTextSource(ctx context.Context, v domain.Vault, path string) (bool, error) {
	for _, plugin := range u.plugins.GetTextPlugins() {
		if plugin.IsAutoRun && plugin.Files.IsFileAccepted(path) {
			return true, nil
		}
	}
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

func (u *Versions) findPlugin(pluginID string) (textport.TextPlugin, bool) {
	for _, plugin := range u.plugins.GetTextPlugins() {
		if plugin.ID == pluginID {
			return plugin, true
		}
	}
	return textport.TextPlugin{}, false
}

// markRequested records a request, and reports whether it was not recorded before.
func (u *Versions) markRequested(key request) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.requested[key] {
		return false
	}
	u.requested[key] = true
	return true
}

func (u *Versions) clearRequested(key request) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.requested, key)
}

func indexOf(versions []domain.TextVersion, id domain.TextVersionID) int {
	return slices.IndexFunc(versions, func(one domain.TextVersion) bool { return one.GetID() == id })
}

func hasVersion(versions []domain.TextVersion, id domain.TextVersionID) bool {
	return indexOf(versions, id) >= 0
}

// isTaking reports whether the plugin's text input takes the file at the path: a file of its
// content types, or a note of its note types.
func (u *Versions) isTaking(ctx context.Context, v domain.Vault, plugin textport.TextPlugin, path string) bool {
	if plugin.Files.IsFileAccepted(path) {
		return true
	}
	if len(plugin.NoteTypes) == 0 {
		return false
	}
	noteType, err := file.ReadNoteType(ctx, u.readers, v, path)
	return err == nil && slices.Contains(plugin.NoteTypes, noteType)
}
