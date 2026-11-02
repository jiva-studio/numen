package textversion_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/textfile"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// memoryVersions is every vault's versions, held in memory.
type memoryVersions struct {
	mu       sync.Mutex
	versions map[domain.VaultID][]domain.TextVersion
}

func (m *memoryVersions) Open(v domain.Vault) (textport.VersionRepository, error) {
	return memoryRepo{store: m, vault: v.ID}, nil
}

type memoryRepo struct {
	store *memoryVersions
	vault domain.VaultID
}

func (r memoryRepo) GetVersions(_ context.Context, path string) ([]domain.TextVersion, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	var out []domain.TextVersion
	for _, one := range r.store.versions[r.vault] {
		if one.Path == path {
			out = append(out, one)
		}
	}
	return out, nil
}

func (r memoryRepo) SaveVersion(_ context.Context, version domain.TextVersion) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if r.store.versions == nil {
		r.store.versions = map[domain.VaultID][]domain.TextVersion{}
	}
	list := r.store.versions[r.vault]
	for i, one := range list {
		if one.Path == version.Path && one.GetID() == version.GetID() {
			list[i] = version
			return nil
		}
	}
	r.store.versions[r.vault] = append(list, version)
	return nil
}

func (r memoryRepo) RemoveVersion(_ context.Context, path string, id domain.TextVersionID) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.versions[r.vault] = slices.DeleteFunc(r.store.versions[r.vault], func(one domain.TextVersion) bool {
		return one.Path == path && one.GetID() == id
	})
	return nil
}

func (r memoryRepo) RemovePluginVersions(_ context.Context, pluginID string) ([]domain.TextVersion, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	var removed []domain.TextVersion
	r.store.versions[r.vault] = slices.DeleteFunc(r.store.versions[r.vault], func(one domain.TextVersion) bool {
		if one.PluginID == pluginID {
			removed = append(removed, one)
			return true
		}
		return false
	})
	return removed, nil
}

// memoryFiles is the plugin folders of every vault, held in memory, by
// "<plugin>/<name>".
type memoryFiles struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (m *memoryFiles) Open(domain.Vault) (textport.TextFiles, error) { return m, nil }

func (m *memoryFiles) ReadTextFile(_ context.Context, pluginID, name string) (textport.TextFile, error) {
	m.mu.Lock()
	raw, ok := m.files[pluginID+"/"+name]
	m.mu.Unlock()
	if !ok {
		return textport.TextFile{}, textport.ErrNoText
	}
	segments, err := textfile.Decode(raw)
	if err != nil {
		return textport.TextFile{}, err
	}
	return textport.TextFile{Segments: segments, Hash: domain.HashContent(raw)}, nil
}

func (m *memoryFiles) WriteTextFile(_ context.Context, pluginID, name string, segments []domain.TextSegment) (string, error) {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return "", err
	}
	m.write(pluginID+"/"+name, raw)
	return domain.HashContent(raw), nil
}

func (m *memoryFiles) write(key string, raw []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[key] = raw
}

func (m *memoryFiles) has(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.files[key]
	return ok
}

// plugins is the text plugins installed, each switched on or off.
type plugins struct {
	mu       sync.Mutex
	all      []textport.TextPlugin
	disabled map[string]bool
}

func (p *plugins) GetTextPlugins() []textport.TextPlugin {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []textport.TextPlugin
	for _, one := range p.all {
		if !p.disabled[one.ID] {
			out = append(out, one)
		}
	}
	return out
}

func (p *plugins) setEnabled(id string, isEnabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.disabled == nil {
		p.disabled = map[string]bool{}
	}
	p.disabled[id] = !isEnabled
}

// listener is every change of versions reported.
type listener struct {
	mu      sync.Mutex
	changes []string
}

func (l *listener) ReportVersionsChanged(_ domain.Vault, path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.changes = append(l.changes, path)
}

func (l *listener) take() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.changes
	l.changes = nil
	return out
}

// providers returns for each plugin the provider it was given.
type providers map[string]textport.TextProvider

func (p providers) OpenProvider(_ context.Context, pluginID string) (textport.TextProvider, error) {
	provider, ok := p[pluginID]
	if !ok {
		return nil, fmt.Errorf("no provider for %s", pluginID)
	}
	return provider, nil
}

// fakeProvider, on CreateText, makes a text with make, writes it into its
// folder and puts its version through the use case, the way a plugin calls the host.
type fakeProvider struct {
	pluginID string
	kind     domain.TextKind
	versions *textversion.Versions
	vault    domain.Vault
	folders  *memoryFiles
	// make is the text made of the file, or of the source text where hasSource is set.
	make func(path string, source domain.VersionText, hasSource bool) []domain.TextSegment
	// files is the content of each vault file, as the plugin reads it.
	files map[string][]byte

	mu      sync.Mutex
	created int
}

// CreateText makes the text of the file, as a plugin does on a host request.
func (f *fakeProvider) CreateText(ctx context.Context, req textport.TextCreation) error {
	return f.create(ctx, req.Path, "")
}

// create makes the text of the file, or of the version where source names one, as a plugin does
// when a command runs.
func (f *fakeProvider) create(ctx context.Context, path string, source domain.TextVersionID) error {
	params := textversion.VersionPut{Path: path, Source: source, Kind: f.kind}
	var segments []domain.TextSegment
	if source == "" {
		params.SourceHash = domain.HashContent(f.files[path])
		params.Key = "file-" + params.SourceHash[:8]
		segments = f.make(path, domain.VersionText{}, false)
	} else {
		text, err := f.versions.GetVersionText(ctx, f.vault, path, source)
		if err != nil {
			return err
		}
		params.SourceHash = text.Version().Hash
		params.Key = "from-" + strings.ReplaceAll(string(source), "/", "-")
		segments = f.make(path, text, true)
	}
	params.File = params.Key + ".jsonl"
	if _, err := f.folders.WriteTextFile(ctx, f.pluginID, params.File, segments); err != nil {
		return err
	}
	f.mu.Lock()
	f.created++
	f.mu.Unlock()
	_, err := f.versions.PutVersion(ctx, f.vault, f.pluginID, params)
	return err
}

func (f *fakeProvider) getCreated() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created
}

// steppingClock is an instant a minute later at every reading.
func steppingClock() *minuteClock {
	return &minuteClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
}

type minuteClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *minuteClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(time.Minute)
	return c.now
}
