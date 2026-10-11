package container_test

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// unopenable is readers on a machine that cannot open any vault.
type unopenable struct{ err error }

func (u unopenable) Open(domain.Vault) (port.VaultReader, error) { return nil, u.err }

// reports collects what an opener says went wrong.
type reports struct {
	mu   sync.Mutex
	held []error
}

func (r *reports) add(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.held = append(r.held, err)
}

func (r *reports) get() []error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]error(nil), r.held...)
}

// The walk answers how many notes the vault holds.
func TestAWalkAnswersHowManyNotesTheVaultHolds(t *testing.T) {
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
func TestAWalkThatCannotOpenTheVaultSaysSo(t *testing.T) {
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

	done := make(chan struct{})
	go func() {
		open.Run(t.Context())
		close(done)
	}()
	receiveFrom(t, done, "running an unwatched vault went on")
}

// The walk an opener makes rebuilds the index exactly when the opener says so.
func TestTheWalkRebuildsTheIndexWhenTheOpenerSaysSo(t *testing.T) {
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
	heard := make(chan error, 4)
	opening.ErrorHandler = func(err error) { heard <- err }
	told := make(chan container.VaultChanges, 4)
	opening.Told = func(m container.VaultChanges) { told <- m }

	open := opening.Begin(t.Context(), v)
	go open.Run(t.Context())

	watcher.lost <- struct{}{}
	if err := receiveFrom(t, heard, "the handler heard nothing"); err != nil {
		t.Errorf("the handler heard %v, where the vault was read again without trouble", err)
	}
	receiveFrom(t, told, "the vault was not read again")
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

	if db.Vectors() == nil || db.VectorsOwing() == nil || db.Sources() == nil || db.SourcesKnown() == nil {
		t.Error("a port of the index answers nothing")
	}
}
