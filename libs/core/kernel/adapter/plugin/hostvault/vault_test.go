package hostvault

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// diskHooks are what a test sets of the vault: the notice of a written file, the indexing of paths, the
// lookup of a vault by identity and the use case that adds links.
type diskHooks struct {
	written  func()
	index    func(ctx context.Context, v domain.Vault, paths []string) error
	find     func(id domain.VaultID) (domain.Vault, error)
	addLinks note.AddLinks
}

func diskVault(t *testing.T, files map[string]string, hooks diskHooks) (*vault.Access, domain.Vault) {
	t.Helper()
	v := vaultfixture.NewVault(t, files)
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	return NewAccess(
		Session{Current: func() (domain.Vault, bool) { return v, true }, Written: hooks.written, Find: hooks.find},
		Files{
			Stat:        file.NewStatFile(vaultreader.Readers{}),
			ReadFile:    file.NewReadFile(vaultreader.Readers{}),
			RangeReader: file.Read{Readers: vaultreader.Readers{}},
			OpenWriter:  func(v domain.Vault) (NoteWriter, error) { return writers.Open(v) },
			Index:       hooks.index,
		},
		Lookups{}, Connections{}, Arranger{Notes: NoteChanges{AddLinks: hooks.addLinks}},
		Stores{
			StoreReaders: func(v domain.Vault, pluginID string) (vault.StoreReader, error) {
				return derivedfile.PluginStores{}.Open(v, pluginID)
			},
			StoreWriters: func(v domain.Vault, pluginID string) (vault.StoreWriter, error) {
				return derivedfile.PluginStores{}.OpenWriter(v, pluginID)
			},
			Cache: appstate.PluginCachesAt(t.TempDir()),
		},
	), v
}

func TestAFileIsReadWithTheFingerprintItHadBeforeItsBytesWereRead(t *testing.T) {
	access, _ := diskVault(t, map[string]string{"samples/A.md": "body"}, diskHooks{})

	content, pos, err := access.Files.Read(t.Context(), "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "body" || pos.Path != "samples/A.md" || pos.Size != 4 {
		t.Errorf("read %q with %+v", content, pos)
	}
	if _, _, err := access.Files.Read(t.Context(), "gone.md"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
	}
}

func TestAFileIsSavedOverWhatWasReadAndRefusedOverWhatMoved(t *testing.T) {
	var indexed []string
	var writeCount int
	var v domain.Vault
	access, v := diskVault(t, map[string]string{"samples/A.md": "body"}, diskHooks{
		written: func() { writeCount++ },
		index: func(_ context.Context, got domain.Vault, paths []string) error {
			if got.ID == v.ID {
				indexed = append(indexed, paths...)
			}
			return nil
		},
	})
	_, pos, err := access.Files.Read(t.Context(), "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Path, "samples", "A.md"), []byte("moved under it"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = access.Writer.WriteFile(t.Context(), "samples/A.md", []byte("mine"), pos)

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("a write over a file that moved was %v", err)
	}
	if writeCount != 0 || len(indexed) != 0 {
		t.Errorf("a refused write notified %d times and indexed %v", writeCount, indexed)
	}
	_, now, err := access.Files.Read(t.Context(), "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := access.Writer.WriteFile(t.Context(), "samples/A.md", []byte("mine"), now); err != nil {
		t.Errorf("a write over what was read was %v", err)
	}
	if writeCount != 1 || !slices.Equal(indexed, []string{"samples/A.md"}) {
		t.Errorf("a write notified %d times and indexed %v", writeCount, indexed)
	}
}

func TestVaultAccess_CreateFileAndLockWriters(t *testing.T) {
	var indexed []string
	var v domain.Vault
	access, v := diskVault(t, map[string]string{"samples/A.md": "body"}, diskHooks{
		index: func(_ context.Context, got domain.Vault, paths []string) error {
			if got.ID == v.ID {
				indexed = append(indexed, paths...)
			}
			return nil
		},
	})

	if err := access.Writer.CreateFile(t.Context(), "samples/B.md", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(indexed, []string{"samples/B.md"}) {
		t.Errorf("a created file indexed %v", indexed)
	}
	if err := access.Writer.CreateFile(t.Context(), "samples/B.md", []byte("b")); !errors.Is(err, vaultdomain.ErrAlreadyExists) {
		t.Errorf("a file made over one was %v", err)
	}
	if !slices.Equal(indexed, []string{"samples/B.md"}) {
		t.Errorf("a refused create indexed %v", indexed)
	}
}

func TestAPluginsFolderIsTheFolderOfItsIdentityInTheVaultsStore(t *testing.T) {
	access, v := diskVault(t, nil, diskHooks{})
	store, err := access.Stores.OpenStoreWriter("md.numen.a")
	if err != nil {
		t.Fatal(err)
	}
	reader, err := access.Stores.OpenStore("md.numen.a")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Append(t.Context(), "one.log", []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	on, err := os.ReadFile(filepath.Join(v.Path, ".numen", "md.numen.a", "one.log"))
	items, listErr := reader.List(t.Context())

	if err != nil || string(on) != "one\n" {
		t.Errorf("the folder holds %q (%v)", on, err)
	}
	if listErr != nil || !slices.Equal(items, []plugin.StoreFile{{Name: "one.log", Size: 4}}) {
		t.Errorf("listed %v (%v)", items, listErr)
	}
	for _, id := range []string{"", "samples", "index", "..", "a/b.c", ".a.b"} {
		if _, err := access.Stores.OpenStore(id); !errors.Is(err, vaultdomain.ErrOutside) {
			t.Errorf("the folder of %q was opened with %v", id, err)
		}
	}
}

func TestVaultAccess_CacheKeyedByPluginAndNameRejectsEscape(t *testing.T) {
	access, _ := diskVault(t, nil, diskHooks{})

	if _, found, err := access.Cache.ReadCache(t.Context(), "md.numen.a", "schedules"); err != nil || found {
		t.Fatalf("an empty cache answered %v, %v", found, err)
	}
	if err := access.Cache.WriteCache(t.Context(), "md.numen.a", "schedules", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	mine, found, err := access.Cache.ReadCache(t.Context(), "md.numen.a", "schedules")
	other, otherFound, otherErr := access.Cache.ReadCache(t.Context(), "md.numen.b", "schedules")

	if err != nil || !found || string(mine) != "{}" {
		t.Errorf("the cache held %q, %v (%v)", mine, found, err)
	}
	if otherErr != nil || otherFound || other != nil {
		t.Errorf("another plugin's cache held %q, %v (%v)", other, otherFound, otherErr)
	}
	for _, name := range []string{"", "../x", "a/b", ".days"} {
		if err := access.Cache.WriteCache(t.Context(), "md.numen.a", name, nil); err == nil {
			t.Errorf("a cache was written under %q", name)
		}
	}
}

func TestVault_NoCurrentVaultReturnsErrNoVault(t *testing.T) {
	access := NewAccess(
		Session{Current: func() (domain.Vault, bool) { return domain.Vault{}, false }},
		Files{}, Lookups{}, Connections{}, Arranger{}, Stores{},
	)

	_, _, readErr := access.Files.Read(t.Context(), "a.md")
	_, writeErr := access.Writer.WriteFile(t.Context(), "a.md", nil, domain.FileStat{})
	_, notesErr := access.Notes.GetNotes(t.Context(), nil)
	_, openErr := access.Stores.OpenStore("md.numen.a")
	_, _, watchErr := access.Watch.Watch(t.Context())

	for name, err := range map[string]error{"read": readErr, "write": writeErr, "notes": notesErr, "store": openErr, "watch": watchErr} {
		if !errors.Is(err, plugin.ErrNoVault) {
			t.Errorf("%s was %v", name, err)
		}
	}
	if err := access.Writer.Scan(t.Context(), []string{"a.md"}); !errors.Is(err, plugin.ErrNoVault) {
		t.Errorf("a scan was %v", err)
	}
}

func TestAddLinksCancelledAfterTheFirstNoteSendsOneFileWriteNotification(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	writeCount := 0
	cancelAfterIndex := func(context.Context, domain.Vault, []string) error {
		cancel()
		return nil
	}
	access, _ := diskVault(t, map[string]string{
		"Danube.md": "# Danube\n", "Rhine.md": "# Rhine\n", "Volga.md": "# Volga\n",
	}, diskHooks{
		written: func() { writeCount++ },
		addLinks: note.NewAddLinks(note.NewEditLinks(
			vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, cancelAfterIndex, testsupport.NewClock())),
	})
	estuary := domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Estuary"}, Role: domain.RoleRef}

	added, err := access.LinkEditor.AddLinks(ctx, []plugin.LinkAddition{
		{From: "Danube.md", Link: estuary}, {From: "Rhine.md", Link: estuary}, {From: "Volga.md", Link: estuary},
	})
	if err != nil {
		t.Fatalf("AddLinks returned %v, want the results and no error", err)
	}
	if len(added) != 3 || added[0].Err != nil || !errors.Is(added[1].Err, plugin.ErrNotAttempted) || !errors.Is(added[2].Err, plugin.ErrNotAttempted) {
		t.Fatalf("AddLinks returned %+v, want Danube.md written and the rest not attempted", added)
	}
	if writeCount != 1 {
		t.Errorf("the file-written notification fired %d times, want 1", writeCount)
	}
}

func TestAddLinksCancelledBeforeAnyWriteDoesNotNotify(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	writeCount := 0
	access, _ := diskVault(t, map[string]string{"Danube.md": "# Danube\n"}, diskHooks{
		written: func() { writeCount++ },
		addLinks: note.NewAddLinks(note.NewEditLinks(
			vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, nil, testsupport.NewClock())),
	})
	estuary := domain.Link{Target: domain.Address{Scheme: domain.SchemeName, Value: "Estuary"}, Role: domain.RoleRef}

	added, err := access.LinkEditor.AddLinks(ctx, []plugin.LinkAddition{{From: "Danube.md", Link: estuary}})
	if err != nil {
		t.Fatalf("AddLinks returned %v, want the results and no error", err)
	}
	if len(added) != 1 || !errors.Is(added[0].Err, plugin.ErrNotAttempted) {
		t.Fatalf("AddLinks returned %+v, want Danube.md not attempted", added)
	}
	if writeCount != 0 {
		t.Errorf("the file-written notification fired %d times, want 0", writeCount)
	}
}

func TestAVaultNamedByIdentityIsWorkedAsTheVaultShown(t *testing.T) {
	other := vaultfixture.NewVault(t, map[string]string{"Baikal.md": "# Baikal\n"})
	access, current := diskVault(t, map[string]string{"samples/A.md": "body"}, diskHooks{
		find: func(id domain.VaultID) (domain.Vault, error) {
			if id == other.ID {
				return other, nil
			}
			return domain.Vault{}, fs.ErrNotExist
		},
	})

	linked, err := access.Switch.With(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, isShowing := linked.Switch.GetVault()
	_, _, readErr := linked.Files.Read(t.Context(), "Baikal.md")
	_, _, notThere := linked.Files.Read(t.Context(), "samples/A.md")
	_, unknown := access.Switch.With("nobody")

	if !isShowing || got.ID != other.ID || readErr != nil || !errors.Is(notThere, fs.ErrNotExist) {
		t.Errorf("the linked vault was %+v (%v), read %v and %v", got, isShowing, readErr, notThere)
	}
	if !errors.Is(unknown, fs.ErrNotExist) {
		t.Errorf("a vault nobody holds was %v", unknown)
	}
	if now, _ := access.Switch.GetVault(); now.ID != current.ID {
		t.Errorf("the vault current changed to %v", now.ID)
	}
}
