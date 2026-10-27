package editor_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/settings"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// openVault is a vault with a window's worth of machinery behind it — the index,
// the first scan, the watcher — and a client talking to it the way the window
// does.
//
// What is asked here is the wire: that a change reaches a client over the
// stream, in the shape the schema describes. What a change means is asked of
// the use case, where no server is needed to ask it.
func openVault(t *testing.T, notes map[string]string) (questions, string) {
	t.Helper()
	client, _, root, _ := openVaultWithWindow(t, notes)
	return client, root
}

// openVaultWithWindow is that same vault, with the window's half of it as well, for a test
// asking what something the window does reaches the client as.
func openVaultWithWindow(t *testing.T, notes map[string]string) (
	questions, numenv1connect.WindowServiceClient, string, *editor.Installation,
) {
	t.Helper()
	root := t.TempDir()
	for name, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}

	settings := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
		TextReaders:  testsupport.Texts{Readers: vaultreader.Readers{}},
	}
	registry, err := settings.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaults.NewAdd(settings.VaultIdentity(), registry, time.Now).Execute(root, "watched"); err != nil {
		t.Fatal(err)
	}

	installation, err := editor.Open(t.Context(), editor.NewAssembly(t, settings), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { installation.Close() })

	rendered, itself := numenv1connect.NewWindowServiceHandler(installation.State.Window)
	mux := http.NewServeMux()
	mountQuestionsServices(mux, installation.Services)
	mux.Handle(rendered, itself)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	client := newQuestionsClient(server.Client(), server.URL)
	watching := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)

	// The window opens on what the first scan stored; the watcher reports only
	// what happens after it.
	for range 200 {
		state, err := client.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if state.Msg.GetScan().GetIsReady() {
			return client, watching, root, installation
		}
		if reason := state.Msg.GetScan().GetError(); reason != "" {
			t.Fatalf("the first scan failed: %s", reason)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the first scan did not finish")
	return questions{}, nil, "", nil
}

// TestAnEditReachesAListener is the whole path: a file on disk, the watcher,
// the index, and the stream a window listens to.
func TestAnEditReachesAListener(t *testing.T) {
	client, root := openVault(t, map[string]string{
		"Note.md":  "---\ntitle: Note\n---\n\n# Note\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n",
	})

	// Its own context, closed before the server is: a stream is an open request,
	// and a test server waits for those.
	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()

	// The first message says the stream is open and nothing has changed yet.
	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}
	if paths := changes.Msg().GetPaths(); len(paths) != 0 {
		t.Fatalf("the stream opened by reporting %v", paths)
	}

	reported := make(chan []string, 1)
	go func() {
		for changes.Receive() {
			if paths := changes.Msg().GetPaths(); len(paths) > 0 {
				reported <- paths
				return
			}
		}
	}()

	if err := os.WriteFile(filepath.Join(root, "Note.md"),
		[]byte("---\ntitle: Renamed\n---\n\n# Renamed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case paths := <-reported:
		if !slices.Equal(paths, []string{"Note.md"}) {
			t.Errorf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the edit never reached the listener")
	}

	// And the index is level with the disk by the time it is announced.
	displayed, err := client.GetNeighbourhood(t.Context(),
		connect.NewRequest(&v1.GetNeighbourhoodRequest{Path: "Note.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if title := displayed.Msg.GetFocus().GetTitle(); title != "Renamed" {
		t.Errorf("the index still says %q", title)
	}
}

// TestABookDroppedInReachesAListener. A client draws every file the vault
// holds, and a book is one of them.
func TestABookDroppedInReachesAListener(t *testing.T) {
	client, root := openVault(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	})

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()

	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}

	reported := make(chan []string, 1)
	go func() {
		for changes.Receive() {
			if paths := changes.Msg().GetPaths(); len(paths) > 0 {
				reported <- paths
				return
			}
		}
	}()

	testsupport.WriteBook(t, root, "library/A Book.epub")

	select {
	case paths := <-reported:
		if !slices.Contains(paths, "library/A Book.epub") {
			t.Errorf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the book never reached the listener")
	}
}

// questions is everything a window asks about the vault it is showing: what the
// One window asks every service that answers them, so a test holds one of these
// and calls a question by its own name.
type questions struct {
	numenv1connect.VaultServiceClient
	numenv1connect.WorkspaceServiceClient
	numenv1connect.FileServiceClient
	numenv1connect.NoteServiceClient
	numenv1connect.SearchServiceClient
}

// newQuestionsClient is that client against a server those services are mounted on.
func newQuestionsClient(client connect.HTTPClient, pos string) questions {
	return questions{
		VaultServiceClient:     numenv1connect.NewVaultServiceClient(client, pos),
		WorkspaceServiceClient: numenv1connect.NewWorkspaceServiceClient(client, pos),
		FileServiceClient:      numenv1connect.NewFileServiceClient(client, pos),
		NoteServiceClient:      numenv1connect.NewNoteServiceClient(client, pos),
		SearchServiceClient:    numenv1connect.NewSearchServiceClient(client, pos),
	}
}

// mountQuestionsServices puts them on a mux, which is what a window serves them from.
func mountQuestionsServices(mux *http.ServeMux, services *editor.Services) {
	mux.Handle(numenv1connect.NewVaultServiceHandler(services.Vault))
	mux.Handle(numenv1connect.NewWorkspaceServiceHandler(services.Workspace))
	mux.Handle(numenv1connect.NewFileServiceHandler(services.File))
	mux.Handle(numenv1connect.NewNoteServiceHandler(services.Note))
	mux.Handle(numenv1connect.NewSearchServiceHandler(services.Search))
}

// going is one vault with everything a window has behind it, and a client
// talking to it the way the window does.
type going struct {
	client questions
	// configuring is the file a person configures this installation in, which
	// is a service of its own beside the vault.
	configuring numenv1connect.SettingsServiceClient
	// drawn is the window itself: what is being done behind it, and the drain
	// that holds it back when it goes.
	canvas       numenv1connect.WindowServiceClient
	installation *editor.Installation
	root         string
	// settings is the vault list this window keeps, which is the folder its
	settings string
	// order is what happened, in the order it happened.
	order *order
}

// order is a record of the moments a test is asking about the sequence of.
type order struct {
	mu     sync.Mutex
	outMsg []string
}

func (o *order) record(what string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.outMsg = append(o.outMsg, what)
}

func (o *order) getAll() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.outMsg...)
}

// openWindow opens a vault the way the window does for an installation nobody has
// configured, with the moment of each write kept and, where a test asked for
// one, the write held there.
func openWindow(t *testing.T, hold *held, notes map[string]string) *going {
	t.Helper()
	return opening(t, hold, notes, true)
}

// writeNamingSettings writes the settings file a rename reads, beside the vault list, which
// is where an installation pointed somewhere of its own keeps one.
func writeNamingSettings(t *testing.T, registry string, sync note.SyncTitleAndFilename) {
	t.Helper()
	body := fmt.Sprintf(`{"naming":{"sync_title_and_filename":%v}}`, bool(sync))
	if err := os.WriteFile(
		filepath.Join(filepath.Dir(registry), "numen.json"), []byte(body), 0o644,
	); err != nil {
		t.Fatal(err)
	}
}

// opening is openWindow with a title and a filename told apart or kept as one
// name, which is the one setting a rename reads.
func opening(t *testing.T, hold *held, notes map[string]string, sync note.SyncTitleAndFilename) *going {
	t.Helper()

	root := t.TempDir()
	for name, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}

	cfg := container.Config{
		IndexPath:       filepath.Join(t.TempDir(), "index.db"),
		RegistryPath:    filepath.Join(t.TempDir(), "vaults.json"),
		PluginCachePath: filepath.Join(t.TempDir(), "plugins"),
	}
	writeNamingSettings(t, cfg.RegistryPath, sync)
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaults.NewAdd(cfg.VaultIdentity(), registry, time.Now).Execute(root, "quitting"); err != nil {
		t.Fatal(err)
	}

	installation, err := editor.Open(t.Context(), editor.NewAssembly(t, cfg), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	// Closing joins the passes still running and lets go of the index, and the
	// folder the index is in is taken away after this.
	t.Cleanup(func() { _ = installation.Close() })

	recorded := &order{}
	// What this watches is the order the writes and the closing land in, so
	// nothing is brought level behind them.
	unlevelled := func(context.Context, domain.Vault, []string) error { return nil }
	writing := note.NewWrite(
		cfg.VaultReaders(),
		recording{VaultWriters: cfg.VaultWriters(), order: recorded, hold: hold},
		unlevelled,
		time.Now,
	)
	installation.Services.SetNoteWrite(&writing)

	turning, settings := numenv1connect.NewSettingsServiceHandler(installation.Services.Settings)
	rendered, itself := numenv1connect.NewWindowServiceHandler(installation.State.Window)
	mux := http.NewServeMux()
	mountQuestionsServices(mux, installation.Services)
	mux.Handle(turning, settings)
	mux.Handle(rendered, itself)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &going{
		client:       newQuestionsClient(server.Client(), server.URL),
		configuring:  numenv1connect.NewSettingsServiceClient(server.Client(), server.URL),
		canvas:       numenv1connect.NewWindowServiceClient(server.Client(), server.URL),
		installation: installation,
		root:         root,
		settings:     cfg.RegistryPath,
		order:        recorded,
	}
}

// read waits for the walk the window started to have been through the vault, so
// that what the index says about a note is there to be asked for.
func (f *going) read(t *testing.T) {
	t.Helper()

	for range 500 {
		if f.installation.State.Ready.Load() {
			return
		}
		if why := f.installation.State.Error.Why(); why != "" {
			t.Fatalf("the vault could not be read: %v", why)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the vault in the window was never read")
}

// held is a writer a test lets through when it says so, which is what a write
// that is still in the air looks like from here.
type held struct {
	begun chan struct{}
	until chan struct{}
	once  sync.Once
}

func newHeld() *held {
	return &held{begun: make(chan struct{}, 1), until: make(chan struct{})}
}

func (h *held) release() { h.once.Do(func() { close(h.until) }) }

// recording is the vault's writers with the moment of a write kept, and the
// write held where a test asked for one.
type recording struct {
	vaultport.VaultWriters
	order *order
	hold  *held
}

func (r recording) Open(v domain.Vault) (vaultport.VaultWriter, error) {
	writer, err := r.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return records{VaultWriter: writer, order: r.order, hold: r.hold}, nil
}

type records struct {
	vaultport.VaultWriter
	order *order
	hold  *held
}

func (w records) Write(
	ctx context.Context, path string, content []byte, ref domain.Fingerprint,
) (domain.Fingerprint, error) {
	if w.hold != nil {
		select {
		case w.hold.begun <- struct{}{}:
		default:
		}
		select {
		case <-w.hold.until:
		case <-ctx.Done():
			return domain.Fingerprint{}, ctx.Err()
		}
	}
	contentVal, err := w.VaultWriter.Write(ctx, path, content, ref)
	w.order.record("wrote " + path)
	return contentVal, err
}

// TestAWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses. The window may not
// take the vault away from a write that is already on its way to it.
func TestAWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses(t *testing.T) {
	hold := newHeld()
	f := openWindow(t, hold, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	defer hold.release()

	writing := make(chan error, 1)
	go func() {
		_, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
			Path: "Note.md",
			Body: "the last thing the person typed\n",
		}))
		writing <- err
	}()

	// The write has reached the vault and is held there, so from here on it is
	// a write the quit must not close the database underneath.
	select {
	case <-hold.begun:
	case <-time.After(5 * time.Second):
		t.Fatal("the write never reached the vault")
	}

	shutCh := make(chan struct{})
	go func() {
		defer close(shutCh)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		f.installation.FlushPending(ctx)
		if err := f.installation.Close(); err != nil {
			t.Error(err)
		}
		f.order.record("closed")
	}()

	select {
	case <-shutCh:
		t.Fatal("the quit closed the database with a write in the air")
	case <-time.After(300 * time.Millisecond):
	}

	hold.release()

	select {
	case <-shutCh:
	case <-time.After(5 * time.Second):
		t.Fatal("the quit never finished")
	}
	if err := <-writing; err != nil {
		t.Fatalf("the write failed: %v", err)
	}

	if got := f.order.getAll(); len(got) != 2 || got[0] != "wrote Note.md" || got[1] != "closed" {
		t.Errorf("the quit went %v", got)
	}
	body, err := os.ReadFile(filepath.Join(f.root, "Note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "the last thing the person typed\n"; !hasBody(string(body), want) {
		t.Errorf("the note holds %q", string(body))
	}
}

// TestTheQuitWaitsForThePageToWriteWhatItOwes. The owed write is a buffer in
// the webview, so the quit asks for it and does not go until it has landed.
func TestTheQuitWaitsForThePageToWriteWhatItOwes(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	stream, err := f.canvas.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	// The stream opens by handing over the token, which asks for nothing.
	if !stream.Receive() {
		t.Fatalf("the stream never opened: %v", stream.Err())
	}
	if stream.Msg().GetFlush() {
		t.Fatal("the stream opened by asking for a flush")
	}

	flushed := make(chan struct{})
	go func() {
		defer close(flushed)
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			// What a page does: it writes what only it holds, and says so
			// afterwards.
			time.Sleep(200 * time.Millisecond)
			if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
				Path: "Note.md",
				Body: "typed and never saved\n",
			})); err != nil {
				t.Error(err)
				return
			}
			if _, err := f.canvas.ReportFlush(t.Context(), connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor,
				Token:  stream.Msg().GetToken(),
				Result: v1.FlushResult_FLUSH_RESULT_WRITTEN,
			})); err != nil {
				t.Error(err)
			}
			return
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	f.installation.FlushPending(ctx)
	f.order.record("settled")

	select {
	case <-flushed:
	case <-time.After(5 * time.Second):
		t.Fatal("the page never answered")
	}

	if got := f.order.getAll(); len(got) != 2 || got[0] != "wrote Note.md" || got[1] != "settled" {
		t.Errorf("the quit went %v", got)
	}

	if err := f.installation.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(f.root, "Note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasBody(string(body), "typed and never saved\n") {
		t.Errorf("the note holds %q", string(body))
	}
}

// TestAPageThatNeverAnswersDoesNotHoldTheQuitPastTheBound. A webview whose
// go gets it.
func TestAPageThatNeverAnswersDoesNotHoldTheQuitPastTheBound(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.installation.Close() })

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	stream, err := f.canvas.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("the stream never opened: %v", stream.Err())
	}

	// The page is listening and will say nothing at all.
	linked := 400 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), linked)
	defer cancel()

	began := time.Now()
	if !f.installation.FlushPending(ctx) {
		t.Fatal("the vault never settled")
	}
	took := time.Since(began)

	if took < linked {
		t.Errorf("the quit gave the page %v of the %v it is owed", took, linked)
	}
	if took > 3*linked {
		t.Errorf("a page that never answered held the quit for %v", took)
	}
}

// hasBody reports whether a note's file carries this prose below whatever
// frontmatter it has.
func hasBody(file, body string) bool {
	return len(file) >= len(body) && file[len(file)-len(body):] == body
}

// speaking is a page listening for the quit the way the window's page does,
// answering each ask with what a test told it to.
type speaking struct {
	f        *going
	token    string
	question chan string
	hangUp   context.CancelFunc
}

// openQuitStream opens the quit stream and takes the token it opens with.
func openQuitStream(t *testing.T, f *going) *speaking {
	t.Helper()

	ctx, hangUp := context.WithCancel(t.Context())
	stream, err := f.canvas.WatchQuit(ctx, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
	if err != nil {
		hangUp()
		t.Fatal(err)
	}
	if !stream.Receive() {
		hangUp()
		t.Fatalf("the stream never opened: %v", stream.Err())
	}
	if stream.Msg().GetFlush() {
		hangUp()
		t.Fatal("the stream opened by asking for a flush")
	}

	p := &speaking{f: f, token: stream.Msg().GetToken(), question: make(chan string, 8), hangUp: hangUp}
	go func() {
		defer close(p.question)
		defer stream.Close()
		for stream.Receive() {
			if stream.Msg().GetFlush() {
				p.question <- stream.Msg().GetToken()
			}
		}
	}()
	t.Cleanup(hangUp)
	return p
}

// answerFlushes is the page doing, at every ask, what a test says a page does.
func (p *speaking) answerFlushes(handler func(token string) v1.FlushResult) {
	go func() {
		for token := range p.question {
			p.sendFlush(token, handler(token))
		}
	}()
}

// sendFlush is the page telling the application what it has left.
func (p *speaking) sendFlush(token string, response v1.FlushResult) {
	_, _ = p.f.canvas.ReportFlush(context.Background(), connect.NewRequest(&v1.ReportFlushRequest{
		Window: wire.Editor,
		Token:  token,
		Result: response,
	}))
}

// endStream is the page's stream ending, and waits for the application to have
// noticed.
func (p *speaking) endStream(t *testing.T) {
	t.Helper()

	p.hangUp()
	select {
	case <-p.question:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream never ended")
	}
	// The stream ends at both ends, and what the application makes of it
	// happens on the thread that was serving it.
	time.Sleep(200 * time.Millisecond)
}

// TestAPageWithAQuestionStandingDoesNotLetTheWindowGo. The text is in that
// buffer and nowhere else, and only the person says where it ends up.
func TestAPageWithAQuestionActiveDoesNotLetTheWindowGo(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.installation.Close() })

	page := openQuitStream(t, f)
	page.answerFlushes(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	began := time.Now()
	if f.installation.FlushPending(ctx) {
		t.Fatal("the vault settled with a question standing")
	}
	if took := time.Since(began); took > 5*time.Second {
		t.Errorf("a page that had answered was waited on for %v", took)
	}

	// The vault is as it was: the person is going back to work, and the writes
	// they do next have to land.
	if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "written after the question was raised\n",
	})); err != nil {
		t.Fatalf("the door was shut on a vault nobody is leaving: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(f.root, "Note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasBody(string(body), "written after the question was raised\n") {
		t.Errorf("the note holds %q", string(body))
	}
}

// TestTheWindowGoesOnceTheQuestionsAreAnswered.
func TestTheWindowGoesOnceTheQuestionsAreAnswered(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.installation.Close() })

	page := openQuitStream(t, f)
	page.answerFlushes(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	asking, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if f.installation.FlushPending(asking) {
		t.Fatal("the vault settled with a question standing")
	}

	// What the window does with a close it called off: it waits on a person,
	// and that wait is not measured.
	reply := make(chan bool, 1)
	go func() { reply <- f.installation.WaitForFlush(t.Context()) }()

	select {
	case <-reply:
		t.Fatal("the window was let go before the person answered")
	case <-time.After(300 * time.Millisecond):
	}

	// The person answered every question, and the page has nothing left.
	page.sendFlush(page.token, v1.FlushResult_FLUSH_RESULT_WRITTEN)

	select {
	case let := <-reply:
		if !let {
			t.Fatal("the page has nothing left and the window was not let go")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the window was never let go")
	}
}

// TestACloseCalledOffAsksThePageAgain. A second quit is a second ask, and what
// the page holds is written for it.
func TestACloseCalledOffAsksThePageAgain(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.installation.Close() })

	page := openQuitStream(t, f)
	asks := 0
	page.answerFlushes(func(_ string) v1.FlushResult {
		asks++
		if asks == 1 {
			return v1.FlushResult_FLUSH_RESULT_ASKING
		}
		// What a page does once the person has said what happens to the text.
		if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
			Path: "Note.md",
			Body: "typed and never saved\n",
		})); err != nil {
			return v1.FlushResult_FLUSH_RESULT_ASKING
		}
		return v1.FlushResult_FLUSH_RESULT_WRITTEN
	})

	first, cancelFirst := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelFirst()
	if f.installation.FlushPending(first) {
		t.Fatal("the vault settled with a question standing")
	}

	second, cancelSecond := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelSecond()
	if !f.installation.FlushPending(second) {
		t.Fatal("the second quit did not settle the vault")
	}

	if got := f.order.getAll(); len(got) != 1 || got[0] != "wrote Note.md" {
		t.Errorf("the quit went %v", got)
	}
	body, err := os.ReadFile(filepath.Join(f.root, "Note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !hasBody(string(body), "typed and never saved\n") {
		t.Errorf("the note holds %q", string(body))
	}
}

// TestAPageThatGoesWithAQuestionStandingIsWaitedForAndThenLeftBehind. A stream
// ending is not an answer, so what the page held is waited for; and it is
// waited for as silence is, which is the bound and no longer.
func TestAPageThatGoesWithAQuestionActiveIsWaitedForAndThenLeftPrior(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.installation.Close() })

	page := openQuitStream(t, f)
	page.answerFlushes(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	asking, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if f.installation.FlushPending(asking) {
		t.Fatal("the vault settled with a question standing")
	}
	page.endStream(t)

	linked := 400 * time.Millisecond
	ctx, endsAt := context.WithTimeout(t.Context(), linked)
	defer endsAt()

	began := time.Now()
	if !f.installation.FlushPending(ctx) {
		t.Fatal("the vault never settled after the page went")
	}
	took := time.Since(began)

	if took < linked {
		t.Errorf("the quit gave the page %v of the %v it is owed", took, linked)
	}
	if took > 3*linked {
		t.Errorf("a page that went held the quit for %v", took)
	}
}

// TestAPageThatComesBackRaisesItsQuestionAgain. The client is back after a
// second under a token of its own, and the text it holds is still its own.
func TestAPageThatReachesBackRaisesItsQuestionAgain(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.installation.Close() })

	first := openQuitStream(t, f)
	first.answerFlushes(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	asking, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if f.installation.FlushPending(asking) {
		t.Fatal("the vault settled with a question standing")
	}
	first.endStream(t)

	back := openQuitStream(t, f)
	if back.token == first.token {
		t.Fatalf("the page that came back listens under the token that went, %q", back.token)
	}
	back.answerFlushes(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	again, cancelAgain := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelAgain()
	if f.installation.FlushPending(again) {
		t.Fatal("the vault settled with the question raised again")
	}
}

// The window asks for every setting and writes one of them back.

// them. What the file leaves out stands there at its default.
func getSetting(t *testing.T, f *going, pos ...string) any {
	t.Helper()
	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var target any
	if err := json.Unmarshal([]byte(response.Msg.GetWritten()), &target); err != nil {
		t.Fatalf("the settings read out as %q: %v", response.Msg.GetWritten(), err)
	}
	for _, step := range pos {
		section, is := target.(map[string]any)
		if !is {
			t.Fatalf("%s is not a section of the settings", strings.Join(pos, "."))
		}
		target = section[step]
	}
	return target
}

// The settings read out hold what the file leaves out, so the window draws what
// this installation is doing.
func TestTheSettingsReadOutStandOnTheDefaults(t *testing.T) {
	f := opening(t, nil, nil, true)

	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetPath() == "" {
		t.Error("the settings stand nowhere")
	}

	var target map[string]any
	if err := json.Unmarshal([]byte(response.Msg.GetWritten()), &target); err != nil {
		t.Fatalf("the settings read out as %q: %v", response.Msg.GetWritten(), err)
	}
	if _, exists := target["appearance"]; !exists {
		t.Errorf("the settings read out as %v", target)
	}

	// A model reaches the window with the setting it is read from and what
	// choosing it writes, which is the whole of what the window does with one.
	models := response.Msg.GetModels()
	if len(models) == 0 {
		t.Fatal("no setting names a model")
	}
	for _, one := range models {
		if len(one.GetNamedAt()) == 0 || len(one.GetWrites()) == 0 {
			t.Errorf("%q is read from nowhere, or writes nothing", one.GetTitle())
		}
	}
}

// A setting written is answered by the next question, and the model chosen for
// the agent stands where the models said it would.
func TestASettingSavedIsAnsweredByTheNextQuestion(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}

	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", response.Msg.GetWritten())
	}
}

// A value the settings could not be read out of again is the client's to
// correct, and nothing is written.
func TestAValueTheSettingsCannotHoldIsRefused(t *testing.T) {
	for _, one := range []struct {
		what   string
		posVal []string
		value  string
	}{
		{"is not JSON at all", []string{"agent", "claude", "model"}, `opus`},
		{"is of the wrong shape", []string{"agent", "claude", "model"}, `7`},
		{"is past what the setting goes to", []string{"appearance", "text_scale"}, `5`},
	} {
		t.Run(one.what, func(t *testing.T) {
			f := opening(t, nil, nil, true)
			was, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
			if err != nil {
				t.Fatal(err)
			}

			_, err = f.configuring.WriteSettings(
				t.Context(),
				connect.NewRequest(&v1.WriteSettingsRequest{
					Settings: []*v1.Setting{{At: one.posVal, Value: one.value}},
				}),
			)
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("the value was taken as %v", err)
			}

			// The settings still read, and read as they did.
			now, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
			if err != nil {
				t.Fatal(err)
			}
			if now.Msg.GetWritten() != was.Msg.GetWritten() {
				t.Errorf("the settings read out as %s", now.Msg.GetWritten())
			}
		})
	}
}

// Settings are written together or not at all, so one the settings cannot hold
// leaves the ones beside it where they were.
func TestSettingsSavedTogetherLeaveTheFileAloneWhereOneIsRefused(t *testing.T) {
	f := opening(t, nil, nil, true)

	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"agent", "claude", "model"}, Value: `"opus"`},
			{At: []string{"appearance", "text_scale"}, Value: `5`},
		},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("the settings were taken as %v", err)
	}

	response, err := f.configuring.GetSettings(t.Context(), connect.NewRequest(&v1.GetSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(response.Msg.GetWritten(), `"model":"opus"`) {
		t.Errorf("the settings read out as %s", response.Msg.GetWritten())
	}
}

// The levels of reasoning a model offers reach the window with the model.
func TestTheLevelsOfAModelReachTheWindow(t *testing.T) {
	got := editor.NewWireModels([]settings.Model{{
		Name: "deep", ReasoningLevels: []settings.ReasoningLevel{{Name: "low", Title: "Low"}}, DefaultReasoning: "low",
	}, {Name: "plain"}})

	if levels := got[0].GetReasoningLevels(); len(levels) != 1 || levels[0].GetName() != "low" || levels[0].GetTitle() != "Low" ||
		got[0].GetDefaultReasoning() != "low" {
		t.Errorf("the model is %v", got[0])
	}
	if len(got[1].GetReasoningLevels()) != 0 {
		t.Errorf("a model offering none is %v", got[1])
	}
}

func isSyncing(t *testing.T, f *going) bool {
	t.Helper()
	target, is := getSetting(t, f, "naming", "sync_title_and_filename").(bool)
	if !is {
		t.Fatal("the setting is not written as a switch")
	}
	return target
}

func writeSetting(t *testing.T, f *going, value string, path ...string) error {
	t.Helper()
	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{{At: path, Value: value}},
	}))
	return err
}

// GetSettings returns a value WriteSettings stored, without a restart.
func TestGetSettingsReturnsTheNewValueAfterWriteSettings(t *testing.T) {
	f := opening(t, nil, nil, true)

	if !isSyncing(t, f) {
		t.Fatal("an installation nobody has configured keeps the names apart")
	}
	if err := writeSetting(t, f, "false", "naming", "sync_title_and_filename"); err != nil {
		t.Fatal(err)
	}
	if isSyncing(t, f) {
		t.Error("the setting was written and the names still sync")
	}
}

// A number the setting does not take is not written, and the settings are left
// as they are.
func TestWriteSettingsRejectsANumberOutOfBounds(t *testing.T) {
	f := opening(t, nil, nil, true)

	if err := writeSetting(t, f, "20", "appearance", "text_scale"); err == nil {
		t.Fatal("a text scale of twenty was written")
	}
	if got := getSetting(t, f, "appearance", "text_scale"); got != float64(1) {
		t.Errorf("the text scale is %v", got)
	}
}

// The setting a person writes is written where they will read it, and every
// other byte of the file is left as they typed it.
func TestWriteSettingsKeepsTheRestOfTheFile(t *testing.T) {
	f := opening(t, nil, nil, true)
	path := filepath.Join(filepath.Dir(f.settings), "numen.json")
	if err := os.WriteFile(path, []byte("{\n  \"appearance\": {\"text_scale\": 1.5}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := writeSetting(t, f, "false", "naming", "sync_title_and_filename"); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"text_scale": 1.5`) {
		t.Errorf("what the person typed was rewritten:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"sync_title_and_filename": false`) {
		t.Errorf("the setting is not in the file:\n%s", raw)
	}
}

func TestANoteIsFoundByName(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\n## Heat and work\n\nA reversible engine.\n",
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n",
	})

	answer, err := client.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "entro"}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("a word still being typed found no name")
	}
	first := found[0]
	if first.GetNote().GetTitle() != "Entropy" {
		t.Errorf("the first answer is %q, want the note called that", first.GetNote().GetTitle())
	}
	if first.GetHeading() != nil {
		t.Errorf("a title matched, and the answer carries heading %+v", first.GetHeading())
	}
	if len(first.GetSpans()) == 0 {
		t.Error("the answer says nothing about where it matched")
	}
}

func TestAHeadingIsFoundWithTheLineItExistsOn(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\n## Heat and work\n\nA reversible engine.\n",
	})

	answer, err := client.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "heat"}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) != 1 {
		t.Fatalf("%d answers, want the one heading: %+v", len(found), found)
	}
	heading := found[0].GetHeading()
	if heading.GetText() != "Heat and work" {
		t.Errorf("the answer is heading %q", heading.GetText())
	}
	if heading.GetLine() < 0 {
		t.Errorf("the heading is on line %d, which is no line", heading.GetLine())
	}
}

// TestANameSaysWhichOfFourTheNoteIs. The palette draws a typed note by its type.
func TestANameReturnsWhichOfFourTheNoteIs(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
		"Animals.md": "---\ntype: sample\ntitle: Animals\n---\n\n## Entropy of a llama\n",
	})

	answer, err := client.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "entropy"}))
	if err != nil {
		t.Fatal(err)
	}

	types := map[string]string{}
	for _, one := range answer.Msg.GetFound() {
		types[one.GetNote().GetPath()] = one.GetType()
	}
	want := map[string]string{
		"Entropy.md": "",
		"Animals.md": "sample",
	}
	for path, is := range want {
		if got, target := types[path]; !target || got != is {
			t.Errorf("the name found in %s is drawn as %v, want %v", path, got, is)
		}
	}
}

// TestAPassageSaysWhichOfFourItsNoteIs. A passage is drawn with the mark of the
// note it was read out of.
func TestAPassageReturnsWhichOfFourItsNoteIs(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Daily.md": "---\ntype: preset\ntitle: Daily\n---\n\n# Daily\n\nNo engine beats a reversible engine.\n",
	})

	answer, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: "reversible", Mode: v1.SearchMode_SEARCH_MODE_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("nothing found")
	}
	if is := found[0].GetType(); is != "preset" {
		t.Errorf("the passage is drawn as %v, want the preset it was read out of", is)
	}
}

// TestAPassageSaysWhatTheVaultHoldsAtItsPath. A list a person runs their eye
// down draws a note as the note it is, and another file as a file.
func TestAPassageReturnsWhatTheVaultContainsAtItsPath(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Engines.md":          "---\ntitle: Engines\n---\n\n# Engines\n\nA book of engines.\n",
		"library/A Book.epub": string(testsupport.Book(t)),
	})

	// The text of a book is taken out of it behind the window, and the search
	// reaches it once that has finished.
	kinds := map[string]bool{}
	for range 500 {
		answer, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
			Query: "book", Mode: v1.SearchMode_SEARCH_MODE_WORDS,
		}))
		if err != nil {
			t.Fatal(err)
		}
		clear(kinds)
		for _, one := range answer.Msg.GetFound() {
			kinds[one.GetPath()] = one.GetNote() != nil
		}
		if _, target := kinds["library/A Book.epub"]; target {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	want := map[string]bool{
		"Engines.md":          true,
		"library/A Book.epub": false,
	}
	for path, is := range want {
		if got, targetVal := kinds[path]; !targetVal || got != is {
			t.Errorf("the passage out of %s came back as a note %v, want %v", path, got, is)
		}
	}
}

func TestEachModeIsAskedByItself(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n\nNo engine beats a reversible engine.\n",
	})

	// No model is set here, so the meaning half has nothing to answer with and
	// the words half answers on its own. Which is which is the handler's to get
	// right: both halves answer with the same shape.
	words, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: "reversible", Mode: v1.SearchMode_SEARCH_MODE_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(words.Msg.GetFound()) == 0 {
		t.Fatal("the words half found nothing for a word that is written")
	}

	meaning, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: "reversible", Mode: v1.SearchMode_SEARCH_MODE_MEANING,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(meaning.Msg.GetFound()); got != 0 {
		t.Errorf("the meaning half answered %d times with no model set", got)
	}
}

func TestAPassageReturnsWhichNoteItCameOutOf(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n\nNo engine beats a reversible engine.\n",
	})

	answer, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: "reversible", Mode: v1.SearchMode_SEARCH_MODE_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("nothing found")
	}
	first := found[0]
	if first.GetNote().GetTitle() != "Engines" {
		t.Errorf("the passage says it came out of %q", first.GetNote().GetTitle())
	}
	if first.GetPath() == "" {
		t.Error("the passage says nothing about where it came from")
	}
	if len(first.GetSpans()) == 0 {
		t.Error("the passage says nothing about where the word typed is in it")
	}
}

// A hit is a place in a source, and the client is told which place: it is what
// opens the source there, and nothing the client holds says it.
func TestAPassageReturnsWhereInItsSourceItExists(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Engines.md": "---\ntitle: Engines\n---\n\n# Engines\n\nNo engine beats a reversible engine.\n",
	})

	answer, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: "reversible", Mode: v1.SearchMode_SEARCH_MODE_WORDS,
	}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetFound()
	if len(found) == 0 {
		t.Fatal("nothing found")
	}
	first := found[0]
	pos := first.GetSpan()
	if pos.GetTo() <= pos.GetFrom() {
		t.Errorf("the passage is over no text: %d to %d", pos.GetFrom(), pos.GetTo())
	}
	if pos.GetFrom() < 0 {
		t.Errorf("the passage begins at %d", pos.GetFrom())
	}
}

func TestAnAnswerIsSplitToWhatWasAskedFor(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"a.md": "# Entropy one\n",
		"b.md": "# Entropy two\n",
		"c.md": "# Entropy three\n",
	})

	answer, err := client.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{
		Query: "entropy", Limit: 2,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(answer.Msg.GetFound()); got != 2 {
		t.Errorf("%d answers for a limit of 2", got)
	}
}

func TestNothingTypedIsAnsweredWithNothing(t *testing.T) {
	client, _ := openVault(t, map[string]string{"a.md": "# Entropy\n"})

	for _, query := range []string{"", "   "} {
		answer, err := client.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{Query: query}))
		if err != nil {
			t.Fatalf("%q: %v", query, err)
		}
		if got := len(answer.Msg.GetFound()); got != 0 {
			t.Errorf("%q was answered with %d passages", query, got)
		}
	}
}

// What a note is divided into, over the wire.

func TestANoteReturnsWhatItIsDividedInto(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n\n## Heat and work\n\nA reversible engine.\n",
	})

	answer, err := client.ListHeadings(t.Context(),
		connect.NewRequest(&v1.ListHeadingsRequest{Paths: []string{"Entropy.md"}}))
	if err != nil {
		t.Fatal(err)
	}

	found := answer.Msg.GetHeadings()
	if len(found) != 1 || found[0].GetPath() != "Entropy.md" {
		t.Fatalf("found = %+v, want the one note the filter named", found)
	}

	headings := found[0].GetHeadings()
	if len(headings) != 2 {
		t.Fatalf("headings = %+v, want the two the note carries", headings)
	}
	if headings[0].GetText() != "Entropy" || headings[0].GetLevel() != 1 {
		t.Errorf("the first is %+v", headings[0])
	}
	if headings[1].GetText() != "Heat and work" || headings[1].GetLevel() != 2 {
		t.Errorf("the second is %+v", headings[1])
	}
	if !(headings[0].GetLine() < headings[1].GetLine()) {
		t.Errorf("lines are %d and %d, want them in the order they stand",
			headings[0].GetLine(), headings[1].GetLine())
	}
}

// TestANeighbourhoodSaysWhichOfThreeEachNoteIs. A plex draws each typed note as
// its type, and the picture is where it is told which is which.
func TestANeighbourhoodReturnsWhichOfThreeEachNoteIs(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\nlinks:\n  - to: Animals\n    role: child\n" +
			"  - to: Animal\n    role: child\n---\n\nan animal is a note\n",
		"Animals.md": "---\ntype: sample\ntitle: Animals\n---\n\n## Llama\n\n[[Animal]]\n",
		"Animal.md": "---\ntype: form\ntitle: Animal\ncolour: brown\n---\n\n" +
			"## Layout\n\n### Header\n\nan animal\n\n### Footer\n\nheight\n",
	})

	center, err := client.GetNeighbourhood(t.Context(),
		connect.NewRequest(&v1.GetNeighbourhoodRequest{Path: "Ontology.md"}))
	if err != nil {
		t.Fatal(err)
	}

	if is := center.Msg.GetFocusType(); is != "" {
		t.Errorf("the note in focus is drawn as %v", is)
	}
	types := map[string]string{}
	for _, one := range center.Msg.GetRelated() {
		types[one.GetNote().GetPath()] = one.GetType()
	}
	want := map[string]string{
		"Animals.md": "sample",
		"Animal.md":  "form",
	}
	for path, is := range want {
		if got, target := types[path]; !target || got != is {
			t.Errorf("the node %s is drawn as %v, want %v", path, got, is)
		}
	}

	// And the note in focus is said the same way, a plex standing on a typed note.
	item, err := client.GetNeighbourhood(t.Context(),
		connect.NewRequest(&v1.GetNeighbourhoodRequest{Path: "Animals.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if is := item.Msg.GetFocusType(); is != "sample" {
		t.Errorf("the sample in focus is drawn as %v", is)
	}
}

func TestAPathAskedTwiceIsAnsweredOnce(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})

	answer, err := client.ListHeadings(t.Context(), connect.NewRequest(&v1.ListHeadingsRequest{
		Paths: []string{"Entropy.md", "Entropy.md", "Entropy.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if found := answer.Msg.GetHeadings(); len(found) != 1 {
		t.Errorf("found = %+v, want one entry for the one note", found)
	}
}

// TestMorePathsThanAreAnsweredAtOnceAreRefused. A note carrying no headings is
// absent from the answer, so an answer cut to fit the ceiling would be one a
// caller cannot tell from a note with nothing in it.
func TestMorePathsThanAreAnsweredAtOnceAreRefused(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})

	paths := make([]string, 0, 201)
	for i := range 201 {
		paths = append(paths, fmt.Sprintf("Note%03d.md", i))
	}

	_, err := client.ListHeadings(t.Context(),
		connect.NewRequest(&v1.ListHeadingsRequest{Paths: paths}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err = %v, want an error of the paths named", err)
	}
}

func TestANoteCarryingNoHeadingIsAbsent(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Plain.md": "---\ntitle: Plain\n---\n\nProse and nothing else.\n",
	})

	answer, err := client.ListHeadings(t.Context(), connect.NewRequest(&v1.ListHeadingsRequest{
		Paths: []string{"Plain.md", "Gone.md"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if found := answer.Msg.GetHeadings(); len(found) != 0 {
		t.Errorf("found = %+v, want nothing for a note with no headings", found)
	}
}
