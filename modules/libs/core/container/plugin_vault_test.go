package container_test

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func openPluginVault(t *testing.T, notes map[string]string) (domain.Vault, func() (domain.Vault, bool), container.EditorAssembly) {
	t.Helper()
	cfg := container.Config{IndexPath: indexfile.Path(t), SchedulesPath: t.TempDir(), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	v := testsupport.NewVault(t, notes)
	if err := registry.Save(v); err != nil {
		t.Fatal(err)
	}
	return v, func() (domain.Vault, bool) { return v, true }, container.NewEditorAssembly(cfg, registry, db)
}

func TestAPluginNoteIsInTheIndexOnceItsFileIsScanned(t *testing.T) {
	v, current, assembly := openPluginVault(t, note)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{})
	if err != nil {
		t.Fatal(err)
	}

	if err := shared.Create(t.Context(), "Made.md", []byte("---\ntitle: Made\n---\n\n# Made\n")); err != nil {
		t.Fatal(err)
	}
	if err := shared.Scan(t.Context(), []string{"Made.md"}); err != nil {
		t.Fatal(err)
	}

	got, err := shared.GetNotes(t.Context(), []string{"Made.md"})
	if err != nil || got["Made.md"].Title != "Made" {
		t.Errorf("got %+v (%v) in %s", got, err, v.Name)
	}
}

func TestAPluginKeepsItsDataAndItsCacheApart(t *testing.T) {
	_, current, assembly := openPluginVault(t, note)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{})
	if err != nil {
		t.Fatal(err)
	}
	data, err := shared.OpenData("flashcards")
	if err != nil {
		t.Fatal(err)
	}

	if err := data.Append(t.Context(), "log", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := shared.WriteCache(t.Context(), "flashcards", "due", []byte("3")); err != nil {
		t.Fatal(err)
	}

	got, err := data.Read(t.Context(), "log")
	cached, found, cacheErr := shared.ReadCache(t.Context(), "flashcards", "due")
	_, missing := data.Read(t.Context(), "due")
	if err != nil || string(got) != "a" || cacheErr != nil || !found || string(cached) != "3" || !errors.Is(missing, fs.ErrNotExist) {
		t.Errorf("data %q (%v), cache %q %v (%v), missing %v", got, err, cached, found, cacheErr, missing)
	}
}

func TestAnotherVaultOfTheInstallationIsWorkedByItsIdentity(t *testing.T) {
	v, _, assembly := openPluginVault(t, note)
	nothing := func() (domain.Vault, bool) { return domain.Vault{}, false }
	shared, err := assembly.OpenPluginVault(nothing, container.PluginVaultHooks{})
	if err != nil {
		t.Fatal(err)
	}

	known, listErr := shared.GetVaults()
	bound, withErr := shared.With(v.ID)
	_, unknown := shared.With("nobody")

	if listErr != nil || len(known) != 1 || known[0].ID != v.ID {
		t.Errorf("the vaults were %v (%v)", known, listErr)
	}
	if withErr != nil {
		t.Fatal(withErr)
	}
	if got, showing := bound.GetVault(); !showing || got.ID != v.ID {
		t.Errorf("the bound vault was %v", got)
	}
	if !errors.Is(unknown, fs.ErrNotExist) {
		t.Errorf("an unknown identity was answered %v", unknown)
	}
}

func TestAWriteByAPluginIsToldToTheWindow(t *testing.T) {
	_, current, assembly := openPluginVault(t, note)
	told := 0
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Written: func() { told++ }})
	if err != nil {
		t.Fatal(err)
	}

	if err := shared.Create(t.Context(), "Made.md", []byte("# Made\n")); err != nil {
		t.Fatal(err)
	}

	if told != 1 {
		t.Errorf("the window was told %d times", told)
	}
}
