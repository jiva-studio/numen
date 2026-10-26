package derivedfile_test

import (
	"errors"
	"io"
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

func TestAPluginFileIsRenamedWholeAndOpenedAtAnyPart(t *testing.T) {
	store, folder := openPluginStore(t)
	ctx := t.Context()

	if err := store.Append(ctx, "clip.part", []byte("the bytes of a copy")); err != nil {
		t.Fatal(err)
	}
	if err := store.Write(ctx, "clip.mp4", []byte("a copy made before")); err != nil {
		t.Fatal(err)
	}
	if err := store.Rename(ctx, "clip.part", "clip.mp4"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(folder, "clip.part")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the old name is still there (%v)", err)
	}
	body, size, err := store.Open(ctx, "clip.mp4")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if _, err := body.Seek(4, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	rest, err := io.ReadAll(body)
	if err != nil || size != 19 || string(rest) != "bytes of a copy" {
		t.Errorf("the renamed file holds %d bytes and %q from its fifth (%v)", size, rest, err)
	}
	for _, name := range []string{"../escape.mp4", ".hidden"} {
		if err := store.Rename(ctx, "clip.mp4", name); !errors.Is(err, vaultconfig.ErrOutside) {
			t.Errorf("a rename to %q was answered %v", name, err)
		}
	}
	if _, _, err := store.Open(ctx, "none.mp4"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there opened as %v", err)
	}
}
