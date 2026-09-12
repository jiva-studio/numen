package editor_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// going is one vault with everything a window has behind it, and a client
// talking to it the way the window does.
type going struct {
	client questions
	// configuring is the file a person configures this installation in, which
	// is a service of its own beside the vault.
	configuring numenv1connect.SettingsServiceClient
	// drawn is the window itself: what is being done behind it, and the drain
	// that holds it back when it goes.
	drawn  numenv1connect.WindowServiceClient
	opened *editor.Installation
	root   string
	// settings is the vault list this window keeps, which is the folder its
	// settings file sits in.
	settings string
	// order is what happened, in the order it happened.
	order *order
}

// order is a record of the moments a test is asking about the sequence of.
type order struct {
	mu   sync.Mutex
	said []string
}

func (o *order) at(what string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.said = append(o.said, what)
}

func (o *order) taken() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.said...)
}

// quitting opens a vault the way the window does for an installation nobody has
// configured, with the moment of each write kept and, where a test asked for
// one, the write held there.
func quitting(t *testing.T, hold *held, notes map[string]string) *going {
	t.Helper()
	return opening(t, hold, notes, true)
}

// naming writes the settings file a rename reads, beside the vault list, which
// is where an installation pointed somewhere of its own keeps one.
func naming(t *testing.T, registry string, sync note.SyncTitleAndFilename) {
	t.Helper()
	body := fmt.Sprintf(`{"naming":{"sync_title_and_filename":%v}}`, bool(sync))
	if err := os.WriteFile(
		filepath.Join(filepath.Dir(registry), "numen.json"), []byte(body), 0o644,
	); err != nil {
		t.Fatal(err)
	}
}

// opening is quitting with a title and a filename told apart or kept as one
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
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}

	cfg := container.Config{
		IndexPath:     filepath.Join(t.TempDir(), "index.db"),
		RegistryPath:  filepath.Join(t.TempDir(), "vaults.json"),
		SchedulesPath: filepath.Join(t.TempDir(), "flashcards"),
	}
	naming(t, cfg.RegistryPath, sync)
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (vaults.Add{
		Registry: registry,
		Identity: cfg.VaultIdentity(),
		Now:      time.Now,
	}).Execute(root, "quitting"); err != nil {
		t.Fatal(err)
	}

	opened, err := editor.Open(t.Context(), cfg, "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	// Closing joins the passes still running and lets go of the index, and the
	// folder the index is in is taken away after this.
	t.Cleanup(func() { _ = opened.Close() })

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
	opened.API.Notes.Write = &writing

	turning, settings := numenv1connect.NewSettingsServiceHandler(opened.API)
	drawn, itself := numenv1connect.NewWindowServiceHandler(opened.API.Window)
	mux := http.NewServeMux()
	answers(mux, opened.API)
	mux.Handle(turning, settings)
	mux.Handle(drawn, itself)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &going{
		client:      asks(server.Client(), server.URL),
		configuring: numenv1connect.NewSettingsServiceClient(server.Client(), server.URL),
		drawn:       numenv1connect.NewWindowServiceClient(server.Client(), server.URL),
		opened:      opened,
		root:        root,
		settings:    cfg.RegistryPath,
		order:       recorded,
	}
}

// read waits for the walk the window started to have been through the vault, so
// that what the index says about a note is there to be asked for.
func (f *going) read(t *testing.T) {
	t.Helper()

	for range 500 {
		if f.opened.API.Ready.Load() {
			return
		}
		if why := f.opened.API.Error.Why(); why != "" {
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

func holding() *held {
	return &held{begun: make(chan struct{}, 1), until: make(chan struct{})}
}

func (h *held) release() { h.once.Do(func() { close(h.until) }) }

// recording is the vault's writers with the moment of a write kept, and the
// write held where a test asked for one.
type recording struct {
	port.VaultWriters
	order *order
	hold  *held
}

func (r recording) Open(v domain.Vault) (port.VaultWriter, error) {
	writer, err := r.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return records{VaultWriter: writer, order: r.order, hold: r.hold}, nil
}

type records struct {
	port.VaultWriter
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
	written, err := w.VaultWriter.Write(ctx, path, content, ref)
	w.order.at("wrote " + path)
	return written, err
}

// TestAWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses. The window may not
// take the vault away from a write that is already on its way to it.
func TestAWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses(t *testing.T) {
	hold := holding()
	f := quitting(t, hold, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
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

	shut := make(chan struct{})
	go func() {
		defer close(shut)
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		f.opened.Settle(ctx)
		if err := f.opened.Close(); err != nil {
			t.Error(err)
		}
		f.order.at("closed")
	}()

	select {
	case <-shut:
		t.Fatal("the quit closed the database with a write in the air")
	case <-time.After(300 * time.Millisecond):
	}

	hold.release()

	select {
	case <-shut:
	case <-time.After(5 * time.Second):
		t.Fatal("the quit never finished")
	}
	if err := <-writing; err != nil {
		t.Fatalf("the write was refused: %v", err)
	}

	if got := f.order.taken(); len(got) != 2 || got[0] != "wrote Note.md" || got[1] != "closed" {
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
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	stream, err := f.drawn.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
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
			if _, err := f.drawn.ReportFlush(t.Context(), connect.NewRequest(&v1.ReportFlushRequest{
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
	f.opened.Settle(ctx)
	f.order.at("settled")

	select {
	case <-flushed:
	case <-time.After(5 * time.Second):
		t.Fatal("the page never answered")
	}

	if got := f.order.taken(); len(got) != 2 || got[0] != "wrote Note.md" || got[1] != "settled" {
		t.Errorf("the quit went %v", got)
	}

	if err := f.opened.Close(); err != nil {
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
// script has stopped answers never, and a person who asked for the window to
// go gets it.
func TestAPageThatNeverAnswersDoesNotHoldTheQuitPastTheBound(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.opened.Close() })

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	stream, err := f.drawn.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("the stream never opened: %v", stream.Err())
	}

	// The page is listening and will say nothing at all.
	bound := 400 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), bound)
	defer cancel()

	began := time.Now()
	if !f.opened.Settle(ctx) {
		t.Fatal("the vault never settled")
	}
	took := time.Since(began)

	if took < bound {
		t.Errorf("the quit gave the page %v of the %v it is owed", took, bound)
	}
	if took > 3*bound {
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
	f      *going
	token  string
	asked  chan string
	hangUp context.CancelFunc
}

// listening opens the quit stream and takes the token it opens with.
func listening(t *testing.T, f *going) *speaking {
	t.Helper()

	ctx, hangUp := context.WithCancel(t.Context())
	stream, err := f.drawn.WatchQuit(ctx, connect.NewRequest(&v1.WatchQuitRequest{Window: wire.Editor}))
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

	p := &speaking{f: f, token: stream.Msg().GetToken(), asked: make(chan string, 8), hangUp: hangUp}
	go func() {
		defer close(p.asked)
		defer stream.Close()
		for stream.Receive() {
			if stream.Msg().GetFlush() {
				p.asked <- stream.Msg().GetToken()
			}
		}
	}()
	t.Cleanup(hangUp)
	return p
}

// answering is the page doing, at every ask, what a test says a page does.
func (p *speaking) answering(doing func(token string) v1.FlushResult) {
	go func() {
		for token := range p.asked {
			p.says(token, doing(token))
		}
	}()
}

// says is the page telling the application what it has left.
func (p *speaking) says(token string, said v1.FlushResult) {
	_, _ = p.f.drawn.ReportFlush(context.Background(), connect.NewRequest(&v1.ReportFlushRequest{
		Window: wire.Editor,
		Token:  token,
		Result: said,
	}))
}

// went is the page's stream ending, and waits for the application to have
// noticed.
func (p *speaking) went(t *testing.T) {
	t.Helper()

	p.hangUp()
	select {
	case <-p.asked:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream never ended")
	}
	// The stream ends at both ends, and what the application makes of it
	// happens on the thread that was serving it.
	time.Sleep(200 * time.Millisecond)
}

// TestAPageWithAQuestionStandingDoesNotLetTheWindowGo. The text is in that
// buffer and nowhere else, and only the person says where it ends up.
func TestAPageWithAQuestionStandingDoesNotLetTheWindowGo(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.opened.Close() })

	page := listening(t, f)
	page.answering(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	began := time.Now()
	if f.opened.Settle(ctx) {
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
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.opened.Close() })

	page := listening(t, f)
	page.answering(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	asking, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if f.opened.Settle(asking) {
		t.Fatal("the vault settled with a question standing")
	}

	// What the window does with a close it called off: it waits on a person,
	// and that wait is not measured.
	answered := make(chan bool, 1)
	go func() { answered <- f.opened.Answered(t.Context()) }()

	select {
	case <-answered:
		t.Fatal("the window was let go before the person answered")
	case <-time.After(300 * time.Millisecond):
	}

	// The person answered every question, and the page has nothing left.
	page.says(page.token, v1.FlushResult_FLUSH_RESULT_WRITTEN)

	select {
	case let := <-answered:
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
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.opened.Close() })

	page := listening(t, f)
	asks := 0
	page.answering(func(token string) v1.FlushResult {
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
	if f.opened.Settle(first) {
		t.Fatal("the vault settled with a question standing")
	}

	second, cancelSecond := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelSecond()
	if !f.opened.Settle(second) {
		t.Fatal("the second quit did not settle the vault")
	}

	if got := f.order.taken(); len(got) != 1 || got[0] != "wrote Note.md" {
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
func TestAPageThatGoesWithAQuestionStandingIsWaitedForAndThenLeftBehind(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.opened.Close() })

	page := listening(t, f)
	page.answering(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	asking, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if f.opened.Settle(asking) {
		t.Fatal("the vault settled with a question standing")
	}
	page.went(t)

	bound := 400 * time.Millisecond
	ctx, endsAt := context.WithTimeout(t.Context(), bound)
	defer endsAt()

	began := time.Now()
	if !f.opened.Settle(ctx) {
		t.Fatal("the vault never settled after the page went")
	}
	took := time.Since(began)

	if took < bound {
		t.Errorf("the quit gave the page %v of the %v it is owed", took, bound)
	}
	if took > 3*bound {
		t.Errorf("a page that went held the quit for %v", took)
	}
}

// TestAPageThatComesBackRaisesItsQuestionAgain. The client is back after a
// second under a token of its own, and the text it holds is still its own.
func TestAPageThatComesBackRaisesItsQuestionAgain(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	t.Cleanup(func() { f.opened.Close() })

	first := listening(t, f)
	first.answering(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	asking, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if f.opened.Settle(asking) {
		t.Fatal("the vault settled with a question standing")
	}
	first.went(t)

	back := listening(t, f)
	if back.token == first.token {
		t.Fatalf("the page that came back listens under the token that went, %q", back.token)
	}
	back.answering(func(string) v1.FlushResult { return v1.FlushResult_FLUSH_RESULT_ASKING })

	again, cancelAgain := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancelAgain()
	if f.opened.Settle(again) {
		t.Fatal("the vault settled with the question raised again")
	}
}
