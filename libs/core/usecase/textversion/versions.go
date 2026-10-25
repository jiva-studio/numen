package textversion

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// Versions is the text versions of vault files: what plugins report, what is
// live, and what a file's menu offers.
type Versions struct {
	repos     textport.VersionRepositories
	plugins   textport.TextPlugins
	providers textport.TextProviders
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

// NewVersions is the versions kept in repos, the plugins that make text and
// their providers, and the clock a new version is dated by. A change is told to
// no one until SetListener names who hears it.
func NewVersions(
	repos textport.VersionRepositories,
	plugins textport.TextPlugins,
	providers textport.TextProviders,
	clock machineport.Clock,
) (*Versions, error) {
	if repos == nil || plugins == nil || providers == nil || clock == nil {
		return nil, errors.New("text versions need a repository, plugins, providers and a clock")
	}
	return &Versions{
		repos: repos, plugins: plugins, providers: providers, clock: clock,
		listener: noListener{}, requested: map[request]bool{},
	}, nil
}

// SetListener names who hears that the versions of a file changed: the index of
// the vault the window shows. Nil hears nothing.
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

// VersionPut is what a plugin reports of one of its versions.
type VersionPut struct {
	Path      string
	Key       string
	Input     domain.TextVersionID
	Format    domain.TextFormat
	InputHash string
	Hash      string
}

// PutVersion records a version of the plugin, or replaces the record of the
// one it has under the key. A replaced version keeps the time it was first
// made.
func (u *Versions) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params VersionPut) (domain.TextVersion, error) {
	if _, ok := u.findPlugin(pluginID); !ok {
		return domain.TextVersion{}, fmt.Errorf("%w: %s", textport.ErrNotTextPlugin, pluginID)
	}
	if params.Path == "" || params.Key == "" || !params.Format.IsValid() {
		return domain.TextVersion{}, fmt.Errorf("%w: path %q, key %q, format %q", textport.ErrInvalidVersion, params.Path, params.Key, params.Format)
	}
	repo, err := u.repos.Open(v)
	if err != nil {
		return domain.TextVersion{}, err
	}
	versions, err := repo.GetVersions(ctx, params.Path)
	if err != nil {
		return domain.TextVersion{}, err
	}
	version := domain.TextVersion{
		Path: params.Path, PluginID: pluginID, Key: params.Key, Input: params.Input,
		Format: params.Format, InputHash: params.InputHash, Hash: params.Hash, CreateTime: u.clock(),
	}
	if params.Input == version.GetID() || (params.Input != "" && !hasVersion(versions, params.Input)) {
		return domain.TextVersion{}, fmt.Errorf("%w: %s is no other version of %s", textport.ErrInvalidVersion, params.Input, params.Path)
	}
	if i := indexOf(versions, version.GetID()); i >= 0 {
		version.CreateTime = versions[i].CreateTime
	}
	if err := repo.SaveVersion(ctx, version); err != nil {
		return domain.TextVersion{}, fmt.Errorf("save version %s of %s: %w", version.GetID(), params.Path, err)
	}
	u.reportChanged(v, params.Path)
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
	if live, ok := domain.FindLiveVersion(versions, fileHash, u.isEnabled); ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

// DeleteVersion asks the plugin that made a version to delete its text, and
// then removes the record. The versions made from it stay, and are not live.
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
	if _, ok := u.findPlugin(version.PluginID); !ok {
		return fmt.Errorf("%w: %s", textport.ErrNotTextPlugin, version.PluginID)
	}
	provider, err := u.providers.OpenProvider(ctx, version.PluginID)
	if err != nil {
		return err
	}
	if err := provider.DeleteText(ctx, v.ID, path, version.Key); err != nil {
		return fmt.Errorf("delete text %s of %s: %w", id, path, err)
	}
	return u.RemoveVersion(ctx, v, version.PluginID, path, version.Key)
}

// CreateText asks a plugin, for a person, for text of a file, or of one of its
// versions where input names one. The plugin has to take the input it is given.
func (u *Versions) CreateText(ctx context.Context, v domain.Vault, pluginID, path string, input domain.TextVersionID) error {
	return u.createText(ctx, v, textport.TextCreation{Vault: v.ID, Path: path, Input: input, IsUserStarted: true}, pluginID)
}

func (u *Versions) createText(ctx context.Context, v domain.Vault, req textport.TextCreation, pluginID string) error {
	path, input := req.Path, req.Input
	plugin, ok := u.findPlugin(pluginID)
	if !ok {
		return fmt.Errorf("%w: %s", textport.ErrNotTextPlugin, pluginID)
	}
	if input == "" && !plugin.Files.IsFileAccepted(path) {
		return fmt.Errorf("%w: %s does not read %s", textport.ErrInputNotAccepted, pluginID, path)
	}
	if input != "" {
		versions, err := u.getVersions(ctx, v, path)
		if err != nil {
			return err
		}
		i := indexOf(versions, input)
		if i < 0 {
			return fmt.Errorf("%w: version %s of %s", textport.ErrNoText, input, path)
		}
		if !slices.Contains(plugin.Formats, versions[i].Format) {
			return fmt.Errorf("%w: %s does not read %s text", textport.ErrInputNotAccepted, pluginID, versions[i].Format)
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
	if _, ok := domain.FindLiveVersion(versions, fileHash, u.isEnabled); ok {
		return nil
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

// ReportPluginChanged tells the listener of every file whose versions a plugin
// switched on or off made live or dead.
func (u *Versions) ReportPluginChanged(ctx context.Context, v domain.Vault, pluginID string) error {
	repo, err := u.repos.Open(v)
	if err != nil {
		return err
	}
	all, err := repo.ListVersions(ctx)
	if err != nil {
		return err
	}
	byPath := map[string][]domain.TextVersion{}
	var paths []string
	for _, one := range all {
		if _, ok := byPath[one.Path]; !ok {
			paths = append(paths, one.Path)
		}
		byPath[one.Path] = append(byPath[one.Path], one)
	}
	for _, path := range paths {
		versions := byPath[path]
		if slices.ContainsFunc(versions, func(one domain.TextVersion) bool {
			return domain.HasPluginInChain(versions, one.GetID(), pluginID)
		}) {
			u.reportChanged(v, path)
		}
	}
	return nil
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

func (u *Versions) isEnabled(pluginID string) bool {
	_, ok := u.findPlugin(pluginID)
	return ok
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
