package container_test

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// moveRecordingWindow is a window that records the note moves it is shown.
type moveRecordingWindow struct {
	mu    sync.Mutex
	moves []domain.Move
}

func (w *moveRecordingWindow) FocusPlace(context.Context, domain.Place) error { return nil }

func (w *moveRecordingWindow) ShowMove(_ context.Context, move domain.Move) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.moves = append(w.moves, move)
	return nil
}

func (w *moveRecordingWindow) ShowEdit(context.Context, domain.Edit) error { return nil }

func (w *moveRecordingWindow) ActivateTab(context.Context, string) error { return nil }

func (w *moveRecordingWindow) getMoves() []domain.Move {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.moves)
}

func TestTheWindowFollowsANoteAPluginRenamesOrMoves(t *testing.T) {
	notes := getRivers()
	v, current, assembly := openPluginVault(t, notes)
	refresh := assembly.GetRefresh()
	window := &moveRecordingWindow{}
	scenarios := assembly.OpenNotesShowingMoves(func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}, current, window)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Notes: scenarios})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Writer.Scan(t.Context(), []string{"Danube.md", "Deltas.md", "Estuaries.md"}); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := shared.Arranger.RenameNote(t.Context(), "Danube.md", "Donau"); err != nil {
		t.Fatalf("the rename in %s failed: %v", v.Name, err)
	}
	if _, err := shared.Arranger.MoveNote(t.Context(), "Donau.md", "Rivers/Donau.md"); err != nil {
		t.Fatalf("the move failed: %v", err)
	}

	want := []domain.Move{{From: "Danube.md", To: "Donau.md"}, {From: "Donau.md", To: "Rivers/Donau.md"}}
	if got := window.getMoves(); !slices.Equal(got, want) {
		t.Errorf("the window was shown %+v, want %+v", got, want)
	}
}

func openTwoPluginVaults(t *testing.T) (other domain.Vault, current func() (domain.Vault, bool), assembly windowVaults) {
	t.Helper()
	cfg := container.NewConfig(container.Config{
		PluginHost: newExternalPluginHost(t, "md.numen.test", t.TempDir()),
		Paths:      container.Paths{IndexPath: indexfile.Path(t), PluginCachePath: t.TempDir(), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	rivers := vaultfixture.NewVault(t, map[string]string{"Danube.md": "# Danube\n\nA river.\n"})
	other = vaultfixture.NewVault(t, map[string]string{"Baikal.md": "# Baikal\n\nA deep lake.\n", "Lakes.md": "# Lakes\n"})
	testsupport.SaveVaults(t, cfg.RegistryPath, rivers, other)
	return other, func() (domain.Vault, bool) { return rivers, true }, windowVaults{cfg: cfg, db: db}
}

func TestWindowIgnoresPluginMoveInOtherVault(t *testing.T) {
	other, current, assembly := openTwoPluginVaults(t)
	refresh := assembly.GetRefresh()
	window := &moveRecordingWindow{}
	scenarios := assembly.OpenNotesShowingMoves(func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}, current, window)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Notes: scenarios})
	if err != nil {
		t.Fatal(err)
	}
	lakes, err := shared.Switch.With(other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := lakes.Writer.Scan(t.Context(), []string{"Baikal.md", "Lakes.md"}); err != nil {
		t.Fatal(err)
	}

	if _, _, _, err := lakes.Arranger.RenameNote(t.Context(), "Baikal.md", "Danube"); err != nil {
		t.Fatalf("the rename in %s failed: %v", other.Name, err)
	}
	if _, err := lakes.Arranger.MoveNote(t.Context(), "Lakes.md", "Siberia/Lakes.md"); err != nil {
		t.Fatalf("the move in %s failed: %v", other.Name, err)
	}

	if got := window.getMoves(); len(got) != 0 {
		t.Errorf("the window showing another vault was shown %+v from %s", got, other.Name)
	}
}

// vaultMoveRecordingWindow records each move it is shown with the vault the window showed at that moment.
type vaultMoveRecordingWindow struct {
	moveRecordingWindow
	current  func() (domain.Vault, bool)
	shownIDs []domain.VaultID
}

func (w *vaultMoveRecordingWindow) ShowMove(ctx context.Context, move domain.Move) error {
	v, _ := w.current()
	w.mu.Lock()
	w.shownIDs = append(w.shownIDs, v.ID)
	w.mu.Unlock()
	return w.moveRecordingWindow.ShowMove(ctx, move)
}

func TestWindowIgnoresPluginRenameCompletingAfterVaultSwitch(t *testing.T) {
	cfg := container.NewConfig(container.Config{
		PluginHost: newExternalPluginHost(t, "md.numen.test", t.TempDir()),
		Paths:      container.Paths{IndexPath: indexfile.Path(t), PluginCachePath: t.TempDir(), RegistryPath: filepath.Join(t.TempDir(), "vaults.json")},
	})
	db, err := container.OpenIndex(cfg, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	rivers := vaultfixture.NewVault(t, map[string]string{"Danube.md": "---\ntitle: Danube\n---\nA river.\n"})
	lakes := vaultfixture.NewVault(t, map[string]string{"Danube.md": "---\ntitle: Danube\n---\nA lake of the same name.\n"})
	testsupport.SaveVaults(t, cfg.RegistryPath, rivers, lakes)
	var isSwitched atomic.Bool
	current := func() (domain.Vault, bool) {
		if isSwitched.Load() {
			return lakes, true
		}
		return rivers, true
	}
	assembly := windowVaults{cfg: cfg, db: db}
	refresh := assembly.GetRefresh()
	window := &vaultMoveRecordingWindow{current: current}
	scenarios := assembly.OpenNotesShowingMoves(func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		// The window changes vault while the rename writes the title, before the file moves.
		isSwitched.Store(true)
		return err
	}, current, window)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Notes: scenarios})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Writer.Scan(t.Context(), []string{"Danube.md"}); err != nil {
		t.Fatal(err)
	}
	isSwitched.Store(false)

	if _, _, _, err := shared.Arranger.RenameNote(t.Context(), "Danube.md", "Donau"); err != nil {
		t.Fatalf("the rename in %s failed: %v", rivers.Name, err)
	}

	moves := window.getMoves()
	window.mu.Lock()
	defer window.mu.Unlock()
	for i, shownID := range window.shownIDs {
		if shownID != rivers.ID {
			t.Errorf("the move %+v of %s was shown to the window while it showed %s", moves[i], rivers.Name, lakes.Name)
		}
	}
}

func TestWindowMoveAfterPluginMoveIsShownToTheWindow(t *testing.T) {
	v, current, assembly := openPluginVault(t, getRivers())
	refresh := assembly.GetRefresh()
	window := &moveRecordingWindow{}
	scenarios := assembly.OpenNotesShowingMoves(func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}, current, window)
	shared, err := assembly.OpenPluginVault(current, container.PluginVaultHooks{Notes: scenarios})
	if err != nil {
		t.Fatal(err)
	}
	if err := shared.Writer.Scan(t.Context(), []string{"Danube.md", "Deltas.md", "Estuaries.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := shared.Arranger.MoveNote(t.Context(), "Danube.md", "Donau.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := scenarios.Place.Move.Execute(t.Context(), v, "Deltas.md", "Delta.md"); err != nil {
		t.Fatal(err)
	}
	moves := window.getMoves()
	if len(moves) != 2 || moves[1].To != "Delta.md" {
		t.Errorf("the window was shown %+v, want the plugin's move and then its own", moves)
	}
}
