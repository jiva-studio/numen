package textversion_test

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/textfake"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/textfile"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

// memoryVersions is every vault's versions, held in memory.
type memoryVersions struct {
	mu       sync.Mutex
	versions map[domain.VaultID][]domain.TextVersion
	sources  map[domain.VaultID]map[string]textdomain.ContentHash
}

func (m *memoryVersions) OpenVersionReader(v domain.Vault) (textport.VersionReader, error) {
	return memoryRepo{store: m, vault: v.ID}, nil
}

func (m *memoryVersions) OpenVersionWriter(v domain.Vault) (textport.VersionWriter, error) {
	return memoryRepo{store: m, vault: v.ID}, nil
}

func (m *memoryVersions) OpenContentHashes(v domain.Vault) (textport.ContentHashes, error) {
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

func (m *memoryFiles) ReadTextFile(_ context.Context, pluginID, name string) (textdomain.File, error) {
	m.mu.Lock()
	raw, ok := m.files[pluginID+"/"+name]
	m.mu.Unlock()
	if !ok {
		return textdomain.File{}, textdomain.ErrNoText
	}
	segments, err := textfile.Decode(raw)
	if err != nil {
		return textdomain.File{}, err
	}
	return textdomain.File{Segments: segments, TextHash: domain.HashContent(raw)}, nil
}

func (m *memoryFiles) WriteTextFile(_ context.Context, pluginID, name string, segments []domain.TextSegment) (string, error) {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return "", err
	}
	m.write(pluginID+"/"+name, raw)
	return domain.HashContent(raw), nil
}

func (m *memoryFiles) DeleteTextFile(_ context.Context, pluginID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, pluginID+"/"+name)
	return nil
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

// plugins is the plugins installed, each switched on or off.
type plugins struct {
	mu       sync.Mutex
	all      []string
	disabled map[string]bool
}

func (p *plugins) IsEnabled(pluginID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Contains(p.all, pluginID) && !p.disabled[pluginID]
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

// fakeProvider, on create, makes a text with make, writes it into its
// folder and puts its version through the use case, the way a plugin calls the host.
type fakeProvider struct {
	pluginID string
	kind     domain.TextKind
	versions *textfake.TextVersions
	vault    domain.Vault
	folders  *memoryFiles
	make     func(path string, source domain.VersionText, hasSource bool) []domain.TextSegment
	files    map[string][]byte

	mu      sync.Mutex
	created int
}

func (f *fakeProvider) create(ctx context.Context, path string, source domain.TextVersionID) error {
	params := plugin.TextPut{Path: path, Source: source, Kind: f.kind}
	var segments []domain.TextSegment
	if source == "" {
		params.InputHash = domain.HashContent(f.files[path])
		params.Key = "file-" + params.InputHash[:8]
		segments = f.make(path, domain.VersionText{}, false)
	} else {
		text, err := f.versions.Texts.GetVersionText(ctx, f.vault, path, source)
		if err != nil {
			return err
		}
		params.InputHash = text.Version().TextHash
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

func (f *fakeProvider) getCreatedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created
}

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

// FindContentHash is the source row of the path, as the sources table has it.
func (r memoryRepo) FindContentHash(_ context.Context, path string) (textdomain.ContentHash, bool, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	row, ok := r.store.sources[r.vault][path]
	return row, ok, nil
}

// SaveContentHash writes the record over the row of its path where the row has
// the record's size and modification time or already has its hash. A path
// with no row is not saved.
func (r memoryRepo) SaveContentHash(_ context.Context, record textdomain.ContentHash) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	row, ok := r.store.sources[r.vault][record.Path]
	if !ok {
		return nil
	}
	isSameStat := row.Size == record.Size && row.ModTime.Equal(record.ModTime)
	if isSameStat || (row.Hash != "" && row.Hash == record.Hash) {
		r.store.sources[r.vault][record.Path] = record
	}
	return nil
}

func (m *memoryVersions) addScannedSource(vault domain.VaultID, path string, size int64, modTime time.Time) {
	m.addSource(vault, textdomain.ContentHash{Path: path, Size: size, ModTime: modTime})
}

func (m *memoryVersions) addSource(vault domain.VaultID, row textdomain.ContentHash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sources == nil {
		m.sources = map[domain.VaultID]map[string]textdomain.ContentHash{}
	}
	if m.sources[vault] == nil {
		m.sources[vault] = map[string]textdomain.ContentHash{}
	}
	m.sources[vault][row.Path] = row
}

func (m *memoryVersions) getSource(vault domain.VaultID, path string) (textdomain.ContentHash, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.sources[vault][path]
	return row, ok
}

func (r memoryRepo) GetVersionsUnder(_ context.Context, path string) ([]domain.TextVersion, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	var out []domain.TextVersion
	for _, one := range r.store.versions[r.vault] {
		if one.Path == path || strings.HasPrefix(one.Path, path+"/") {
			out = append(out, one)
		}
	}
	return out, nil
}
