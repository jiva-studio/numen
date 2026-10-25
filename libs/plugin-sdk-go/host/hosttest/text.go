package hosttest

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// Texts is the text versions of a vault in memory, worked as one plugin works them through the
// host. The live version is the one the core would show: newest, made by enabled plugins, from
// inputs that are unchanged.
type Texts struct {
	world    *textWorld
	pluginID string
}

// textWorld is what every plugin's view of one vault's texts shares.
type textWorld struct {
	mu        sync.Mutex
	vault     domain.VaultID
	versions  []domain.TextVersion
	files     map[string][]byte
	providers map[string]host.TextProvider
	disabled  map[string]bool
	changes   []string
	clock     time.Time
}

var _ host.Texts = (*Texts)(nil)

// NewTexts is the texts of the vault of the identity, as the plugin of the id works them.
func NewTexts(vault domain.VaultID, pluginID string) *Texts {
	return &Texts{world: &textWorld{
		vault: vault, files: map[string][]byte{}, providers: map[string]host.TextProvider{}, disabled: map[string]bool{},
		clock: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}, pluginID: pluginID}
}

// As is the same texts, as another plugin works them.
func (t *Texts) As(pluginID string) *Texts { return &Texts{world: t.world, pluginID: pluginID} }

// File puts a file's bytes in the vault: what a version made from the file is compared with.
func (t *Texts) File(path string, raw []byte) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.files[path] = raw
}

// Provide is the provider the host reads a plugin's texts from.
func (t *Texts) Provide(pluginID string, provider host.TextProvider) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.providers[pluginID] = provider
}

// SetEnabled switches a plugin on or off. The versions of a plugin that is off are not live, nor
// any version made from them.
func (t *Texts) SetEnabled(pluginID string, isEnabled bool) {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	t.world.disabled[pluginID] = !isEnabled
}

// TakeChanges is every file whose versions changed since the last time it was asked, in order.
func (t *Texts) TakeChanges() []string {
	t.world.mu.Lock()
	defer t.world.mu.Unlock()
	out := t.world.changes
	t.world.changes = nil
	return out
}

func (t *Texts) PutVersion(_ context.Context, params host.TextPut) (domain.TextVersion, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	if params.Path == "" || params.Key == "" || !params.Format.IsValid() {
		return domain.TextVersion{}, host.ErrInvalidVersion
	}
	version := domain.TextVersion{
		Path: params.Path, PluginID: t.pluginID, Key: params.Key, Input: params.Input, Format: params.Format,
		InputHash: params.InputHash, Hash: params.Hash,
	}
	if params.Input == version.GetID() || (params.Input != "" && w.indexOf(params.Path, params.Input) < 0) {
		return domain.TextVersion{}, host.ErrInvalidVersion
	}
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
	id := domain.NewTextVersionID(t.pluginID, key)
	w.versions = slices.DeleteFunc(w.versions, func(v domain.TextVersion) bool { return v.Path == path && v.GetID() == id })
	w.changes = append(w.changes, path)
	return nil
}

func (t *Texts) ListVersions(_ context.Context, path string) (host.TextListing, error) {
	w := t.world
	w.mu.Lock()
	defer w.mu.Unlock()
	var listing host.TextListing
	for _, v := range w.versions {
		if v.Path == path {
			listing.Versions = append(listing.Versions, v)
		}
	}
	isEnabled := func(pluginID string) bool { return !w.disabled[pluginID] }
	if live, ok := domain.FindLiveVersion(listing.Versions, domain.HashContent(w.files[path]), isEnabled); ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

func (t *Texts) ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error) {
	w := t.world
	w.mu.Lock()
	i := w.indexOf(path, id)
	if i < 0 {
		w.mu.Unlock()
		return host.TextPage{}, host.ErrNotExist
	}
	version := w.versions[i]
	provider, ok := w.providers[version.PluginID]
	vault := w.vault
	w.mu.Unlock()
	if !ok {
		return host.TextPage{}, fmt.Errorf("hosttest: no provider of %s: %w", version.PluginID, host.ErrNotExist)
	}
	return provider.GetText(ctx, host.TextRequest{Vault: vault, Path: path, Key: version.Key, Cursor: cursor})
}

func (w *textWorld) indexOf(path string, id domain.TextVersionID) int {
	return slices.IndexFunc(w.versions, func(v domain.TextVersion) bool { return v.Path == path && v.GetID() == id })
}

// MemoryProvider is a TextProvider over texts held in memory, a page a part. CreateText makes
// nothing; a test puts each text with Hold.
type MemoryProvider struct {
	mu    sync.Mutex
	texts map[string][]domain.TextPart
}

var _ host.TextProvider = (*MemoryProvider)(nil)

// NewMemoryProvider is a provider that holds no text yet.
func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{texts: map[string][]domain.TextPart{}}
}

// Hold keeps a text of a file under a key.
func (p *MemoryProvider) Hold(path, key string, parts []domain.TextPart) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.texts[path+"\x00"+key] = parts
}

func (p *MemoryProvider) GetText(_ context.Context, req host.TextRequest) (host.TextPage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	parts, ok := p.texts[req.Path+"\x00"+req.Key]
	if !ok {
		return host.TextPage{}, host.ErrNotExist
	}
	from := 0
	if req.Cursor != "" {
		if _, err := fmt.Sscan(req.Cursor, &from); err != nil || from < 0 || from > len(parts) {
			return host.TextPage{}, fmt.Errorf("hosttest: cursor %q: %w", req.Cursor, host.ErrNotExist)
		}
	}
	if from >= len(parts) {
		return host.TextPage{}, nil
	}
	page := host.TextPage{Parts: parts[from : from+1]}
	if from+1 < len(parts) {
		page.NextCursor = fmt.Sprint(from + 1)
	}
	return page, nil
}

func (p *MemoryProvider) CreateText(context.Context, host.TextCreation) error { return nil }

func (p *MemoryProvider) DeleteText(_ context.Context, _ domain.VaultID, path, key string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.texts, path+"\x00"+key)
	return nil
}
