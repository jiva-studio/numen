package editor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	adapterSettings "github.com/jiva-studio/numen/libs/core/adapter/settings"
	editorSearch "github.com/jiva-studio/numen/libs/core/adapter/window/editor/search"
	editorSettings "github.com/jiva-studio/numen/libs/core/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	coreSettings "github.com/jiva-studio/numen/libs/core/usecase/settings"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// bindNotes gives the handlers the questions a window asks of the index in db
// about the notes.
func bindNotes(services *Services, db *container.Index) {
	queries, links := db.Queries(), db.Links()
	neighbourhood := note.NewShowNeighbourhood(links, queries)
	services.Note.getOpeningNote = note.NewGetOpeningNote(queries)
	services.Note.getTypes = note.NewGetTypes(queries)
	services.Note.getHeadings = note.NewGetHeadings(queries)
	services.Note.resolveLinks = note.NewResolveLinks(links)
	services.Note.showNeighbourhood = &neighbourhood
	services.Note.getLinkAddress = note.NewGetLinkAddress(queries)
	services.Search.GetNameMatches = search.NewGetNameMatches(queries)
	services.Search.GetTitles = note.NewGetTitles(queries)
	services.Search.GetTypes = services.Note.getTypes
}

// is being read can be asked without a filesystem or a timer.
type hand struct {
	changes  chan []string
	lostChan chan struct{}
}

func newManualWatcher() *hand {
	return &hand{changes: make(chan []string), lostChan: make(chan struct{}, 1)}
}

func (h *hand) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return h.changes, h.lostChan, nil
}

// unwatchable is a watch that will not start, which is what an operating system
// out of watches looks like from here.
type unwatchable struct{}

func (unwatchable) WatchVault(context.Context, domain.Vault) (<-chan []string, <-chan struct{}, error) {
	return nil, nil, errors.New("no watches left")
}

// waiting is a set of readers whose reads stop where a test can see them and
// finish when it says so, which is what a vault that takes minutes to read
// looks like from here.
type waiting struct {
	vaultport.VaultReaders
	// begun carries the path of a read that is under way.
	begun chan string
	until chan struct{}
	once  sync.Once
}

func newSlowWait() *waiting {
	return &waiting{
		VaultReaders: vaultreader.Readers{},
		begun:        make(chan string, 1),
		until:        make(chan struct{}),
	}
}

// release lets every read through, from now on.
func (w *waiting) release() { w.once.Do(func() { close(w.until) }) }

func (w *waiting) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := w.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return waits{VaultReader: reader, posVal: w}, nil
}

// waits holds a read open once the bytes are in hand, so a test can change the
// file the reader is holding a copy of.
type waits struct {
	vaultport.VaultReader
	posVal *waiting
}

func (r waits) Read(ctx context.Context, path string) ([]byte, error) {
	raw, err := r.VaultReader.Read(ctx, path)
	select {
	case r.posVal.begun <- path:
	default:
	}
	select {
	case <-r.posVal.until:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return raw, err
}

// unwalkable is a set of readers that cannot list the vault, which is what a
// folder the walk fails looks like from here. A note named outright is
// still read.
type unwalkable struct{ vaultport.VaultReaders }

func (u unwalkable) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := u.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return unlisted{VaultReader: reader}, nil
}

type unlisted struct{ vaultport.VaultReader }

func (unlisted) WalkFiles(context.Context, func(domain.FileStat) error) error {
	return errors.New("the vault cannot be listed")
}

// unreadable is a set of readers whose walk fails, which is what a device that
// went away looks like from here. A single file nobody can read is counted and
// walked past, so it is the walk itself that has to go.
type unreadable struct{ vaultport.VaultReaders }

func (u unreadable) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := u.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return refuses{VaultReader: reader}, nil
}

type refuses struct{ vaultport.VaultReader }

func (refuses) WalkFiles(context.Context, func(domain.FileStat) error) error {
	return errors.New("the vault cannot be read")
}

// behind is one vault with the work a window has running behind it: the index
// it fills, the first scan, the watch beside it, and what a client listening
// for changes is told.
type behind struct {
	api         *Services
	vault       domain.Vault
	index       *container.Index
	recordedVal <-chan change
}

// opening puts a vault, an index, a scan and a watch together the way Open
// does, with the pieces a test drives in place of the ones a window gets.
func opening(t *testing.T, notes map[string]string, watcher vaultport.VaultWatcher, readers vaultport.VaultReaders) *behind {
	t.Helper()
	return openingWith(t, notes, watcher, readers, nil, quietPeriod)
}

// openingWith is the same, with the model that fills the index for meaning and
// how long the vault has to have been still before a note is embedded.
func openingWith(
	t *testing.T,
	notes map[string]string,
	watcher vaultport.VaultWatcher,
	readers vaultport.VaultReaders,
	embedder embeddingport.Embedder,
	idleDuration time.Duration,
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
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, levelNothing, time.Now)
	read := note.NewRead(vaultreader.Readers{})
	api := createServices(&State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
	})
	bindNotes(api, db)
	api.Note.read, api.Note.write = &read, &writing
	api.state.setVault(v)
	if embedder != nil {
		if err := db.FitVectors(t.Context(), embedder.GetModel().Dimensions, embedder.GetModel().Recipe()); err != nil {
			t.Fatal(err)
		}
		api.state.Indexing.Model.Store(embedder.GetModel().String())
		api.state.Indexing.Progress = db.Progress()
	}
	line, isDone := api.state.ChangeEvents.subscribe()
	t.Cleanup(isDone)

	made := newWatched(cfg, db, readers, watcher)
	worker := newIndexWorker(made, api.state, embedder, idleDuration, io.Discard)
	api.state.NotifyNoteWritten = worker.WakeNotes

	ctx, stop := context.WithCancel(t.Context())
	_, wait := startIndexing(ctx, v, made, api.state, false, worker, worker.NewSession(), io.Discard)
	t.Cleanup(func() {
		stop()
		wait()
	})

	return &behind{api: api, vault: v, index: db, recordedVal: line}
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
	out, err := f.api.Note.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{Path: path, Body: body}))
	if err != nil {
		t.Fatal(err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the save of %s failed: %v", path, code)
	}
}

// sendChange hands the watch a change, which takes somebody acting on the
// watch.
func sendChange(t *testing.T, watcher *hand, paths ...string) {
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

// waitFor waits for something the work behind the window does in its own
// time.
func waitFor(t *testing.T, what string, is func() bool) {
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
	found, err := db.Queries().GetNotes(t.Context(), v.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	return found[path].Title
}

// TestAChangeArrivesWhileTheVaultIsStillBeingRead. A change reaches the window
// through the watch and through nothing else, and the vault is read for the
// first time with the person already working in it.
func TestAChangeArrivesWhileTheVaultIsStillBeingRead(t *testing.T) {
	watcher := newManualWatcher()
	readers := newSlowWait()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, readers)

	// A read is under way and held, so the vault has not been read yet.
	if path := next(t, readers.begun); path != "Note.md" {
		t.Fatalf("the scan is reading %q", path)
	}
	if f.api.state.Ready.Load() {
		t.Fatal("the scan finished with its reads held")
	}

	write(t, f.vault, "Note.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	sendChange(t, watcher, "Note.md")

	if got := next(t, f.recordedVal); len(got.paths) != 1 || got.paths[0] != "Note.md" {
		t.Errorf("the window received %+v", got)
	}
	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "Renamed" {
		t.Errorf("the index says %q", title)
	}
	if f.api.state.Ready.Load() {
		t.Error("the scan finished with its reads held")
	}
}

// TestANoteChangedUnderTheScanIsReadAgain. The scan writes in groups from what
// it read, so its copy of a note lands last however early the note was read.
func TestANoteChangedBelowTheScanIsReadAgain(t *testing.T) {
	watcher := newManualWatcher()
	readers := newSlowWait()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: One\n---\n\n# One\n",
	}, watcher, readers)

	// The scan holds the note as it was.
	next(t, readers.begun)

	write(t, f.vault, "Note.md", "---\ntitle: Two\n---\n\n# Two\n")
	sendChange(t, watcher, "Note.md")
	next(t, f.recordedVal)

	// Now the scan writes what it read, and finishes.
	readers.release()
	waitFor(t, "the scan did not finish", f.api.state.Ready.Load)

	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "Two" {
		t.Errorf("the index says %q", title)
	}
}

// TestANoteDeletedUnderTheScanStaysDeleted. The scan holds a copy of a note it
// read, and the note went while it held it.
func TestANoteDeletedBelowTheScanStaysDeleted(t *testing.T) {
	watcher := newManualWatcher()
	readers := newSlowWait()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, readers)

	next(t, readers.begun)

	if err := os.Remove(filepath.Join(f.vault.Path, "Note.md")); err != nil {
		t.Fatal(err)
	}
	sendChange(t, watcher, "Note.md")
	next(t, f.recordedVal)

	readers.release()
	waitFor(t, "the scan did not finish", f.api.state.Ready.Load)

	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "" {
		t.Errorf("the index still holds %q", title)
	}
}

// TestANoteMadeWhileTheVaultIsBeingReadStays. The scan takes the vault as it
// was when it walked it, and what it does not hold in that list it leaves
// alone.
func TestANoteMadeWhileTheVaultIsBeingReadStays(t *testing.T) {
	watcher := newManualWatcher()
	readers := newSlowWait()
	defer readers.release()

	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, readers)

	// The walk is over and the first read is under way, so the vault the scan
	// is working from does not hold what is written now.
	next(t, readers.begun)

	write(t, f.vault, "Later.md", "---\ntitle: Later\n---\n\n# Later\n")
	sendChange(t, watcher, "Later.md")
	next(t, f.recordedVal)

	readers.release()
	waitFor(t, "the scan did not finish", f.api.state.Ready.Load)

	if title := titleOf(t, f.index, f.vault, "Later.md"); title != "Later" {
		t.Errorf("the index says %q of a note made while it was being read", title)
	}
}

// TestAVaultWhoseScanFailedIsStillFollowed. A vault that could not be read is
func TestAVaultWhoseScanFailedIsStillFollowed(t *testing.T) {
	watcher := newManualWatcher()
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, unreadable{VaultReaders: vaultreader.Readers{}})

	waitFor(t, "the scan was not reported as failed", func() bool {
		return f.api.state.Error.Why() != ""
	})

	write(t, f.vault, "Note.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	sendChange(t, watcher, "Note.md")

	if got := next(t, f.recordedVal); len(got.paths) != 1 || got.paths[0] != "Note.md" {
		t.Errorf("the window received %+v", got)
	}
}

// TestAVaultThatCannotBeWatchedSaysSo. The window draws a warning from this,
// and a vault nobody is following looks exactly like a vault nothing happens
// to.
func TestAVaultThatCannotBeWatchedReturnsSo(t *testing.T) {
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, unwatchable{}, vaultreader.Readers{})

	if why := f.api.state.WatchError.Why(); why == "" {
		t.Error("a vault whose watch never started is shown as followed")
	}
}

// TestAWatchThatStopsSaysSo.
func TestAWatchThatStopsReturnsSo(t *testing.T) {
	watcher := newManualWatcher()
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, watcher, vaultreader.Readers{})

	close(watcher.changes)

	waitFor(t, "a vault whose watch stopped is shown as followed", func() bool {
		return f.api.state.WatchError.Why() != ""
	})
}

// saying is where the reading behind the window writes what it did, kept for a
// test to read back. It is written from the pass and read from the test.
type saying struct {
	mu     sync.Mutex
	outMsg strings.Builder
}

func (s *saying) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.outMsg.Write(p)
}

// books is what each pass said it took apart, one number to a pass.
func (s *saying) getBooks(t *testing.T) []int {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	var out []int
	for _, line := range strings.Split(s.outMsg.String(), "\n") {
		_, response, found := strings.Cut(line, ": ")
		if !found {
			continue
		}
		count, _, found := strings.Cut(response, " file texts,")
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
		target  = "library/A Book.epub"
		dropped = "library/Another Book.epub"
	)

	v := testsupport.NewVault(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	testsupport.WriteBook(t, v.Path, target)

	cfg := container.Config{
		IndexPath:          filepath.Join(t.TempDir(), "index.db"),
		ShouldRebuildIndex: true,
		TextReaders:        testsupport.Texts{Readers: vaultreader.Readers{}},
	}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	readers := vaultreader.Readers{}
	watcher := newManualWatcher()
	api := createServices(&State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
		Window:       &wire.Window{Named: wire.Editor, Tasking: run.NewTasks()},
	})
	bindNotes(api, db)
	api.state.setVault(v)

	out := &saying{}
	made := newWatched(cfg, db, readers, watcher)
	worker := newIndexWorker(made, api.state, nil, quietPeriod, out)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	_, wait := startIndexing(ctx, v, made, api.state, true, worker, worker.NewSession(), out)
	t.Cleanup(func() {
		stop()
		wait()
	})

	waitFor(t, "the book the vault held was never taken apart", func() bool {
		return len(out.getBooks(t)) == 1
	})

	// A second book is dropped into the vault and the watch says so, which is
	// the pass running again over a library it has already read.
	testsupport.WriteBook(t, v.Path, dropped)
	sendChange(t, watcher, dropped)

	waitFor(t, "the book dropped into the vault was never taken apart", func() bool {
		return len(out.getBooks(t)) == 2
	})

	if took := out.getBooks(t); took[1] != 1 {
		t.Errorf("the second pass took %d books apart for the one file dropped in", took[1])
	}
}

// newBareServices is the handlers of a window standing on nothing, with no use
// case bound.
func newBareServices() *Services { return createServices(&State{}) }

// newThemedServices is that window, dressed in these themes.
func newThemedServices(themes numenv1connect.ThemeServiceHandler) *Services {
	services := newBareServices()
	services.Themes = themes
	return services
}

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so a handler is never told the page it was opened from is gone: the
// editor reloads on a vault swap and leaves one of each stream behind. A write
// that fails is the one report there is, so each stream makes one.
func TestAStreamWhoseClientWentAwayEnds(t *testing.T) {
	streams := map[string]func(context.Context, connect.HTTPClient, string) error{
		"changes": func(ctx context.Context, http connect.HTTPClient, pos string) error {
			_, err := numenv1connect.NewVaultServiceClient(http, pos).
				WatchVaultChanges(ctx, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
			return err
		},
		"editing": func(ctx context.Context, http connect.HTTPClient, offset string) error {
			_, err := numenv1connect.NewNoteServiceClient(http, offset).
				WatchEdits(ctx, connect.NewRequest(&v1.WatchEditsRequest{}))
			return err
		},
		"focus": func(ctx context.Context, http connect.HTTPClient, index string) error {
			_, err := numenv1connect.NewWorkspaceServiceClient(http, index).
				WatchFocus(ctx, connect.NewRequest(&v1.WatchFocusRequest{}))
			return err
		},
		"quitting": func(ctx context.Context, http connect.HTTPClient, targetPath string) error {
			_, err := numenv1connect.NewWindowServiceClient(http, targetPath).
				WatchQuit(ctx, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
			return err
		},
		"tasks": func(ctx context.Context, http connect.HTTPClient, targetOffset string) error {
			_, err := numenv1connect.NewWindowServiceClient(http, targetOffset).
				WatchTasks(ctx, connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Editor}))
			return err
		},
	}

	for name, open := range streams {
		t.Run(name, func(t *testing.T) {
			state := &State{Window: &wire.Window{Named: wire.Editor, Tasking: run.NewTasks()}}
			services := createServices(state)

			entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
			mux := http.NewServeMux()
			vault, vaults := numenv1connect.NewVaultServiceHandler(services.Vault)
			mux.Handle(vault, testsupport.NewRootedHandler(vaults, entered, returned))
			standing, workspace := numenv1connect.NewWorkspaceServiceHandler(services.Workspace)
			mux.Handle(standing, testsupport.NewRootedHandler(workspace, entered, returned))
			filed, notes := numenv1connect.NewNoteServiceHandler(services.Note)
			mux.Handle(filed, testsupport.NewRootedHandler(notes, entered, returned))
			rendered, itself := numenv1connect.NewWindowServiceHandler(state.Window)
			mux.Handle(rendered, testsupport.NewRootedHandler(itself, entered, returned))
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			if err := open(t.Context(), server.Client(), server.URL); err != nil {
				t.Fatal(err)
			}

			select {
			case <-entered:
			case <-time.After(testsupport.Patience):
				t.Fatal("the stream never reached the handler")
			}

			// The page is gone: the editor reloaded, and nothing cancelled the
			// context the handler holds.
			server.CloseClientConnections()

			select {
			case <-returned:
			case <-time.After(testsupport.Patience):
				t.Fatal("the client went away and the stream is still standing")
			}
		})
	}
}

func TestEverySearchModeIsRead(t *testing.T) {
	testsupport.CheckHandled(t, func(mode v1.SearchMode) bool {
		_, named := editorSearch.ConvertSearchModeFromWire(mode)
		return named
	})
}

func TestEveryRoleIsRead(t *testing.T) {
	testsupport.CheckHandled(t, func(role v1.Role) bool {
		_, named := convertRoleFromWire(role)
		return named
	})
}

func TestEveryRelationIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Relation]domain.Relation{
		v1.Relation_RELATION_PARENT:  domain.RelationParent,
		v1.Relation_RELATION_CHILD:   domain.RelationChild,
		v1.Relation_RELATION_JUMP:    domain.RelationJump,
		v1.Relation_RELATION_SIBLING: domain.RelationSibling,
	}, convertRelationToWire)
}

func TestANoteTypeIsSavedAsTheWordItContainsAndAnOrdinaryNoteAsNone(t *testing.T) {
	for typed, want := range map[domain.NoteType]string{
		domain.TypeNote: "",
		"sample":        "sample",
		"sheet":         "sheet",
	} {
		if got := convertNoteTypeToWire(typed); got != want {
			t.Errorf("%q is written as %q, want %q", typed, got, want)
		}
	}
}

func TestEveryNamingIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.NameSource]note.NameSource{
		v1.NameSource_NAME_SOURCE_FRONTMATTER: note.ByFrontmatter,
		v1.NameSource_NAME_SOURCE_FILENAME:    note.ByFilename,
	}, newWireNameSource)
}

func TestEveryPresenceIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Presence]settingsport.Presence{
		v1.Presence_PRESENCE_PRESENT:          settingsport.Present,
		v1.Presence_PRESENCE_NOT_FETCHED:      settingsport.NotFetched,
		v1.Presence_PRESENCE_NOTHING_TO_FETCH: settingsport.NothingToFetch,
	}, func(is settingsport.Presence) v1.Presence { return editorSettings.Presences[is] })
}

func TestEveryCompletionConsentIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CompletionConsent]coreSettings.Consent{
		v1.CompletionConsent_COMPLETION_CONSENT_UNANSWERED: coreSettings.ConsentUnanswered,
		v1.CompletionConsent_COMPLETION_CONSENT_ASKED:      coreSettings.ConsentAsked,
		v1.CompletionConsent_COMPLETION_CONSENT_ALLOWED:    coreSettings.ConsentAllowed,
		v1.CompletionConsent_COMPLETION_CONSENT_REFUSED:    coreSettings.ConsentRefused,
	}, func(is coreSettings.Consent) v1.CompletionConsent { return editorSettings.Consents[is] })
}

func TestEveryVaultsErrorCodeIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.VaultsErrorCode]error{
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE: vaults.ErrUnreadable,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY:       vaults.ErrCopy,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS:   domain.ErrOverlaps,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN: vaults.ErrNameTaken,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT: vaults.ErrLastVault,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING:    vaults.ErrShowing,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN:    vaults.ErrUnknown,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH:   machineport.ErrNoTrash,
		v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING:     errAnswerPending,
	}, func(err error) v1.VaultsErrorCode {
		reason, _ := vaultsErrorCodeBy(err)
		return reason
	})
}

// newThemeClient is a window's themes as a client reaches them: through the handler
// that answers everything else the window asks.
func newThemeClient(t *testing.T, cfg container.Config) numenv1connect.ThemeServiceClient {
	t.Helper()

	themes, err := cfg.Themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newThemedServices(themes).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	return numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
}

// newConfig is a configuration whose files are all in one folder of a test's
// own, which is what pointing the registry somewhere does.
func newConfig(t *testing.T) container.Config {
	t.Helper()
	return container.Config{RegistryPath: filepath.Join(t.TempDir(), "vaults.json")}
}

func TestTheWindowAsksTheSameHandlerAboutItsThemes(t *testing.T) {
	client := newThemeClient(t, newConfig(t))

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != adapterSettings.DefaultTheme {
		t.Errorf("wears %q", answer.Msg.GetApplied())
	}
	if len(answer.Msg.GetThemes()) == 0 {
		t.Error("nothing to wear")
	}

	text, err := client.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: answer.Msg.GetApplied()}))
	if err != nil {
		t.Fatal(err)
	}
	if text.Msg.GetCss() == "" {
		t.Error("the theme it wears has no text")
	}
}

// The choice lands in the settings file, and the keys a person typed into it
// are still there afterwards.
func TestAThemeChosenInTheWindowIsSavedIntoTheSettings(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(
		`{"indexing":{"embedding":{"indexing":{"service":{"key":"sk-the-persons-own"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	client := newThemeClient(t, cfg)
	selected, err := client.WriteAppearance(t.Context(),
		connect.NewRequest(&v1.WriteAppearanceRequest{Name: "preset:nord", Mode: v1.Mode_MODE_DARK}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := selected.Msg.GetError(); reason != "" {
		t.Fatalf("failed: %s", reason)
	}

	response := container.DefaultSettings()
	if err := adapterSettings.OpenAt(file, &response); err != nil {
		t.Fatal(err)
	}
	if response.Appearance.Theme != "preset:nord" || response.Appearance.Mode != adapterSettings.ModeDark {
		t.Errorf("the file says %+v", response.Appearance)
	}
	// through a provider. What it is still doing there is what the file says.
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "sk-the-persons-own") {
		t.Errorf("the file came back as %s", raw)
	}

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != "preset:nord" || answer.Msg.GetMode() != v1.Mode_MODE_DARK {
		t.Errorf("wears %q, read as %v", answer.Msg.GetApplied(), answer.Msg.GetMode())
	}
}

// The two sizes land in the settings file, and each is written on its own.
func TestASizeChosenInTheWindowIsSavedIntoTheSettings(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	client := newThemeClient(t, cfg)

	zoomLevel := 1.5
	selected, err := client.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:           adapterSettings.DefaultTheme,
		Mode:           v1.Mode_MODE_LIGHT,
		InterfaceScale: &zoomLevel,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := selected.Msg.GetError(); reason != "" {
		t.Fatalf("failed: %s", reason)
	}

	response := container.DefaultSettings()
	if err := adapterSettings.OpenAt(file, &response); err != nil {
		t.Fatal(err)
	}
	if response.Appearance.InterfaceScale != 1.5 || response.Appearance.TextScale != 1 {
		t.Errorf("the file says %+v", response.Appearance)
	}

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetInterfaceScale() != 1.5 || answer.Msg.GetTextScale() != 1 {
		t.Errorf("drawn at %v and set at %v", answer.Msg.GetInterfaceScale(), answer.Msg.GetTextScale())
	}
}

// A window can be made hard to read only as far as the bounds go, and the file
// is left currently.
func TestASizeOutsideWhatItGoesToIsRefusedAndNothingIsSaved(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(`{"appearance":{"theme":"preset:nord"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	client := newThemeClient(t, cfg)
	set := 4.0
	selected, err := client.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:      adapterSettings.DefaultTheme,
		Mode:      v1.Mode_MODE_LIGHT,
		TextScale: &set,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := selected.Msg.GetError(); !strings.Contains(reason, "appearance.text_scale") {
		t.Errorf("failed with %q", reason)
	}

	target, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(target) != `{"appearance":{"theme":"preset:nord"}}` {
		t.Errorf("the file now reads %s", target)
	}
}

// size for themselves is what it is release for.
func TestListThemes_LaunchScaleAppliesUntilUserSetsScale(t *testing.T) {
	cfg := newConfig(t)
	cfg.InterfaceScale = 1.25
	client := newThemeClient(t, cfg)

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetInterfaceScale() != 1.25 {
		t.Errorf("drawn at %v", answer.Msg.GetInterfaceScale())
	}

	zoomLevel := 1.5
	if _, err := client.WriteAppearance(t.Context(), connect.NewRequest(&v1.WriteAppearanceRequest{
		Name:           adapterSettings.DefaultTheme,
		Mode:           v1.Mode_MODE_LIGHT,
		InterfaceScale: &zoomLevel,
	})); err != nil {
		t.Fatal(err)
	}

	answer, err = client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetInterfaceScale() != 1.5 {
		t.Errorf("drawn at %v", answer.Msg.GetInterfaceScale())
	}
}

// A build put together without a catalogue answers that it has none, and the
// rest of the window is as it was.
func TestAWindowWithNoCatalogueAnswersThatItHasNone(t *testing.T) {
	server := httptest.NewServer(newBareServices().NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{})); err == nil {
		t.Error("a build with no catalogue listed themes")
	}
}

type testAgentRun struct {
	steps chan agentport.Step
}

func (r *testAgentRun) GetSteps() <-chan agentport.Step { return r.steps }
func (r *testAgentRun) Stop() error                     { return nil }

type testAgentProvider struct {
	agentport.Agent
	tasks chan agentport.Task
}

func (m *testAgentProvider) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	m.tasks <- task
	run := &testAgentRun{steps: make(chan agentport.Step, 2)}
	run.steps <- agentport.Step{Kind: domain.StepSay, Text: "hello from agent"}
	run.steps <- agentport.Step{Kind: domain.StepStop, Detail: "done"}
	close(run.steps)
	return run, nil
}

func (m *testAgentProvider) EndConversation(ctx context.Context, _ string) error {
	return nil
}

type testInterceptingPlugins struct {
	servedPaths []string
}

func (p *testInterceptingPlugins) ListFrameOrigins() []string { return nil }

func (p *testInterceptingPlugins) GetFrameOrigins(string) []string { return nil }

func (p *testInterceptingPlugins) Serves(_ string) bool {
	return true
}

func (p *testInterceptingPlugins) ServeRoute(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "plugin not found", http.StatusNotFound)
}

func (p *testInterceptingPlugins) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.servedPaths = append(p.servedPaths, r.URL.Path)
	http.Error(w, "should not be called for AgentService", http.StatusInternalServerError)
}

func TestAskAgentRoutesToAgentAndBypassesPluginServices(t *testing.T) {
	state := &State{Window: &wire.Window{Named: wire.Editor, Tasking: run.NewTasks()}}
	state.setVault(domain.Vault{ID: "v-garden", Path: "/vaults/Garden"})
	services := createServices(state)
	agent := &testAgentProvider{tasks: make(chan agentport.Task, 1)}
	services.Agent.GetAgent = func() agentport.Agent { return agent }
	plugins := &testInterceptingPlugins{}
	services.PluginServices = plugins

	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	client := numenv1connect.NewAgentServiceClient(server.Client(), server.URL)
	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{
		Prompt: "test question",
		Agent:  "claude",
	}))
	if err != nil {
		t.Fatal(err)
	}

	var gotText string
	for stream.Receive() {
		if text := stream.Msg().GetText(); text != "" {
			gotText += text
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if gotText != "hello from agent" {
		t.Errorf("got %q, want %q", gotText, "hello from agent")
	}
	if len(plugins.servedPaths) != 0 {
		t.Errorf("plugin services was erroneously called: %v", plugins.servedPaths)
	}
	select {
	case task := <-agent.tasks:
		if task.Agent != "claude" || task.Question != "test question" || task.Vault != "v-garden" || task.Root != "/vaults/Garden" {
			t.Errorf("unexpected task: %+v", task)
		}
	default:
		t.Fatal("agent received no task")
	}
}

// routedPlugins records the requests forwarded on the plugin route.
type routedPlugins struct {
	routed []string
	frames map[string][]string
	vaults []string
}

func (p *routedPlugins) ListFrameOrigins() []string {
	var out []string
	for _, origins := range p.frames {
		out = append(out, origins...)
	}
	return out
}

func (p *routedPlugins) GetFrameOrigins(pluginID string) []string { return p.frames[pluginID] }

func (p *routedPlugins) Serves(string) bool { return false }

func (p *routedPlugins) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "no plugin serves this service", http.StatusNotFound)
}

func (p *routedPlugins) ServeRoute(w http.ResponseWriter, r *http.Request) {
	p.routed = append(p.routed, r.URL.RequestURI())
	p.vaults = append(p.vaults, r.Header.Get("Numen-Vault"))
	w.Header().Set("Content-Type", "image/jpeg")
	_, _ = io.WriteString(w, "page bytes")
}

func TestPluginRouteReachesThePluginServices(t *testing.T) {
	plugins := &routedPlugins{}
	services := newBareServices()
	services.PluginServices = plugins
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/plugin/org.example.reader/pages/2?path=a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || string(body) != "page bytes" {
		t.Fatalf("answered %d %q", resp.StatusCode, body)
	}
	if !slices.Equal(plugins.routed, []string{"/plugin/org.example.reader/pages/2?path=a.pdf"}) {
		t.Errorf("forwarded %q", plugins.routed)
	}
}

func TestPluginRouteNamesTheVaultTheWindowShows(t *testing.T) {
	plugins := &routedPlugins{}
	state := &State{Window: &wire.Window{Named: wire.Editor, Tasking: run.NewTasks()}}
	state.setVault(domain.Vault{ID: "v-garden", Path: "/vaults/Garden"})
	services := createServices(state)
	services.PluginServices = plugins
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/plugin/org.example.reader/pages/2?path=a.sample")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if !slices.Equal(plugins.vaults, []string{"v-garden"}) {
		t.Errorf("forwarded with the vaults %q, want v-garden", plugins.vaults)
	}
}

func TestPluginRouteIsNotServedByANarrowBuild(t *testing.T) {
	plugins := &routedPlugins{}
	services := newBareServices()
	services.PluginServices = plugins
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler(), numenv1connect.NoteServiceName))
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + "/plugin/org.example.reader/pages/2")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound || len(plugins.routed) != 0 {
		t.Errorf("answered %d, forwarded %q", resp.StatusCode, plugins.routed)
	}
}

func TestServeUpdateAsksTheEndpointTheServicesName(t *testing.T) {
	release := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"version":"9.9.9"}`)
	}))
	t.Cleanup(release.Close)

	services := newBareServices()
	services.UpdateEndpoint = release.URL + "/latest.json"
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	resp, err := server.Client().Get(server.URL + updateRoute)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != `{"version":"9.9.9"}` {
		t.Errorf("answered %d %q, want 200 and the manifest of the endpoint", resp.StatusCode, body)
	}
}

func TestPluginRouteServesAFolderFileAndForwardsTheRest(t *testing.T) {
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "manifest.json"), []byte(`{"id":"org.example.reader"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plugins := &routedPlugins{}
	services := newBareServices()
	services.PluginServices = plugins
	services.getPluginDir = func(id string) (string, bool) { return folder, id == "org.example.reader" }
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	for path, want := range map[string]string{
		"/plugin/org.example.reader/manifest.json": `{"id":"org.example.reader"}`,
		"/plugin/org.example.reader/pages/2":       "page bytes",
	} {
		resp, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != want {
			t.Errorf("%s answered %d %q, want %q", path, resp.StatusCode, body, want)
		}
	}
	if !slices.Equal(plugins.routed, []string{"/plugin/org.example.reader/pages/2"}) {
		t.Errorf("forwarded %q", plugins.routed)
	}
}
