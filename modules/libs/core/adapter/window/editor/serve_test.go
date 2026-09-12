package editor

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// hand is a watch a test drives itself, so what a window is told while a vault
// is being read can be asked without a filesystem or a timer.
type hand struct {
	changes chan []string
	lost    chan struct{}
}

func byHand() *hand {
	return &hand{changes: make(chan []string), lost: make(chan struct{}, 1)}
}

func (h *hand) Watch(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return h.changes, h.lost, nil
}

// unwatchable is a watch that will not start, which is what an operating system
// out of watches looks like from here.
type unwatchable struct{}

func (unwatchable) Watch(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return nil, nil, errors.New("no watches left")
}

// waiting is a set of readers whose reads stop where a test can see them and
// finish when it says so, which is what a vault that takes minutes to read
// looks like from here.
type waiting struct {
	port.VaultReaders
	// begun carries the path of a read that is under way.
	begun chan string
	until chan struct{}
	once  sync.Once
}

func slowly() *waiting {
	return &waiting{
		VaultReaders: filesystem.VaultReaders{},
		begun:        make(chan string, 1),
		until:        make(chan struct{}),
	}
}

// release lets every read through, from now on.
func (w *waiting) release() { w.once.Do(func() { close(w.until) }) }

func (w *waiting) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := w.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return waits{VaultReader: reader, at: w}, nil
}

// waits holds a read open once the bytes are in hand, so a test can change the
// file the reader is holding a copy of.
type waits struct {
	port.VaultReader
	at *waiting
}

func (r waits) Read(ctx context.Context, path string) ([]byte, error) {
	raw, err := r.VaultReader.Read(ctx, path)
	select {
	case r.at.begun <- path:
	default:
	}
	select {
	case <-r.at.until:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return raw, err
}

// unwalkable is a set of readers that cannot list the vault, which is what a
// folder the walk is refused looks like from here. A note named outright is
// still read.
type unwalkable struct{ port.VaultReaders }

func (u unwalkable) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := u.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return unlisted{VaultReader: reader}, nil
}

type unlisted struct{ port.VaultReader }

func (unlisted) Walk(context.Context, func(domain.Fingerprint) error) error {
	return errors.New("the vault cannot be listed")
}

// unreadable is a set of readers whose walk fails, which is what a device that
// went away looks like from here. A single file nobody can read is counted and
// walked past, so it is the walk itself that has to go.
type unreadable struct{ port.VaultReaders }

func (u unreadable) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := u.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return refuses{VaultReader: reader}, nil
}

type refuses struct{ port.VaultReader }

func (refuses) Walk(context.Context, func(domain.Fingerprint) error) error {
	return errors.New("the vault cannot be read")
}

// behind is one vault with the work a window has running behind it: the index
// it fills, the first scan, the watch beside it, and what a client listening
// for changes is told.
type behind struct {
	api   *API
	vault domain.Vault
	index *container.Index
	heard <-chan change
}

// opening puts a vault, an index, a scan and a watch together the way Open
// does, with the pieces a test drives in place of the ones a window gets.
func opening(t *testing.T, notes map[string]string, watcher port.VaultWatcher, readers port.VaultReaders) *behind {
	t.Helper()
	return openingWith(t, notes, watcher, readers, nil, settled)
}

// openingWith is the same, with the model that fills the index for meaning and
// how long the vault has to have been still before a note is embedded.
func openingWith(
	t *testing.T,
	notes map[string]string,
	watcher port.VaultWatcher,
	readers port.VaultReaders,
	embedder port.Embedder,
	still time.Duration,
) *behind {
	t.Helper()

	v := testsupport.NewVault(t, notes)
	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	writing := note.NewWrite(
		filesystem.VaultReaders{}, filesystem.VaultWriters{}, unlevelled, time.Now)
	api := &API{
		Listeners: following(),
		Places:    focusing(),
		Notes: Notes{
			Queries: db.Queries(),
			Links:   db.Links(),
			Read:    &note.Read{Readers: filesystem.VaultReaders{}},
			Write:   &writing,
		},
	}
	api.show(v)
	if embedder != nil {
		if err := db.FitVectors(t.Context(), embedder.Model().Dimensions, embedder.Model().Recipe()); err != nil {
			t.Fatal(err)
		}
		api.Indexing.Model.Store(embedder.Model().String())
		api.Indexing.Progress = db.Progress()
	}
	opened := cfg.VaultOpenerWith(db, readers, watcher)

	line, done := api.Listeners.listen()
	t.Cleanup(done)

	wake := waking(still)
	api.Wrote = func() { raise(wake.notes) }

	ctx, stop := context.WithCancel(t.Context())
	wait := begin(ctx, v, cfg, db, api, opened, readers, embedder, wake, &pending{}, io.Discard)
	t.Cleanup(func() {
		stop()
		wait()
	})

	return &behind{api: api, vault: v, index: db, heard: line}
}

func write(t *testing.T, v domain.Vault, path, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Path, filepath.FromSlash(path)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// save puts a body into a note the way the window does: through the handler a
// save lands in.
func save(t *testing.T, f *behind, path, body string) {
	t.Helper()
	out, err := f.api.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{Path: path, Body: body}))
	if err != nil {
		t.Fatal(err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the save of %s was refused: %v", path, code)
	}
}

// tells hands the watch a change, which takes somebody acting on the watch.
func tells(t *testing.T, watcher *hand, paths ...string) {
	t.Helper()
	select {
	case watcher.changes <- paths:
	case <-time.After(5 * time.Second):
		t.Fatal("nothing is acting on the watch")
	}
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

// eventually waits for something the work behind the window does in its own
// time.
func eventually(t *testing.T, what string, is func() bool) {
	t.Helper()
	for range 500 {
		if is() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(what)
}

func titleOf(t *testing.T, db *container.Index, v domain.Vault, path string) string {
	t.Helper()
	found, err := db.Queries().Notes(t.Context(), v.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	return found[path].Title
}

// TestAChangeArrivesWhileTheVaultIsStillBeingRead. A change reaches the window
// through the watch and through nothing else, and the vault is read for the
// first time with the person already working in it.
func TestAChangeArrivesWhileTheVaultIsStillBeingRead(t *testing.T) {
	watcher := byHand()
	readers := slowly()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, readers)

	// A read is under way and held, so the vault has not been read yet.
	if path := next(t, readers.begun); path != "Note.md" {
		t.Fatalf("the scan is reading %q", path)
	}
	if f.api.Ready.Load() {
		t.Fatal("the scan finished with its reads held")
	}

	write(t, f.vault, "Note.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	tells(t, watcher, "Note.md")

	if got := next(t, f.heard); len(got.paths) != 1 || got.paths[0] != "Note.md" {
		t.Errorf("the window was told %+v", got)
	}
	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "Renamed" {
		t.Errorf("the index says %q", title)
	}
	if f.api.Ready.Load() {
		t.Error("the scan finished with its reads held")
	}
}

// TestANoteChangedUnderTheScanIsReadAgain. The scan writes in groups from what
// it read, so its copy of a note lands last however early the note was read.
func TestANoteChangedUnderTheScanIsReadAgain(t *testing.T) {
	watcher := byHand()
	readers := slowly()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: One\n---\n\n# One\n",
	}, watcher, readers)

	// The scan holds the note as it was.
	next(t, readers.begun)

	write(t, f.vault, "Note.md", "---\ntitle: Two\n---\n\n# Two\n")
	tells(t, watcher, "Note.md")
	next(t, f.heard)

	// Now the scan writes what it read, and finishes.
	readers.release()
	eventually(t, "the scan did not finish", f.api.Ready.Load)

	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "Two" {
		t.Errorf("the index says %q", title)
	}
}

// TestANoteDeletedUnderTheScanStaysDeleted. The scan holds a copy of a note it
// read, and the note went while it held it.
func TestANoteDeletedUnderTheScanStaysDeleted(t *testing.T) {
	watcher := byHand()
	readers := slowly()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, readers)

	next(t, readers.begun)

	if err := os.Remove(filepath.Join(f.vault.Path, "Note.md")); err != nil {
		t.Fatal(err)
	}
	tells(t, watcher, "Note.md")
	next(t, f.heard)

	readers.release()
	eventually(t, "the scan did not finish", f.api.Ready.Load)

	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "" {
		t.Errorf("the index still holds %q", title)
	}
}

// TestANoteMadeWhileTheVaultIsBeingReadStays. The scan takes the vault as it
// was when it walked it, and what it does not hold in that list it leaves
// alone.
func TestANoteMadeWhileTheVaultIsBeingReadStays(t *testing.T) {
	watcher := byHand()
	readers := slowly()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, readers)

	// The walk is over and the first read is under way, so the vault the scan
	// is working from does not hold what is written now.
	next(t, readers.begun)

	write(t, f.vault, "Later.md", "---\ntitle: Later\n---\n\n# Later\n")
	tells(t, watcher, "Later.md")
	next(t, f.heard)

	readers.release()
	eventually(t, "the scan did not finish", f.api.Ready.Load)

	if title := titleOf(t, f.index, f.vault, "Later.md"); title != "Later" {
		t.Errorf("the index says %q of a note made while it was being read", title)
	}
}

// TestAVaultWhoseScanFailedIsStillFollowed. A vault that could not be read is
// still a vault being edited, and the window is told what changes in it.
func TestAVaultWhoseScanFailedIsStillFollowed(t *testing.T) {
	watcher := byHand()
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, unreadable{VaultReaders: filesystem.VaultReaders{}})

	eventually(t, "the scan was not reported as failed", func() bool {
		return f.api.Error.Why() != ""
	})

	write(t, f.vault, "Note.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	tells(t, watcher, "Note.md")

	if got := next(t, f.heard); len(got.paths) != 1 || got.paths[0] != "Note.md" {
		t.Errorf("the window was told %+v", got)
	}
}

// TestAVaultThatCannotBeWatchedSaysSo. The window draws a warning from this,
// and a vault nobody is following looks exactly like a vault nothing happens
// to.
func TestAVaultThatCannotBeWatchedSaysSo(t *testing.T) {
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, unwatchable{}, filesystem.VaultReaders{})

	if why := f.api.Unwatched.Why(); why == "" {
		t.Error("a vault whose watch never started is shown as followed")
	}
}

// TestAWatchThatStopsSaysSo.
func TestAWatchThatStopsSaysSo(t *testing.T) {
	watcher := byHand()
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, filesystem.VaultReaders{})

	close(watcher.changes)

	eventually(t, "a vault whose watch stopped is shown as followed", func() bool {
		return f.api.Unwatched.Why() != ""
	})
}

// runningBehind publishes the passes a request is answered through, the way a
// vault arriving in the window does.
func runningBehind(api *API, change func(*passes)) {
	on := passes{}
	if held := api.showing.Load(); held != nil {
		on = *held
	}
	change(&on)
	api.runs(&on)
}

// saying is where the reading behind the window writes what it did, kept for a
// test to read back. It is written from the pass and read from the test.
type saying struct {
	mu   sync.Mutex
	said strings.Builder
}

func (s *saying) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.said.Write(p)
}

// books is what each pass said it took apart, one number to a pass.
func (s *saying) books(t *testing.T) []int {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	var out []int
	for _, line := range strings.Split(s.said.String(), "\n") {
		_, said, found := strings.Cut(line, ": ")
		if !found {
			continue
		}
		count, _, found := strings.Cut(said, " books,")
		if !found {
			continue
		}
		n, err := strconv.Atoi(count)
		if err != nil {
			t.Fatalf("a pass said %q", line)
		}
		out = append(out, n)
	}
	return out
}

// TestReadingEveryFileAgainIsSpentOnOnePass. A launch asked to read every file
// again is asked it once. A flag left standing records every book of the vault
// as owing its text each time the watch nudges the reading, so a library of
// thousands is taken apart afresh for one file dropped into it.
func TestReadingEveryFileAgainIsSpentOnOnePass(t *testing.T) {
	const (
		held    = "library/A Book.epub"
		dropped = "library/Another Book.epub"
	)

	v := testsupport.NewVault(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	testsupport.WriteBook(t, v.Path, held)

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RebuildIndex: true,
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	readers := filesystem.VaultReaders{}
	watcher := byHand()
	api := &API{
		Listeners: following(),
		Places:    focusing(),
		Window:    &wire.Window{Named: wire.Editor, Tasking: task.New()},
		Notes:     Notes{Queries: db.Queries(), Links: db.Links()},
	}
	api.show(v)

	out := &saying{}
	ctx, stop := context.WithCancel(t.Context())
	wait := begin(ctx, v, cfg, db, api, cfg.VaultOpenerWith(db, readers, watcher),
		readers, nil, waking(settled), &pending{}, out)
	t.Cleanup(func() {
		stop()
		wait()
	})

	eventually(t, "the book the vault held was never taken apart", func() bool {
		return len(out.books(t)) == 1
	})

	// A second book is dropped into the vault and the watch says so, which is
	// the pass running again over a library it has already read.
	testsupport.WriteBook(t, v.Path, dropped)
	tells(t, watcher, dropped)

	eventually(t, "the book dropped into the vault was never taken apart", func() bool {
		return len(out.books(t)) == 2
	})

	if took := out.books(t); took[1] != 1 {
		t.Errorf("the second pass took %d books apart for the one file dropped in", took[1])
	}
}
