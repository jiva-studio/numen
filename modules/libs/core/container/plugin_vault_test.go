package container_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func openPluginVault(t *testing.T, notes map[string]string) (domain.Vault, func() (domain.Vault, bool), container.EditorAssembly) {
	t.Helper()
	cfg := container.Config{IndexPath: indexfile.Path(t), SchedulesPath: t.TempDir()}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	v := testsupport.NewVault(t, notes)
	return v, func() (domain.Vault, bool) { return v, true }, container.NewEditorAssembly(cfg, nil, db)
}

func TestAPluginNoteIsInTheIndexOnceItsFileIsScanned(t *testing.T) {
	v, current, assembly := openPluginVault(t, note)
	shared, err := assembly.OpenPluginVault(current)
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
	shared, err := assembly.OpenPluginVault(current)
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
