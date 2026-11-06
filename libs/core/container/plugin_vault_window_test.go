package container_test

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// movesWindow is a window that records the note moves it is shown.
type movesWindow struct {
	mu    sync.Mutex
	moves []domain.Move
}

var _ windowport.Window = (*movesWindow)(nil)

func (w *movesWindow) FocusPlace(context.Context, domain.Place) error { return nil }

func (w *movesWindow) ShowMove(_ context.Context, move domain.Move) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.moves = append(w.moves, move)
	return nil
}

func (w *movesWindow) ShowEdit(context.Context, domain.Edit) error { return nil }

func (w *movesWindow) getMoves() []domain.Move {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.moves)
}

func TestTheWindowFollowsANoteAPluginRenamesOrMoves(t *testing.T) {
	notes := getRivers()
	v, current, assembly := openPluginVault(t, notes)
	refresh := assembly.GetRefresh()
	window := &movesWindow{}
	scenarios := assembly.OpenNotes(func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}).FollowMoves(window)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Notes: scenarios})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Scan(t.Context(), []string{"Danube.md", "Deltas.md", "Estuaries.md"}); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := shared.RenameNote(t.Context(), "Danube.md", "Donau"); err != nil {
		t.Fatalf("the rename in %s failed: %v", v.Name, err)
	}
	if _, err := shared.MoveNote(t.Context(), "Donau.md", "Rivers/Donau.md"); err != nil {
		t.Fatalf("the move failed: %v", err)
	}

	want := []domain.Move{{From: "Danube.md", To: "Donau.md"}, {From: "Donau.md", To: "Rivers/Donau.md"}}
	if got := window.getMoves(); !slices.Equal(got, want) {
		t.Errorf("the window was shown %+v, want %+v", got, want)
	}
}

// openTwoPluginVaults registers a shown vault of rivers and another vault of lakes, whose notes share
// no words, and returns the assembly over both with the shown one current.
func openTwoPluginVaults(t *testing.T) (other domain.Vault, current func() (domain.Vault, bool), assembly container.EditorAssembly) {
	t.Helper()
	cfg := container.NewConfig(container.Config{IndexPath: indexfile.Path(t), PluginCachePath: t.TempDir(), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")})
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	rivers := testsupport.NewVault(t, map[string]string{"Danube.md": "# Danube\n\nA river.\n"})
	other = testsupport.NewVault(t, map[string]string{"Baikal.md": "# Baikal\n\nA deep lake.\n", "Lakes.md": "# Lakes\n"})
	for _, v := range []domain.Vault{rivers, other} {
		if err := registry.Save(v); err != nil {
			t.Fatal(err)
		}
	}
	return other, func() (domain.Vault, bool) { return rivers, true }, container.NewEditorAssembly(cfg, registry, db)
}

func TestTheWindowIsNotShownAPluginMoveInAnotherVault(t *testing.T) {
	other, current, assembly := openTwoPluginVaults(t)
	refresh := assembly.GetRefresh()
	window := &movesWindow{}
	scenarios := assembly.OpenNotes(func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}).FollowMoves(window)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Notes: scenarios})
	if err != nil {
		t.Fatal(err)
	}
	lakes, err := shared.With(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := lakes.Scan(t.Context(), []string{"Baikal.md", "Lakes.md"}); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := lakes.RenameNote(t.Context(), "Baikal.md", "Danube"); err != nil {
		t.Fatalf("the rename in %s failed: %v", other.Name, err)
	}
	if _, err := lakes.MoveNote(t.Context(), "Lakes.md", "Siberia/Lakes.md"); err != nil {
		t.Fatalf("the move in %s failed: %v", other.Name, err)
	}

	if got := window.getMoves(); len(got) != 0 {
		t.Errorf("the window showing another vault was shown %+v from %s", got, other.Name)
	}
}
