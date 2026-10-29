package vault_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// manualWatcher is a watcher whose events a test emits itself; vault change
// handling is tested without a filesystem, a timer or a server.
type manualWatcher struct {
	changes  chan []string
	lostChan chan struct{}
}

func newManualWatcher() *manualWatcher {
	return &manualWatcher{changes: make(chan []string), lostChan: make(chan struct{}, 1)}
}

func (h *manualWatcher) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return h.changes, h.lostChan, nil
}

// failingWatcher is a watcher that fails to start, which models an operating system
// that has run out of watches.
type failingWatcher struct{}

func (failingWatcher) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return nil, nil, errors.New("no watches left")
}

// sometimes is a set of readers that can be made to fail, which models a
// permission error or a removed device. The flag is written from one goroutine
// and read from another, so it is atomic.
type sometimes struct {
	vaultport.VaultReaders
	refuse atomic.Bool
}

func (s *sometimes) Open(v domain.Vault) (vaultport.VaultReader, error) {
	if s.refuse.Load() {
		return nil, errors.New("cannot open the vault")
	}
	return s.VaultReaders.Open(v)
}

// followed is a vault being watched: the vault itself, the index that is being
// kept level with it, and what the following reported.
type followed struct {
	vault domain.Vault
	index *container.Index
	moved <-chan vaults.VaultChanges
}

// newFollowed puts one vault, one index and a watcher a test drives together.
func newFollowed(t *testing.T, notes map[string]string, watcher *manualWatcher) followed {
	t.Helper()
	v := testsupport.NewVault(t, notes)
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	moved := make(chan vaults.VaultChanges, 8)
	follow := vaults.NewFollow(
		watcher,
		vaults.NewRefresh(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), time.Now),
		scanner(vaultreader.Readers{}, db),
	)
	follow.Changed = func(m vaults.VaultChanges) { moved <- m }

	started, err := follow.Begin(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	go started.Run(t.Context())
	return followed{vault: v, index: db, moved: moved}
}

func next[T any](t *testing.T, from <-chan T) T {
	t.Helper()
	select {
	case value := <-from:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was reported")
		var zero T
		return zero
	}
}

// TestAChangedNoteIsBroughtUpToDateAndReported.
func TestAChangedNoteIsBroughtUpToDateAndReported(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	f := newFollowed(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	}, watcher)

	if err := os.WriteFile(filepath.Join(f.vault.Path, "Note.md"),
		[]byte("---\ntitle: Renamed\n---\n\n# Renamed\n\nentropy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	watcher.changes <- []string{"Note.md"}

	if got := next(t, f.moved); !slices.Equal(got.Paths, []string{"Note.md"}) {
		t.Errorf("reported %+v", got)
	}
	if got := titles(t, f.index, f.vault, "entropy"); !slices.Equal(got, []string{"Renamed"}) {
		t.Errorf("the index holds %v", got)
	}
}

// TestABookThatChangedIsNotANoteThatMoved. The watcher reports changes to every source
// type; the listener is notified only about notes.
func TestABookThatChangedIsNotANoteThatMoved(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	f := newFollowed(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	}, watcher)
	testsupport.WriteBook(t, f.vault.Path, "library/A Book.epub")

	watcher.changes <- []string{"library/A Book.epub", "Note.md"}

	got := next(t, f.moved)
	if !slices.Equal(got.Paths, []string{"Note.md"}) {
		t.Errorf("reported %+v", got)
	}
	// A book change is not a note move, and it is not ignored: reading a book is
	// separate work, so the listener receives the book that needs it.
	if !slices.Equal(got.Assets, []string{"library/A Book.epub"}) {
		t.Errorf("the book was reported as %v", got.Assets)
	}
	if got := titles(t, f.index, f.vault, "entropy"); !slices.Equal(got, []string{"Note"}) {
		t.Errorf("the index holds %v", got)
	}
}

// Not knowing what changed is answered by looking at everything, books included.
func TestWhatCannotBeFollowedConsumesTheBooksWithIt(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	f := newFollowed(t, map[string]string{"Note.md": "# Note\n"}, watcher)

	watcher.lostChan <- struct{}{}

	if got := next(t, f.moved); !got.ShouldReload || !got.ShouldReadAssets() {
		t.Errorf("reported %+v", got)
	}
}

// TestWhatCannotBeFollowedIsRead. When the changed paths are unknown, the only
// correct response is to rescan everything, and the listener is notified to read
// again and does not receive a list of moved notes.
func TestWhatCannotBeFollowedIsRead(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	f := newFollowed(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	}, watcher)

	if err := os.WriteFile(filepath.Join(f.vault.Path, "Later.md"),
		[]byte("---\ntitle: Later\n---\n\n# Later\n\nentropy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	watcher.lostChan <- struct{}{}

	got := next(t, f.moved)
	if !got.ShouldReload {
		t.Fatalf("reported %+v, want the whole picture asked for again", got)
	}
	if len(got.Paths) != 0 || len(got.Assets) != 0 {
		t.Errorf("reported %+v, and what changed cannot be known", got)
	}
	if got := titles(t, f.index, f.vault, "entropy"); !slices.Contains(got, "Later") {
		t.Errorf("the index holds %v", got)
	}
}

// TestATroubleThatIsOverStopsBeingReported. A message that persists after the
// trouble ends reports a failure while the application is working.
func TestATroubleThatIsOverStopsBeingReported(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	v := testsupport.NewVault(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n\nentropy\n",
	})
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	reported := make(chan error, 8)
	readers := &sometimes{VaultReaders: vaultreader.Readers{}}
	follow := vaults.NewFollow(
		watcher,
		vaults.NewRefresh(readers, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), time.Now),
		scanner(vaultreader.Readers{}, db),
	)
	follow.ErrorHandler = func(err error) { reported <- err }
	started, err := follow.Begin(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	go started.Run(t.Context())

	readers.refuse.Store(true)
	watcher.changes <- []string{"Note.md"}
	if err := next(t, reported); err == nil {
		t.Fatal("a vault that could not be opened was reported as working")
	}

	readers.refuse.Store(false)
	watcher.changes <- []string{"Note.md"}
	if err := next(t, reported); err != nil {
		t.Errorf("still reporting %v after it worked", err)
	}
}

// TestAVaultThatCannotBeWatchedSaysSo. It is still a vault: only following it
// fails, and the failure must reach the listener so the window does not appear
// up to date.
func TestAVaultThatCannotBeWatchedReturnsSo(t *testing.T) {
	t.Parallel()
	v := testsupport.NewVault(t, map[string]string{"Note.md": "# Note\n"})
	// No refresh or scan is available: a watch that never starts uses neither.
	_, err := vaults.NewFollow(failingWatcher{}, vaults.Refresh{}, vaults.Scan{}).Begin(t.Context(), v)
	if err == nil {
		t.Fatal("a watcher that could not start was taken for one that did")
	}
}

func TestWhatAVaultChangedAsksForAReadWhenAnythingMayHaveChanged(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		changes vaults.VaultChanges
		want    bool
	}{
		{"nothing", vaults.VaultChanges{}, false},
		{"notes only", vaults.VaultChanges{Paths: []string{"Note.md"}}, false},
		{"one asset", vaults.VaultChanges{Assets: []string{"Book.pdf"}}, true},
		{"two assets", vaults.VaultChanges{Assets: []string{"A.pdf", "B.pdf"}}, true},
		{"a reload", vaults.VaultChanges{ShouldReload: true}, true},
	} {
		if got := tc.changes.ShouldReadAssets(); got != tc.want {
			t.Errorf("%s: ShouldReadAssets is %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestAWatchEndsWhenTheWatcherStops(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	v := testsupport.NewVault(t, map[string]string{"Note.md": "# Note\n"})
	db := openIndex(t)
	follow := vaults.NewFollow(
		watcher,
		vaults.NewRefresh(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), time.Now),
		scanner(vaultreader.Readers{}, db),
	)
	started, err := follow.Begin(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	isDone := make(chan struct{})
	go func() {
		started.Run(t.Context())
		close(isDone)
	}()

	close(watcher.changes)

	select {
	case <-isDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the watch went on after its watcher stopped")
	}
}

// A walk that fails reports the failure and not a vault that is level again,
// and the report clears once a walk succeeds.
func TestAWalkThatFailsIsReportedAndClearedByTheNextOne(t *testing.T) {
	t.Parallel()
	watcher := newManualWatcher()
	v := testsupport.NewVault(t, map[string]string{"Note.md": "# Note\n"})
	db := openIndex(t)
	if _, err := scanner(vaultreader.Readers{}, db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	reported := make(chan error, 8)
	moved := make(chan vaults.VaultChanges, 8)
	readers := &sometimes{VaultReaders: vaultreader.Readers{}}
	follow := vaults.NewFollow(
		watcher,
		vaults.NewRefresh(vaultreader.Readers{}, db.Vaults(), db.Notes(), db.SourcesKnown(), db.Sources(), time.Now),
		scanner(readers, db),
	)
	follow.ErrorHandler = func(err error) { reported <- err }
	follow.Changed = func(m vaults.VaultChanges) { moved <- m }
	started, err := follow.Begin(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}
	go started.Run(t.Context())

	readers.refuse.Store(true)
	watcher.lostChan <- struct{}{}
	if err := next(t, reported); err == nil {
		t.Fatal("a walk that could not open the vault was reported as working")
	}
	select {
	case m := <-moved:
		t.Fatalf("a failed walk reported %+v", m)
	default:
	}

	readers.refuse.Store(false)
	watcher.lostChan <- struct{}{}
	if err := next(t, reported); err != nil {
		t.Errorf("still reporting %v after the walk worked", err)
	}
	if got := next(t, moved); !got.ShouldReload {
		t.Errorf("reported %+v, want the whole picture asked for again", got)
	}
}
