package plugin

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// diskVault is the vault of a folder on this machine, worked as the host works it for a plugin.
func diskVault(t *testing.T, files map[string]string) (Vault, domain.Vault) {
	t.Helper()
	v := testsupport.NewVault(t, files)
	return Vault{
		Current: func() (domain.Vault, bool) { return v, true },
		Readers: filesystem.VaultReaders{},
		Writers: filesystem.VaultWriters{},
		Data:    filesystem.PluginData{},
		Cache:   appstate.PluginCachesAt(t.TempDir()),
	}, v
}

func TestAFileIsReadWithTheFingerprintItHadBeforeItsBytesWereRead(t *testing.T) {
	access, _ := diskVault(t, map[string]string{"decks/A.md": "body"})

	content, pos, err := access.Read(t.Context(), "decks/A.md")
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "body" || pos.Path != "decks/A.md" || pos.Size != 4 {
		t.Errorf("read %q with %+v", content, pos)
	}
	if _, _, err := access.Read(t.Context(), "gone.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
	}
}

func TestAFileIsSavedOverWhatWasReadAndRefusedOverWhatMoved(t *testing.T) {
	access, v := diskVault(t, map[string]string{"decks/A.md": "body"})
	_, pos, err := access.Read(t.Context(), "decks/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Path, "decks", "A.md"), []byte("moved under it"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = access.Write(t.Context(), "decks/A.md", []byte("mine"), pos)

	if !errors.Is(err, vaultport.ErrStale) {
		t.Errorf("a write over a file that moved was %v", err)
	}
	_, now, err := access.Read(t.Context(), "decks/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := access.Write(t.Context(), "decks/A.md", []byte("mine"), now); err != nil {
		t.Errorf("a write over what was read was %v", err)
	}
}

func TestAFileIsMadeOnlyWhereNoneExistsAndTheWritersAreHeldAndLetGo(t *testing.T) {
	access, _ := diskVault(t, map[string]string{"decks/A.md": "body"})

	if err := access.Create(t.Context(), "decks/B.md", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := access.Create(t.Context(), "decks/B.md", []byte("b")); !errors.Is(err, vaultport.ErrOccupied) {
		t.Errorf("a file made over one was %v", err)
	}
	release, err := access.Lock(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestTheDataOfAPluginIsCachedInItsFolderAndNoOther(t *testing.T) {
	access, _ := diskVault(t, nil)
	store, err := access.OpenData("flashcards")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Append(t.Context(), "run.log", []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(t.Context(), "run.log", []byte("two\n")); err != nil {
		t.Fatal(err)
	}
	content, err := store.Read(t.Context(), "run.log")
	items, listErr := store.List(t.Context())

	if err != nil || string(content) != "one\ntwo\n" {
		t.Errorf("read %q (%v)", content, err)
	}
	if listErr != nil || !slices.Equal(items, []pluginport.PluginDataFile{{Name: "run.log", Size: 8}}) {
		t.Errorf("listed %v (%v)", items, listErr)
	}
	for _, name := range []string{"", "../run.log", "a/b", ".hidden"} {
		if _, err := store.Read(t.Context(), name); !errors.Is(err, filesystem.ErrOutside) {
			t.Errorf("%q was read with %v", name, err)
		}
	}
	for _, folder := range []string{"", "a/b", "..", "a.b"} {
		if _, err := access.OpenData(folder); err == nil {
			t.Errorf("the folder %q was opened", folder)
		}
	}
	if _, err := store.Read(t.Context(), "other.log"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
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
	_, writeErr := access.Write(t.Context(), "a.md", nil, domain.Fingerprint{})
	_, lockErr := access.Lock(t.Context())
	_, notesErr := access.GetNotes(t.Context(), nil)
	_, openErr := access.OpenData("flashcards")
	_, _, watchErr := access.Watch(t.Context())

	for name, err := range map[string]error{"read": readErr, "write": writeErr, "lock": lockErr, "notes": notesErr, "data": openErr, "watch": watchErr} {
		if !errors.Is(err, ErrNoVault) {
			t.Errorf("%s was %v", name, err)
		}
	}
	if err := access.Scan(t.Context(), []string{"a.md"}); !errors.Is(err, ErrNoVault) {
		t.Errorf("a scan was %v", err)
	}
}
