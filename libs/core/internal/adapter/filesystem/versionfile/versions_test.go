package versionfile_test

import (
	"context"
	"io/fs"
	"maps"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/versionfile"
)

type memoryFolder struct {
	mu    sync.Mutex
	files map[string][]byte
}

func (m *memoryFolder) Read(_ context.Context, name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

func (m *memoryFolder) Write(_ context.Context, name string, content []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[name] = content
	return nil
}

func newRepositories() (*versionfile.Repositories, map[domain.VaultID]*memoryFolder) {
	folders := map[domain.VaultID]*memoryFolder{}
	var mu sync.Mutex
	return versionfile.NewRepositories(func(v domain.Vault) (versionfile.Folder, error) {
		mu.Lock()
		defer mu.Unlock()
		if folders[v.ID] == nil {
			folders[v.ID] = &memoryFolder{files: map[string][]byte{}}
		}
		return folders[v.ID], nil
	}), folders
}

func TestVersionsSurviveTheFileAndStayInTheirVault(t *testing.T) {
	ctx := t.Context()
	repos, folders := newRepositories()
	one, err := repos.Open(domain.Vault{ID: "one"})
	if err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 5, 1, 10, 0, 0, 5, time.UTC)
	a := domain.TextVersion{Path: "a.pdf", PluginID: "p", Key: "k", Format: domain.TextFormatPositioned, InputHash: "f", Hash: "h", CreateTime: created}
	b := domain.TextVersion{Path: "a.pdf", PluginID: "q", Key: "k", Input: a.GetID(), Format: domain.TextFormatPlain, InputHash: "h", CreateTime: created.Add(time.Second)}
	c := domain.TextVersion{Path: "b.mp3", PluginID: "p", Key: "k", Format: domain.TextFormatTimed, CreateTime: created}
	for _, v := range []domain.TextVersion{a, b, c} {
		if err := one.SaveVersion(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	a.Hash = "h2"
	if err := one.SaveVersion(ctx, a); err != nil {
		t.Fatal(err)
	}

	reopened, err := repos.Open(domain.Vault{ID: "one"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.GetVersions(ctx, "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("got %+v, want the two versions of a.pdf in the order first saved", got)
	}
	if !strings.Contains(string(folders["one"].files[versionfile.FileName]), `"input": "p/k"`) {
		t.Fatalf("the file does not name the input: %s", folders["one"].files[versionfile.FileName])
	}

	other, err := repos.Open(domain.Vault{ID: "two"})
	if err != nil {
		t.Fatal(err)
	}
	if all, err := other.ListVersions(ctx); err != nil || len(all) != 0 {
		t.Fatalf("another vault holds %+v (%v)", all, err)
	}

	if err := reopened.RemoveVersion(ctx, "a.pdf", a.GetID()); err != nil {
		t.Fatal(err)
	}
	all, err := reopened.ListVersions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || all[0] != b || all[1] != c {
		t.Fatalf("after removing a: %+v", all)
	}
	if keys := maps.Keys(folders["one"].files); len(folders["one"].files) != 1 {
		t.Fatalf("the folder holds %v, want one file", keys)
	}
}

func TestAFileThatIsNotJSONIsAnError(t *testing.T) {
	repos, folders := newRepositories()
	repo, err := repos.Open(domain.Vault{ID: "one"})
	if err != nil {
		t.Fatal(err)
	}
	folders["one"].files[versionfile.FileName] = []byte("not json")
	if _, err := repo.ListVersions(t.Context()); err == nil {
		t.Fatal("a broken file was read as no versions")
	}
}
