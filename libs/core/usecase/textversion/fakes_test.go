package textversion_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

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

func (r memoryRepo) ListVersions(context.Context) ([]domain.TextVersion, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	return slices.Clone(r.store.versions[r.vault]), nil
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

// providers answers for each plugin with the provider it was given.
type providers map[string]textport.TextProvider

func (p providers) OpenProvider(_ context.Context, pluginID string) (textport.TextProvider, error) {
	provider, ok := p[pluginID]
	if !ok {
		return nil, fmt.Errorf("no provider for %s", pluginID)
	}
	return provider, nil
}

// fakeProvider keeps its texts in memory, and on CreateText makes one with
// make and puts its version through the use case, the way a plugin calls the host.
type fakeProvider struct {
	pluginID string
	format   domain.TextFormat
	versions *textversion.Versions
	vault    domain.Vault
	// make is the text made of the file, or of the input text where input is set.
	make func(path string, input domain.VersionText, hasInput bool) []domain.TextPart
	// files is the content of each file, as the plugin reads it.
	files map[string][]byte

	mu      sync.Mutex
	texts   map[string][]domain.TextPart
	created int
	deleted []string
	// pageSize is how many parts one page holds. Zero is all of them.
	pageSize int
}

func (f *fakeProvider) GetText(_ context.Context, req textport.TextRequest) (textport.TextPage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	parts, ok := f.texts[req.Path+"|"+req.Key]
	if !ok {
		return textport.TextPage{}, textport.ErrNoText
	}
	if f.pageSize == 0 {
		return textport.TextPage{Parts: parts}, nil
	}
	from := 0
	if req.Cursor != "" {
		if _, err := fmt.Sscan(req.Cursor, &from); err != nil {
			return textport.TextPage{}, err
		}
	}
	to := min(from+f.pageSize, len(parts))
	page := textport.TextPage{Parts: parts[from:to]}
	if to < len(parts) {
		page.NextCursor = fmt.Sprint(to)
	}
	return page, nil
}

func (f *fakeProvider) CreateText(ctx context.Context, req textport.TextCreation) error {
	params := textversion.VersionPut{Path: req.Path, Input: req.Input, Format: f.format}
	var parts []domain.TextPart
	if req.Input == "" {
		params.InputHash = domain.HashContent(f.files[req.Path])
		params.Key = "file-" + params.InputHash[:8]
		parts = f.make(req.Path, domain.VersionText{}, false)
	} else {
		input, err := f.versions.GetVersionText(ctx, f.vault, req.Path, req.Input)
		if err != nil {
			return err
		}
		params.InputHash = input.Version.Hash
		params.Key = "from-" + strings.ReplaceAll(string(req.Input), "/", "-")
		parts = f.make(req.Path, input, true)
	}
	params.Hash = domain.HashContent([]byte(domain.NewVersionText(domain.TextVersion{}, parts).Text))
	f.mu.Lock()
	if f.texts == nil {
		f.texts = map[string][]domain.TextPart{}
	}
	f.texts[req.Path+"|"+params.Key] = parts
	f.created++
	f.mu.Unlock()
	_, err := f.versions.PutVersion(ctx, f.vault, f.pluginID, params)
	return err
}

func (f *fakeProvider) DeleteText(_ context.Context, _ domain.VaultID, path, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.texts, path+"|"+key)
	f.deleted = append(f.deleted, key)
	return nil
}

func (f *fakeProvider) getCreated() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.created
}

// steppingClock is an instant a minute later at every reading.
func steppingClock() func() time.Time {
	var mu sync.Mutex
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(time.Minute)
		return now
	}
}
