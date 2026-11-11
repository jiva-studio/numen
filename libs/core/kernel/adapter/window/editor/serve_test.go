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

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	adapterSettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	windowagent "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/agent"
	windowvault "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	agentuse "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/note/domain"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// noTabKinds gives no note a kind of tab.
type noTabKinds struct{}

func (noTabKinds) Execute(context.Context, domain.Vault, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func bindNotes(services *Services, db *testIndex) {
	refs, links := db.NoteQueries.NoteRefs, db.NoteQueries.Links
	services.NoteCases.OpeningNote = query.NewGetOpeningNote(db.NoteQueries.OpeningNote)
	services.NoteCases.Headings = query.NewListHeadings(db.NoteQueries.Headings)
	services.NoteCases.ResolveLinks = query.NewResolveLinks(links)
	services.NoteCases.Names = db.NoteQueries.NamedPaths
	services.NoteCases.Neighborhood = query.NewGetNeighborhoodView(links, refs, db.NoteQueries.NoteTypes, noTabKinds{})
	services.UseNotes()
}

// Vault watcher driven manually by the test, so reading a vault needs no filesystem or timer.
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

// waiting is a set of readers whose reads block until the test releases them,
// which simulates a vault that takes minutes to read.
type waiting struct {
	vaultport.VaultReaders
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

func (w *waiting) release() { w.once.Do(func() { close(w.until) }) }

func (w *waiting) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := w.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return waits{VaultReader: reader, posVal: w}, nil
}

// waits holds a read open after the bytes are read, so a test can change the
// file while the reader holds the earlier content.
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

// unreadable is a set of readers whose walk fails, which simulates a device that
// disappeared. A single unreadable file is counted and skipped, so only a walk
// failure exercises this path.
type unreadable struct{ vaultport.VaultReaders }

func (u unreadable) Open(v domain.Vault) (vaultport.VaultReader, error) {
	reader, err := u.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return failingWalkReader{VaultReader: reader}, nil
}

type failingWalkReader struct{ vaultport.VaultReader }

func (failingWalkReader) WalkFiles(context.Context, func(domain.FileStat) error) error {
	return errors.New("the vault cannot be read")
}

// behind is one vault with the background work of a window: the index it fills,
// the first scan, the watch, and the changes sent to a subscribed client.
type behind struct {
	api         *Services
	vault       domain.Vault
	index       *testIndex
	recordedVal <-chan window.Change
}

func opening(t *testing.T, notes map[string]string, watcher vaultport.VaultWatcher, readers vaultport.VaultReaders) *behind {
	t.Helper()
	return openingWith(t, notes, watcher, readers, nil, indexing.QuietPeriod)
}

func openingWith(
	t *testing.T,
	notes map[string]string,
	watcher vaultport.VaultWatcher,
	readers vaultport.VaultReaders,
	embedder embeddingport.Embedder,
	idleDuration time.Duration,
) *behind {
	t.Helper()

	v := vaultfixture.NewVault(t, notes)
	wired := newWiring(filepath.Join(t.TempDir(), "index.db"), false, nil)
	db := wired.openIndex(t)

	writing := note.NewWrite(
		vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, levelNothing, testsupport.NewClock(),
		note.OpenGate{},
	)
	read := note.NewRead(vaultreader.Readers{})
	api := createServices(NewState(run.NewTasks(), nil, nil))
	bindNotes(api, db)
	api.NoteCases.Read, api.NoteCases.Write = &read, &writing
	api.UseNotes()
	api.state.Runtime.SetVault(v)
	if embedder != nil {
		if err := db.VectorIndex().FitVectors(t.Context(), embedder.GetModel().Dimensions, embedder.GetModel().Recipe()); err != nil {
			t.Fatal(err)
		}
		api.state.Runtime.RecordModel(embedder.GetModel().String(), "")
		api.state.Indexing.Embedding = indexing.NewEmbedding(nil, nil, db.Chunks.Progress)
	}
	line, isDone := api.state.Notices.ChangeEvents.Subscribe()
	t.Cleanup(isDone)

	end := wired.startSession(t, t.Context(), db, api.state, v, readers, watcher, sessionOptions{
		embedding: indexing.NewEmbedding(embedder, nil, db.Chunks.Progress), idle: idleDuration, out: io.Discard,
	})
	t.Cleanup(end)

	return &behind{api: api, vault: v, index: db, recordedVal: line}
}

func write(t *testing.T, v domain.Vault, path, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Path, filepath.FromSlash(path)), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

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

func titleOf(t *testing.T, db *testIndex, v domain.Vault, path string) string {
	t.Helper()
	found, err := db.NoteQueries.NoteRefs.GetNotes(t.Context(), v.ID, []string{path})
	if err != nil {
		t.Fatal(err)
	}
	return found[path].Title
}

// TestAChangeArrivesWhileTheVaultIsStillBeingRead. A change arrives at the window
// only through the watch, and the vault is read for the first time while the
// user is already editing.
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
	if f.api.state.Runtime.IsReady() {
		t.Fatal("the scan finished with its reads held")
	}

	write(t, f.vault, "Note.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	sendChange(t, watcher, "Note.md")

	if got := next(t, f.recordedVal); len(got.Paths) != 1 || got.Paths[0] != "Note.md" {
		t.Errorf("the window received %+v", got)
	}
	if title := titleOf(t, f.index, f.vault, "Note.md"); title != "Renamed" {
		t.Errorf("the index says %q", title)
	}
	if f.api.state.Runtime.IsReady() {
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
	waitFor(t, "the scan did not finish", f.api.state.Runtime.IsReady)

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
	waitFor(t, "the scan did not finish", f.api.state.Runtime.IsReady)

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
	waitFor(t, "the scan did not finish", f.api.state.Runtime.IsReady)

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
		return f.api.state.Runtime.GetError() != ""
	})

	write(t, f.vault, "Note.md", "---\ntitle: Renamed\n---\n\n# Renamed\n")
	sendChange(t, watcher, "Note.md")

	if got := next(t, f.recordedVal); len(got.Paths) != 1 || got.Paths[0] != "Note.md" {
		t.Errorf("the window received %+v", got)
	}
}

// TestAVaultThatCannotBeWatchedSaysSo. The window shows a warning from this
// error, because an unwatched vault is indistinguishable from an idle one.
func TestAVaultThatCannotBeWatchedReturnsSo(t *testing.T) {
	f := opening(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	}, unwatchable{}, vaultreader.Readers{})

	if why := f.api.state.Runtime.GetWatchError(); why == "" {
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
		return f.api.state.Runtime.GetWatchError() != ""
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

func (s *saying) getBooks(t *testing.T) []int {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	var out []int
	for line := range strings.SplitSeq(s.outMsg.String(), "\n") {
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

// TestReadingEveryFileAgainIsSpentOnOnePass. A launch that requests a full re-read
// applies it once. A flag left set marks every book in the vault for text
// extraction on each watch notification, so a library of thousands is
// reprocessed for one added file.
func TestReadingEveryFileAgainIsSpentOnOnePass(t *testing.T) {
	const (
		target  = "library/A Book.epub"
		dropped = "library/Another Book.epub"
	)

	v := vaultfixture.NewVault(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	vaultfixture.WriteBook(t, v.Path, target)

	wired := newWiring(filepath.Join(t.TempDir(), "index.db"), true, testsupport.Texts{Readers: vaultreader.Readers{}})
	db := wired.openIndex(t)

	readers := vaultreader.Readers{}
	watcher := newManualWatcher()
	api := createServices(NewState(run.NewTasks(), nil, nil))
	bindNotes(api, db)
	api.state.Runtime.SetVault(v)

	out := &saying{}
	end := wired.startSession(t, t.Context(), db, api.state, v, readers, watcher, sessionOptions{
		idle: indexing.QuietPeriod, rebuild: true, out: out,
	})
	t.Cleanup(end)

	waitFor(t, "the book the vault held was never taken apart", func() bool {
		return len(out.getBooks(t)) == 1
	})

	// A second book is added to the vault and the watch reports it, which runs
	// another pass over an already-read library.
	vaultfixture.WriteBook(t, v.Path, dropped)
	sendChange(t, watcher, dropped)

	waitFor(t, "the book dropped into the vault was never taken apart", func() bool {
		return len(out.getBooks(t)) == 2
	})

	if took := out.getBooks(t); took[1] != 1 {
		t.Errorf("the second pass took %d books apart for the one file dropped in", took[1])
	}
}

func newBareServices() *Services { return createServices(NewState(run.NewTasks(), nil, nil)) }

func newThemedServices(themes numenv1connect.ThemeServiceHandler) *Services {
	services := newBareServices()
	services.Themes = themes
	return services
}

// TestAStreamWhoseClientWentAwayEnds. The window roots every request at the
// process, so a handler is not notified when its page disconnects: the editor
// reloads on a vault swap and leaves one stream of each kind behind. A failed
// write is the only disconnect signal, so each stream writes.
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
		"tab links": func(ctx context.Context, http connect.HTTPClient, linksPath string) error {
			_, err := numenv1connect.NewWorkspaceServiceClient(http, linksPath).
				WatchTabLinks(ctx, connect.NewRequest(&v1.WatchTabLinksRequest{WindowId: "window-a"}))
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
			t.Parallel()
			state := NewState(run.NewTasks(), nil, nil)
			services := createServices(state)

			entered, returned := make(chan struct{}, 1), make(chan struct{}, 1)
			mux := http.NewServeMux()
			vault, vaults := numenv1connect.NewVaultServiceHandler(windowvault.NewHandler(state, state.Notices))
			mux.Handle(vault, wiretest.NewRootedHandler(vaults, entered, returned))
			standing, workspace := numenv1connect.NewWorkspaceServiceHandler(newWorkspace(t, state))
			mux.Handle(standing, wiretest.NewRootedHandler(workspace, entered, returned))
			filed, notes := numenv1connect.NewNoteServiceHandler(services.Note)
			mux.Handle(filed, wiretest.NewRootedHandler(notes, entered, returned))
			rendered, itself := numenv1connect.NewWindowServiceHandler(state.Window)
			mux.Handle(rendered, wiretest.NewRootedHandler(itself, entered, returned))
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

			// The page is gone: the editor reloaded, and nothing canceled the
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

func newThemeClient(t *testing.T, cfg themeConfig) numenv1connect.ThemeServiceClient {
	t.Helper()

	themes, err := cfg.themes(nil)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newThemedServices(themes).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	return numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
}

func newConfig(t *testing.T) themeConfig {
	t.Helper()
	return themeConfig{RegistryPath: filepath.Join(t.TempDir(), "vaults.json")}
}

func TestTheWindowAsksTheSameHandlerAboutItsThemes(t *testing.T) {
	client := newThemeClient(t, newConfig(t))

	answer, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetApplied() != adapterSettings.DefaultTheme {
		t.Errorf("applied %q", answer.Msg.GetApplied())
	}
	if len(answer.Msg.GetThemes()) == 0 {
		t.Error("no themes")
	}

	text, err := client.ReadTheme(t.Context(),
		connect.NewRequest(&v1.ReadThemeRequest{Name: answer.Msg.GetApplied()}))
	if err != nil {
		t.Fatal(err)
	}
	if text.Msg.GetCss() == "" {
		t.Error("the applied theme has no text")
	}
}

// The chosen theme is saved to the settings file, and the keys the user wrote
// there remain.
func TestAThemeChosenInTheWindowIsSavedIntoTheSettings(t *testing.T) {
	cfg := newConfig(t)
	file := filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json")
	if err := os.WriteFile(file, []byte(
		`{"indexing":{"embedding":{"indexing":{"service":{"key":"sk-the-persons-own"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	client := newThemeClient(t, cfg)
	writeTheme(t, cfg, "nord.css", ":root{--numen-surface:#2e3440}")
	selected, err := client.WriteAppearance(t.Context(),
		connect.NewRequest(&v1.WriteAppearanceRequest{Name: "mine:nord", Mode: v1.Mode_MODE_DARK}))
	if err != nil {
		t.Fatal(err)
	}
	if reason := selected.Msg.GetError(); reason != "" {
		t.Fatalf("failed: %s", reason)
	}

	response := adapterSettings.DefaultFile()
	if err := adapterSettings.NewFiles().OpenAt(file, &response); err != nil {
		t.Fatal(err)
	}
	if response.Appearance.Theme != "mine:nord" || response.Appearance.Mode != theme.SchemeDark {
		t.Errorf("the file says %+v", response.Appearance)
	}
	// The stored API key is preserved.
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
	if answer.Msg.GetApplied() != "mine:nord" || answer.Msg.GetMode() != v1.Mode_MODE_DARK {
		t.Errorf("applied %q, read as %v", answer.Msg.GetApplied(), answer.Msg.GetMode())
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

	response := adapterSettings.DefaultFile()
	if err := adapterSettings.NewFiles().OpenAt(file, &response); err != nil {
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
	if err := os.WriteFile(file, []byte(`{"app":{"appearance":{"theme":"preset:nord"}}}`), 0o600); err != nil {
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
	if reason := selected.Msg.GetError(); !strings.Contains(reason, "appearance.textScale") {
		t.Errorf("failed with %q", reason)
	}

	target, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(target) != `{"app":{"appearance":{"theme":"preset:nord"}}}` {
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

// A build put together without a catalog answers that it has none, and the
// rest of the window is as it was.
func TestAWindowWithNoCatalogueAnswersThatItHasNone(t *testing.T) {
	server := httptest.NewServer(newBareServices().NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	client := numenv1connect.NewThemeServiceClient(server.Client(), server.URL)
	if _, err := client.ListThemes(t.Context(), connect.NewRequest(&v1.ListThemesRequest{})); err == nil {
		t.Error("a build with no catalog listed themes")
	}
}

type testAgentRun struct {
	steps chan agentdomain.Step
}

func (r *testAgentRun) GetSteps() <-chan agentdomain.Step { return r.steps }
func (r *testAgentRun) Stop() error                       { return nil }

type testAgentProvider struct {
	agentport.Agent
	tasks chan agentdomain.Task
}

func (m *testAgentProvider) StartRun(ctx context.Context, task agentdomain.Task) (agentport.Run, error) {
	m.tasks <- task
	run := &testAgentRun{steps: make(chan agentdomain.Step, 2)}
	run.steps <- agentdomain.Step{Kind: domain.StepSay, Text: "hello from agent"}
	run.steps <- agentdomain.Step{Kind: domain.StepStop, Detail: "done"}
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
	state := NewState(run.NewTasks(), nil, nil)
	state.Runtime.SetVault(domain.Vault{ID: "v-garden", Path: "/vaults/Garden"})
	services := createServices(state)
	agent := &testAgentProvider{tasks: make(chan agentdomain.Task, 1)}
	services.Mount(RouteOf(state, &windowagent.Handler{
		Start:    agentuse.NewStartTask(agent),
		End:      agentuse.NewEndConversation(agent),
		GetRoot:  func() string { return state.GetCurrentVault().Path },
		GetVault: func() domain.VaultID { return state.GetCurrentVault().ID },
	}, windowagent.NewRoute))
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
	if _, err := io.WriteString(w, "page bytes"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestPluginRouteReachesThePluginServices(t *testing.T) {
	plugins := &routedPlugins{}
	services := newBareServices()
	services.PluginServices = plugins
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	resp := fetch(t, server.Client(), server.URL+"/plugin/org.example.reader/pages/2?path=a.pdf")
	body := resp.Body

	if resp.StatusCode != http.StatusOK || string(body) != "page bytes" {
		t.Fatalf("answered %d %q", resp.StatusCode, body)
	}
	if !slices.Equal(plugins.routed, []string{"/plugin/org.example.reader/pages/2?path=a.pdf"}) {
		t.Errorf("forwarded %q", plugins.routed)
	}
}

func TestPluginRouteNamesTheVaultTheWindowShows(t *testing.T) {
	plugins := &routedPlugins{}
	state := NewState(run.NewTasks(), nil, nil)
	state.Runtime.SetVault(domain.Vault{ID: "v-garden", Path: "/vaults/Garden"})
	services := createServices(state)
	services.PluginServices = plugins
	server := httptest.NewServer(services.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)

	fetch(t, server.Client(), server.URL+"/plugin/org.example.reader/pages/2?path=a.sample")

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

	resp := fetch(t, server.Client(), server.URL+"/plugin/org.example.reader/pages/2")

	if resp.StatusCode != http.StatusNotFound || len(plugins.routed) != 0 {
		t.Errorf("answered %d, forwarded %q", resp.StatusCode, plugins.routed)
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
		resp := fetch(t, server.Client(), server.URL+path)
		body := resp.Body
		if resp.StatusCode != http.StatusOK || string(body) != want {
			t.Errorf("%s answered %d %q, want %q", path, resp.StatusCode, body, want)
		}
	}
	if !slices.Equal(plugins.routed, []string{"/plugin/org.example.reader/pages/2"}) {
		t.Errorf("forwarded %q", plugins.routed)
	}
}
