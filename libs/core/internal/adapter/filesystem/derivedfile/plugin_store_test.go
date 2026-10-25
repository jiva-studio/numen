package derivedfile_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

const pluginID = "md.numen.a"

func openPluginStore(t *testing.T) (pluginport.PluginStore, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	store, err := derivedfile.PluginStores{}.Open(domain.Vault{ID: "v", Path: root}, pluginID)
	if err != nil {
		t.Fatal(err)
	}
	return store, filepath.Join(root, vaultconfig.DefaultServiceDir, pluginID)
}

func TestPluginStoreWritesToFolderNamedByPluginID(t *testing.T) {
	store, folder := openPluginStore(t)
	ctx := t.Context()

	if err := store.Append(ctx, "log.jsonl", []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, "log.jsonl", []byte("two\n")); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, "a.jsonl", []byte("a")); err != nil {
		t.Fatal(err)
	}

	on, err := os.ReadFile(filepath.Join(folder, "log.jsonl"))
	if err != nil || string(on) != "one\ntwo\n" {
		t.Errorf("the folder holds %q (%v)", on, err)
	}
	read, err := store.Read(ctx, "log.jsonl")
	if err != nil || string(read) != "one\ntwo\n" {
		t.Errorf("read %q (%v)", read, err)
	}
	listed, err := store.List(ctx)
	if err != nil || !slices.Equal(listed, []pluginport.PluginStoreFile{{Name: "a.jsonl", Size: 1}, {Name: "log.jsonl", Size: 8}}) {
		t.Errorf("listed %v (%v)", listed, err)
	}
	if _, err := store.Read(ctx, "none"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
	}
}

func TestAnEmptyFolderListsNothing(t *testing.T) {
	store, _ := openPluginStore(t)

	listed, err := store.List(t.Context())

	if err != nil || len(listed) != 0 {
		t.Errorf("listed %v (%v)", listed, err)
	}
}

func TestAPluginReachesNothingOutsideItsFolder(t *testing.T) {
	store, folder := openPluginStore(t)
	ctx := t.Context()
	vault := filepath.Dir(filepath.Dir(folder))
	if err := os.WriteFile(filepath.Join(vault, "note.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(vault, "note.md"), filepath.Join(folder, "out")); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"", "..", "../config.json", "../md.numen.b/a", "/etc/passwd", ".hidden", `a\b`} {
		if _, err := store.Read(ctx, name); !errors.Is(err, vaultconfig.ErrOutside) {
			t.Errorf("%q was read with %v", name, err)
		}
		if err := store.Append(ctx, name, []byte("x")); !errors.Is(err, vaultconfig.ErrOutside) {
			t.Errorf("%q was appended to with %v", name, err)
		}
	}
	if err := store.Append(ctx, "out", []byte("x")); !errors.Is(err, vaultconfig.ErrOutside) {
		t.Errorf("a link out of the folder was appended to with %v", err)
	}
	if on, _ := os.ReadFile(filepath.Join(vault, "note.md")); string(on) != "mine" {
		t.Errorf("the note now holds %q", on)
	}
}
