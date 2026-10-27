package hosttest

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/textfile"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Texts is the text versions of a vault in memory, worked as one plugin works them through the
// host. A version's file is read from the folder of the plugin that made it, as the host reads it:
// the store SetStore names, or a folder in memory. The live version is the one the core would show:
// the newest whose sources are unchanged and whose file still has the hash it was recorded with.
// With reaches the texts of another vault of the same host, made empty on first use.
type Texts struct {
	world    *textWorld
	pluginID string
}

// textHost is the texts of every vault one host keeps, by identity.
type textHost struct {
	mu     sync.Mutex
	worlds map[host.VaultID]*textWorld
}

// textWorld is what every plugin's view of one vault's texts shares.
type textWorld struct {
	mu       sync.Mutex
	all      *textHost
	vault    host.VaultID
	versions []host.TextVersion
	files    map[string][]byte
	stores   map[string]host.Store
	changes  []string
	clock    time.Time
}

var _ host.Texts = (*Texts)(nil)

// NewTexts is the texts of the vault of the identity, as the plugin of the id works them.
func NewTexts(vault host.VaultID, pluginID string) *Texts {
	vaults := &textHost{worlds: map[host.VaultID]*textWorld{}}
	return &Texts{world: vaults.getWorld(vault), pluginID: pluginID}
}

// getWorld is the texts of the vault, made empty on first use.
func (h *textHost) getWorld(vault host.VaultID) *textWorld {
	h.mu.Lock()
	defer h.mu.Unlock()
	w, ok := h.worlds[vault]
	if !ok {
		w = &textWorld{
			all: h, vault: vault, files: map[string][]byte{}, stores: map[string]host.Store{},
			clock: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		h.worlds[vault] = w
	}
	return w
}

func (t *Texts) With(id host.VaultID) host.Texts {
	return &Texts{world: t.world.all.getWorld(id), pluginID: t.pluginID}
}

// As is the same texts, as another plugin works them.
func (t *Texts) As(pluginID string) *Texts { return &Texts{world: t.world, pluginID: pluginID} }

// File puts a file's bytes in the vault: what a version made from the file is compared with.
func (t *Texts) File(path string, raw []byte) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.files[path] = raw
}

// SetStore is the folder the host reads a plugin's version files from.
func (t *Texts) SetStore(pluginID string, store host.Store) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.stores[pluginID] = store
}

// GetStore is the folder of a plugin's version files: the one SetStore named, or a folder in
// memory made on first use.
func (t *Texts) GetStore(pluginID string) host.Store {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	return t.world.getStore(pluginID)
}

// getStore is GetStore; the caller holds the lock.
func (w *textWorld) getStore(pluginID string) host.Store {
	store, ok := w.stores[pluginID]
	if !ok {
		store = New(pluginID).Store()
		w.stores[pluginID] = store
	}
	return store
}

// TakeChanges is every file whose versions changed since the last time it was asked, in order.
func (t *Texts) TakeChanges() []string {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	out := t.world.changes
	t.world.changes = nil
	return out
}

func (t *Texts) PutVersion(ctx context.Context, params host.TextPut) (host.TextVersion, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	if params.Path == "" || params.Key == "" || params.Kind == "" || params.File == "" {
		return host.TextVersion{}, host.ErrInvalidVersion
	}
	version := host.TextVersion{
		Path: params.Path, PluginID: t.pluginID, Key: params.Key, Source: params.Source, Kind: params.Kind,
		File: params.File, SourceHash: params.SourceHash,
	}
	if params.Source == version.GetID() || (params.Source != "" && w.indexOf(params.Path, params.Source) < 0) {
		return host.TextVersion{}, host.ErrInvalidVersion
	}
	_, hash, err := w.readFile(ctx, version)
	if err != nil {
		return host.TextVersion{}, errors.Join(host.ErrInvalidVersion, err)
	}
	version.Hash = hash
	if i := w.indexOf(params.Path, version.GetID()); i >= 0 {
		version.CreateTime = w.versions[i].CreateTime
		w.versions[i] = version
	} else {
		w.clock = w.clock.Add(time.Second)
		version.CreateTime = w.clock
		w.versions = append(w.versions, version)
	}
	w.changes = append(w.changes, params.Path)
	return version, nil
}

func (t *Texts) RemoveVersion(_ context.Context, path, key string) error {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	id := host.NewTextVersionID(t.pluginID, key)
	w.versions = slices.DeleteFunc(w.versions, func(v host.TextVersion) bool { return v.Path == path && v.GetID() == id })
	w.changes = append(w.changes, path)
	return nil
}

func (t *Texts) ListVersions(ctx context.Context, path string) (host.TextListing, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	var listing host.TextListing
	for _, v := range w.versions {
		if v.Path == path {
			listing.Versions = append(listing.Versions, v)
		}
	}
	isIntact := func(v host.TextVersion) bool {
		_, hash, err := w.readFile(ctx, v)
		return err == nil && hash == v.Hash
	}
	if live, ok := host.FindLiveVersion(listing.Versions, host.HashContent(w.files[path]), isIntact); ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

// ReadText returns every segment of the version in one page.
func (t *Texts) ReadText(ctx context.Context, path string, id host.TextVersionID, _ string) (host.TextPage, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	i := w.indexOf(path, id)
	if i < 0 {
		return host.TextPage{}, host.ErrNotExist
	}
	segments, hash, err := w.readFile(ctx, w.versions[i])
	if err != nil || hash != w.versions[i].Hash {
		return host.TextPage{}, host.ErrNotExist
	}
	return host.TextPage{Segments: segments}, nil
}

// readFile is the segments of a version's file and the hash of its bytes. The caller holds the lock.
func (w *textWorld) readFile(ctx context.Context, v host.TextVersion) ([]host.TextSegment, string, error) {
	raw, err := w.getStore(v.PluginID).Read(ctx, v.File)
	if err != nil {
		return nil, "", err
	}
	segments, err := textfile.Decode(raw)
	if err != nil {
		return nil, "", err
	}
	return segments, host.HashContent(raw), nil
}

func (w *textWorld) indexOf(path string, id host.TextVersionID) int {
	return slices.IndexFunc(w.versions, func(v host.TextVersion) bool { return v.Path == path && v.GetID() == id })
}
