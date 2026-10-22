package container_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// unopenable is readers on a machine that cannot open any vault.
type unopenable struct{ err error }

func (u unopenable) Open(domain.Vault) (port.VaultReader, error) { return nil, u.err }

// The walk answers how many notes the vault holds.
func TestAWalkAnswersHowManyNotesTheVaultContains(t *testing.T) {
	cfg, db, v := openVault(t, map[string]string{
		"One.md":     "---\ntitle: One\n---\n\n# One\n",
		"Two.md":     "---\ntitle: Two\n---\n\n# Two\n",
		"sub/Six.md": "---\ntitle: Six\n---\n\n# Six\n",
	})
	open := cfg.VaultOpener(db).Begin(t.Context(), v)

	got, err := open.Read(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Errorf("the vault holds %d notes, the walk says %d", 3, got)
	}
}

// A walk that cannot open the vault says so, and nothing is reported as read.
func TestAWalkThatCannotOpenTheVaultReturnsSo(t *testing.T) {
	cfg, db, v := openVault(t, note)
	broken := errors.New("the disk is gone")
	open := cfg.VaultOpenerWith(db, unopenable{err: broken}, cfg.VaultWatcher()).Begin(t.Context(), v)

	got, err := open.Read(t.Context(), nil)
	if !errors.Is(err, broken) {
		t.Errorf("the walk said %v", err)
	}
	if got != 0 {
		t.Errorf("the walk read %d notes", got)
	}
}

// A vault nobody can follow has nothing to run, and running it returns.
func TestRunningAVaultThatCannotBeWatchedReturns(t *testing.T) {
	cfg, db, v := openVault(t, note)
	open := cfg.VaultOpenerWith(db, cfg.VaultReaders(), refusing{}).Begin(t.Context(), v)

	isDone := make(chan struct{})
	go func() {
		open.Run(t.Context())
		close(isDone)
	}()
	receiveFrom(t, isDone, "running an unwatched vault went on")
}

// The walk an opener makes rebuilds the index exactly when the opener says so.
func TestTheWalkRebuildsTheIndexWhenTheOpenerReturnsSo(t *testing.T) {
	cfg, db, _ := openVault(t, note)
	opening := cfg.VaultOpener(db)

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

// What went wrong while the vault is followed reaches the opener's handler, and
// so does the end of it.
func TestTheOpenersHandlerHearsHowFollowingTheVaultWent(t *testing.T) {
	cfg, db, v := openVault(t, note)
	watcher := makeWatcher()
	opening := cfg.VaultOpenerWith(db, cfg.VaultReaders(), watcher)
	received := make(chan error, 4)
	opening.ErrorHandler = func(err error) { received <- err }
	changesCh := make(chan container.VaultChanges, 4)
	opening.Told = func(m container.VaultChanges) { changesCh <- m }

	open := opening.Begin(t.Context(), v)
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

	db, err := container.Config{IndexPath: filepath.Join(file, "index.db")}.OpenIndex(t.Context())
	if err == nil {
		db.Close()
		t.Fatal("an index was opened below a file")
	}
	if db != nil {
		t.Error("an index came back with the error")
	}
}

// A folder is not an index, and opening one says so.
func TestAFolderCannotBeOpenedAsAnIndex(t *testing.T) {
	isolate(t)
	db, err := container.Config{IndexPath: t.TempDir()}.OpenIndex(t.Context())
	if err == nil {
		db.Close()
		t.Fatal("a folder was opened as an index")
	}
	if db != nil {
		t.Error("an index came back with the error")
	}
}

// Every port an index answers is there to be used.
func TestAnIndexAnswersEveryPortItNames(t *testing.T) {
	_, db, _ := openVault(t, note)

	if db.Vectors() == nil || db.VectorQueries() == nil || db.Sources() == nil || db.SourcesKnown() == nil {
		t.Error("a port of the index answers nothing")
	}
}

// An index answers every port it names, and says where its file is.
func TestAnIndexReturnsWhereItsFileIsAndAnswersItsPorts(t *testing.T) {
	cfg, db, _ := openVault(t, note)

	if db.Path() != cfg.IndexPath {
		t.Errorf("the index is at %q, not %q", db.Path(), cfg.IndexPath)
	}
	if db.Notes() == nil || db.Progress() == nil || db.Links() == nil || db.Problems() == nil {
		t.Error("a port of the index answers nothing")
	}
}

// Making room in an index that is open works, and in one that is closed says so.
func TestAnIndexCompactsWhileOpenAndRefusesOnceClosed(t *testing.T) {
	_, db, _ := openVault(t, note)

	if err := db.Compact(t.Context()); err != nil {
		t.Fatalf("an open index rejected: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db.Compact(t.Context()); err == nil {
		t.Error("a closed index made room")
	}
}

// Watching an index that has no folder to stand in says so.
func TestWatchingAnIndexWithNoFolderReturnsSo(t *testing.T) {
	isolate(t)
	file := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	moves, err := container.Config{IndexPath: filepath.Join(file, "index.db")}.Moves(t.Context())
	if err == nil {
		t.Error("an index below a file was watched")
	}
	if moves != nil {
		t.Error("a channel came back with the error")
	}
}

// Whether an asset owes a read is whether one changed or everything did.
func TestAnAssetOwesAReadWhenOneChangedOrEverythingDid(t *testing.T) {
	for _, one := range []struct {
		what string
		got  container.VaultChanges
		want bool
	}{
		{"nothing", container.VaultChanges{}, false},
		{"a note", container.VaultChanges{Paths: []string{"a.md"}}, false},
		{"an asset", container.VaultChanges{Assets: []string{"a.png"}}, true},
		{"everything", container.VaultChanges{ShouldReload: true}, true},
		{"both", container.VaultChanges{ShouldReload: true, Assets: []string{"a.png"}}, true},
	} {
		if got := one.got.ShouldReadAssets(); got != one.want {
			t.Errorf("%s: %v", one.what, got)
		}
	}
}

// The refresh an opener hands out brings named notes up to date.
func TestTheRefreshAnOpenerPassesOutIndexesNamedNotes(t *testing.T) {
	cfg, db, v := openVault(t, note)

	res, err := cfg.VaultOpener(db).GetRefresh().Execute(t.Context(), v, []string{"Leaf.md"})
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

// A walk that stops partway still says how many notes it had read.
func TestAWalkThatStopsReturnsHowManyNotesItHadRead(t *testing.T) {
	notes := map[string]string{}
	for _, name := range []string{"a", "b", "c", "d"} {
		notes[name+".md"] = "---\ntitle: " + name + "\n---\n\n# " + name + "\n"
	}
	cfg, db, v := openVault(t, notes)
	open := cfg.VaultOpener(db).Begin(t.Context(), v)
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
