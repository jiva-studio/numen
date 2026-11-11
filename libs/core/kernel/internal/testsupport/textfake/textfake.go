// Package textfake holds the outer ports of the text versions in memory: the records, the plugin
// folders, the files of a vault and the clock, for tests that build the real use cases over them.
package textfake

import (
	"bytes"
	"context"
	"io"
	"github.com/jiva-studio/numen/libs/note/edit"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/textfile"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Versions is the records of text versions of every vault, held in memory.
type Versions struct {
	mu       sync.Mutex
	versions []domain.TextVersion
}

func (m *Versions) OpenVersionReader(domain.Vault) (textport.VersionReader, error) {
	return repo{store: m}, nil
}

func (m *Versions) OpenVersionWriter(domain.Vault) (textport.VersionWriter, error) {
	return repo{store: m}, nil
}

func (m *Versions) OpenContentHashes(domain.Vault) (textport.ContentHashes, error) {
	return repo{store: m}, nil
}

type repo struct{ store *Versions }

func (r repo) GetVersions(_ context.Context, path string) ([]domain.TextVersion, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	var out []domain.TextVersion
	for _, one := range r.store.versions {
		if one.Path == path {
			out = append(out, one)
		}
	}
	return out, nil
}

func (r repo) SaveVersion(_ context.Context, version domain.TextVersion) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for i, one := range r.store.versions {
		if one.Path == version.Path && one.GetID() == version.GetID() {
			r.store.versions[i] = version
			return nil
		}
	}
	r.store.versions = append(r.store.versions, version)
	return nil
}

func (r repo) RemoveVersion(_ context.Context, path string, id domain.TextVersionID) error {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	r.store.versions = slices.DeleteFunc(r.store.versions, func(one domain.TextVersion) bool {
		return one.Path == path && one.GetID() == id
	})
	return nil
}

func (r repo) RemovePluginVersions(_ context.Context, pluginID string) ([]domain.TextVersion, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	var removed []domain.TextVersion
	r.store.versions = slices.DeleteFunc(r.store.versions, func(one domain.TextVersion) bool {
		if one.PluginID == pluginID {
			removed = append(removed, one)
			return true
		}
		return false
	})
	return removed, nil
}

func (repo) FindContentHash(context.Context, string) (textdomain.ContentHash, bool, error) {
	return textdomain.ContentHash{}, false, nil
}

func (repo) SaveContentHash(context.Context, textdomain.ContentHash) error { return nil }

// Folders is the folders of the plugins, held in memory by "<plugin>/<name>".
type Folders struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (m *Folders) Open(domain.Vault) (textport.TextFiles, error) { return m, nil }

func (m *Folders) ReadTextFile(_ context.Context, pluginID, name string) (textdomain.File, error) {
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

func (m *Folders) WriteTextFile(_ context.Context, pluginID, name string, segments []domain.TextSegment) (string, error) {
	raw, err := textfile.Encode(segments)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	m.files[pluginID+"/"+name] = raw
	return domain.HashContent(raw), nil
}

func (m *Folders) DeleteTextFile(_ context.Context, pluginID, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, pluginID+"/"+name)
	return nil
}

// Plugins is the plugins switched on.
type Plugins []string

func (p Plugins) IsEnabled(pluginID string) bool { return slices.Contains(p, pluginID) }

// Readers opens a vault whose files are the map.
type Readers map[string][]byte

func (r Readers) Open(domain.Vault) (vaultport.VaultReader, error) { return reader{files: r}, nil }

type reader struct {
	vaultport.VaultReader
	files Readers
}

func (r reader) Read(_ context.Context, path string) ([]byte, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, edit.ErrNotExist
	}
	return raw, nil
}

func (r reader) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, edit.ErrNotExist
	}
	return memoryFile{bytes.NewReader(raw)}, nil
}

func (r reader) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	raw, ok := r.files[path]
	if !ok {
		return domain.FileStat{}, edit.ErrNotExist
	}
	return domain.FileStat{
		Path: path, Kind: domain.KindFile, Size: int64(len(raw)), ModTime: time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC),
	}, nil
}

// memoryFile is a file stored in memory, with nothing to release.
type memoryFile struct{ *bytes.Reader }

func (memoryFile) Close() error { return nil }

// SteppingClock is a clock that moves a minute at every reading.
type SteppingClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewSteppingClock() *SteppingClock {
	return &SteppingClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
}

func (c *SteppingClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(time.Minute)
	return c.now
}
