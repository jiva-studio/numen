package vaultaccess

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// diskVault is the vault of a folder on this machine, worked as the host works it for a plugin.
func diskVault(t *testing.T, files map[string]string) (Vault, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, files)
	return Vault{
		Current: func() (domain.Vault, bool) { return v, true },
		Readers: vaultreader.Readers{},
		Writers: vaultwriter.VaultWriters{},
		Stores:  derivedfile.PluginStores{},
		Cache:   appstate.PluginCachesAt(t.TempDir()),
	}, v
}

func TestAFileIsReadWithTheFingerprintItHadBeforeItsBytesWereRead(t *testing.T) {
	access, _ := diskVault(t, map[string]string{"samples/A.md": "body"})

	content, pos, err := access.Read(t.Context(), "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "body" || pos.Path != "samples/A.md" || pos.Size != 4 {
		t.Errorf("read %q with %+v", content, pos)
	}
	if _, _, err := access.Read(t.Context(), "gone.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
	}
}

func TestAFileIsSavedOverWhatWasReadAndRefusedOverWhatMoved(t *testing.T) {
	access, v := diskVault(t, map[string]string{"samples/A.md": "body"})
	var indexed []string
	var writeCount int
	access.Written = func() { writeCount++ }
	access.Index = func(_ context.Context, got domain.Vault, paths []string) error {
		if got.ID == v.ID {
			indexed = append(indexed, paths...)
		}
		return nil
	}
	_, pos, err := access.Read(t.Context(), "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Path, "samples", "A.md"), []byte("moved under it"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = access.Write(t.Context(), "samples/A.md", []byte("mine"), pos)

	if !errors.Is(err, vaultport.ErrStale) {
		t.Errorf("a write over a file that moved was %v", err)
	}
	if writeCount != 0 || len(indexed) != 0 {
		t.Errorf("a refused write notified %d times and indexed %v", writeCount, indexed)
	}
	_, now, err := access.Read(t.Context(), "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := access.Write(t.Context(), "samples/A.md", []byte("mine"), now); err != nil {
		t.Errorf("a write over what was read was %v", err)
	}
	if writeCount != 1 || !slices.Equal(indexed, []string{"samples/A.md"}) {
		t.Errorf("a write notified %d times and indexed %v", writeCount, indexed)
	}
}

func TestAFileIsMadeOnlyWhereNoneExistsAndTheWritersAreHeldAndLetGo(t *testing.T) {
	access, v := diskVault(t, map[string]string{"samples/A.md": "body"})
	var indexed []string
	access.Index = func(_ context.Context, got domain.Vault, paths []string) error {
		if got.ID == v.ID {
			indexed = append(indexed, paths...)
		}
		return nil
	}

	if err := access.Create(t.Context(), "samples/B.md", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(indexed, []string{"samples/B.md"}) {
		t.Errorf("a created file indexed %v", indexed)
	}
	if err := access.Create(t.Context(), "samples/B.md", []byte("b")); !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Errorf("a file made over one was %v", err)
	}
	if !slices.Equal(indexed, []string{"samples/B.md"}) {
		t.Errorf("a refused create indexed %v", indexed)
	}
	release, err := access.Lock(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestAPluginsFolderIsTheFolderOfItsIdentityInTheVaultsStore(t *testing.T) {
	access, v := diskVault(t, nil)
	store, err := access.OpenStore("md.numen.a")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Append(t.Context(), "one.log", []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	on, err := os.ReadFile(filepath.Join(v.Path, ".numen", "md.numen.a", "one.log"))
	items, listErr := store.List(t.Context())

	if err != nil || string(on) != "one\n" {
		t.Errorf("the folder holds %q (%v)", on, err)
	}
	if listErr != nil || !slices.Equal(items, []pluginport.PluginStoreFile{{Name: "one.log", Size: 4}}) {
		t.Errorf("listed %v (%v)", items, listErr)
	}
	for _, id := range []string{"", "samples", "index", "..", "a/b.c", ".a.b"} {
		if _, err := access.OpenStore(id); !errors.Is(err, vaultconfig.ErrOutside) {
			t.Errorf("the folder of %q was opened with %v", id, err)
		}
	}
}

func TestACacheIsCachedPerPluginAndNameAndRefusesNamesThatEscape(t *testing.T) {
	access, _ := diskVault(t, nil)

	if _, found, err := access.ReadCache(t.Context(), "md.numen.a", "schedules"); err != nil || found {
		t.Fatalf("an empty cache answered %v, %v", found, err)
	}
	if err := access.WriteCache(t.Context(), "md.numen.a", "schedules", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	mine, found, err := access.ReadCache(t.Context(), "md.numen.a", "schedules")
	other, otherFound, otherErr := access.ReadCache(t.Context(), "md.numen.b", "schedules")

	if err != nil || !found || string(mine) != "{}" {
		t.Errorf("the cache held %q, %v (%v)", mine, found, err)
	}
	if otherErr != nil || otherFound || other != nil {
		t.Errorf("another plugin's cache held %q, %v (%v)", other, otherFound, otherErr)
	}
	for _, name := range []string{"", "../x", "a/b", ".days"} {
		if err := access.WriteCache(t.Context(), "md.numen.a", name, nil); err == nil {
			t.Errorf("a cache was written under %q", name)
		}
	}
}

func TestAHostShowingNoVaultAnswersEveryCallThatWorksOneWithErrNoVault(t *testing.T) {
	access := Vault{Current: func() (domain.Vault, bool) { return domain.Vault{}, false }}

	_, _, readErr := access.Read(t.Context(), "a.md")
	_, writeErr := access.Write(t.Context(), "a.md", nil, domain.FileStat{})
	_, lockErr := access.Lock(t.Context())
	_, notesErr := access.GetNotes(t.Context(), nil)
	_, openErr := access.OpenStore("md.numen.a")
	_, _, watchErr := access.Watch(t.Context())

	for name, err := range map[string]error{"read": readErr, "write": writeErr, "lock": lockErr, "notes": notesErr, "store": openErr, "watch": watchErr} {
		if !errors.Is(err, ErrNoVault) {
			t.Errorf("%s was %v", name, err)
		}
	}
	if err := access.Scan(t.Context(), []string{"a.md"}); !errors.Is(err, ErrNoVault) {
		t.Errorf("a scan was %v", err)
	}
}
