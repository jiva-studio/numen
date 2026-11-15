package container_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// unopenable is readers on a machine that cannot open any vault.
type unopenable struct{ err error }

func (u unopenable) Open(domain.Vault) (vaultport.VaultReader, error) { return nil, u.err }

func TestAWalkAnswersHowManyNotesTheVaultContains(t *testing.T) {
	cfg, db, v := openVault(t, map[string]string{
		"One.md":     "---\ntitle: One\n---\n\n# One\n",
		"Two.md":     "---\ntitle: Two\n---\n\n# Two\n",
		"sub/Six.md": "---\ntitle: Six\n---\n\n# Six\n",
	})
	open := container.VaultOpener(cfg, db).Begin(t.Context(), v, nil, nil)

	got, err := open.Read(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Errorf("the vault holds %d notes, the walk says %d", 3, got)
	}
}

// A walk that cannot open the vault returns the error and reports no notes as read.
func TestAWalkThatCannotOpenTheVaultReturnsSo(t *testing.T) {
	cfg, db, v := openVault(t, getNote())
	broken := errors.New("the disk is gone")
	open := cfg.VaultOpenerWith(db, unopenable{err: broken}, cfg.VaultWatcher()).Begin(t.Context(), v, nil, nil)

	got, err := open.Read(t.Context(), nil)
	if !errors.Is(err, broken) {
		t.Errorf("the walk said %v", err)
	}
	if got != 0 {
		t.Errorf("the walk read %d notes", got)
	}
}

// A vault that cannot be watched has nothing to run, and running it returns.
func TestRunningAVaultThatCannotBeWatchedReturns(t *testing.T) {
	cfg, db, v := openVault(t, getNote())
	open := cfg.VaultOpenerWith(db, container.VaultReaders(cfg), refusing{}).Begin(t.Context(), v, nil, nil)

	isDone := make(chan struct{})
	go func() {
		open.Run(t.Context())
		close(isDone)
	}()
	receiveFrom(t, isDone, "running an unwatched vault went on")
}

// The walk of an opener rebuilds the index exactly when the opener requests it.
func TestTheWalkRebuildsTheIndexWhenTheOpenerReturnsSo(t *testing.T) {
	cfg, db, _ := openVault(t, getNote())
	opening := container.VaultOpener(cfg, db)

	if opening.GetScan().ShouldRebuildIndex {
		t.Error("the walk rebuilds the index, and the opener did not say so")
	}
	opening.ShouldRebuild = true
	if !opening.GetScan().ShouldRebuildIndex {
		t.Error("the opener says to rebuild the index, and the walk does not")
	}
	opening.ShouldRebuild = false
	if opening.GetScan().ShouldRebuildIndex {
		t.Error("the walk rebuilds the index after the opener stopped saying so")
	}
}

// Errors raised while the vault is followed reach the opener's handler, as does
// the end of the follow.
func TestVaultOpener_ReportsFollowErrorsAndEnd(t *testing.T) {
	cfg, db, v := openVault(t, getNote())
	watcher := makeWatcher()
	opening := cfg.VaultOpenerWith(db, container.VaultReaders(cfg), watcher)
	received := make(chan error, 4)
	changesCh := make(chan vault.VaultChanges, 4)

	open := opening.Begin(t.Context(), v, func(m vault.VaultChanges) { changesCh <- m }, func(err error) { received <- err })
	go open.Run(t.Context())

	watcher.lostChan <- struct{}{}
	if err := receiveFrom(t, received, "the handler heard nothing"); err != nil {
		t.Errorf("the handler heard %v, where the vault was read again without trouble", err)
	}
	receiveFrom(t, changesCh, "the vault was not read again")
}

// An index path below a plain file has no folder to be kept in.
func TestAnIndexPathBelowAFileCannotBeOpened(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "plain")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(file, "index.db")},
	}), t.Context())
	if err == nil {
		testsupport.Close(t, db)
		t.Fatal("an index was opened below a file")
	}
	if db != nil {
		t.Error("an index came back with the error")
	}
}

// A folder is not an index, and opening one returns an error.
func TestAFolderCannotBeOpenedAsAnIndex(t *testing.T) {
	isolate(t)
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: t.TempDir()},
	}), t.Context())
	if err == nil {
		testsupport.Close(t, db)
		t.Fatal("a folder was opened as an index")
	}
	if db != nil {
		t.Error("an index came back with the error")
	}
}

// An index returns the path of its file.
func TestAnIndexReturnsWhereItsFileIs(t *testing.T) {
	cfg, db, _ := openVault(t, getNote())

	if db.File != cfg.IndexPath {
		t.Errorf("the index is at %q, not %q", db.File, cfg.IndexPath)
	}
}

// Compacting an open index succeeds, and compacting a closed one returns an error.
func TestCompact_SucceedsOpenAndFailsClosed(t *testing.T) {
	_, db, _ := openVault(t, getNote())

	if err := db.VectorIndex().Compact(t.Context()); err != nil {
		t.Fatalf("an open index rejected: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.VectorIndex().Compact(t.Context()); err == nil {
		t.Error("a closed index made room")
	}
}

func TestWatchingAnIndexWithNoFolderReturnsSo(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	moves, err := container.Moves(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(file, "index.db")},
	}), t.Context())
	if err == nil {
		t.Error("an index below a file was watched")
	}
	if moves != nil {
		t.Error("a channel came back with the error")
	}
}

// Assets are read when an asset changed or everything changed.
func TestShouldReadAssets_TrueForAssetsOrReload(t *testing.T) {
	for _, one := range []struct {
		what string
		got  vault.VaultChanges
		want bool
	}{
		{"nothing", vault.VaultChanges{}, false},
		{"a note", vault.VaultChanges{Paths: []string{"a.md"}}, false},
		{"an asset", vault.VaultChanges{Assets: []string{"a.png"}}, true},
		{"everything", vault.VaultChanges{ShouldReload: true}, true},
		{"both", vault.VaultChanges{ShouldReload: true, Assets: []string{"a.png"}}, true},
	} {
		if got := one.got.ShouldReadAssets(); got != one.want {
			t.Errorf("%s: %v", one.what, got)
		}
	}
}

// The refresh an opener hands out brings named notes up to date.
func TestTheRefreshAnOpenerPassesOutIndexesNamedNotes(t *testing.T) {
	cfg, db, v := openVault(t, getNote())

	res, err := container.VaultOpener(cfg, db).GetRefresh().Execute(t.Context(), v, []string{"Leaf.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Indexed) != 1 {
		t.Errorf("indexed %v", res.Indexed)
	}
	if got := titleOf(t, db, v, "Leaf.md"); got != "Leaf" {
		t.Errorf("the index says %q", got)
	}
}

func TestAWalkThatStopsReturnsHowManyNotesItHadRead(t *testing.T) {
	notes := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d"} {
		notes[name+".md"] = "---\ntitle: " + name + "\n---\n\n# " + name + "\n"
	}
	cfg, db, v := openVault(t, notes)
	open := container.VaultOpener(cfg, db).Begin(t.Context(), v, nil, nil)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()

	got, err := open.Read(ctx, func() { stop() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("the walk said %v", err)
	}
	if got == 0 {
		t.Error("the walk says it read nothing")
	}
}
