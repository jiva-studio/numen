package webui_test

import (
	"context"
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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// going is one vault with everything a window has behind it, and a client
// talking to it the way the window does.
type going struct {
	client numenv1connect.VaultServiceClient
	opened *webui.Opened
	root   string
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

// quitting opens a vault the way the window does, with the moment of each
// write kept and, where a test asked for one, the write held there.
func quitting(t *testing.T, hold *held, notes map[string]string) *going {
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

	settings := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := settings.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (usecase.Add{
		Registry: registry,
		Identity: settings.VaultIdentity(),
		Now:      time.Now,
	}).Execute(root, "quitting"); err != nil {
		t.Fatal(err)
	}

	opened, err := webui.Open(t.Context(), settings, os.Stderr)
	if err != nil {
		t.Fatal(err)
	}

	recorded := &order{}
	opened.API.Saves = &note.Write{
		Readers: settings.VaultReaders(),
		Writers: recording{VaultWriters: settings.VaultWriters(), order: recorded, hold: hold},
	}

	route, handler := numenv1connect.NewVaultServiceHandler(opened.API)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &going{
		client: numenv1connect.NewVaultServiceClient(server.Client(), server.URL),
		opened: opened,
		root:   root,
		order:  recorded,
	}
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

func (w records) Write(ctx context.Context, path string, content []byte, ref domain.FileRef) error {
	if w.hold != nil {
		select {
		case w.hold.begun <- struct{}{}:
		default:
		}
		select {
		case <-w.hold.until:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	err := w.VaultWriter.Write(ctx, path, content, ref)
	w.order.at("wrote " + path)
	return err
}

// TestAWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses. The window may not
// take the vault away from a write that is already on its way to it.
func TestAWriteInFlightAtTheQuitLandsBeforeTheDatabaseCloses(t *testing.T) {
	hold := holding()
	f := quitting(t, hold, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	defer hold.release()

	writing := make(chan error, 1)
	go func() {
		_, err := f.client.Write(context.Background(), connect.NewRequest(&v1.WriteRequest{
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	listening, hangUp := context.WithCancel(context.Background())
	defer hangUp()

	stream, err := f.client.Quitting(listening, connect.NewRequest(&v1.QuittingRequest{}))
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
			if _, err := f.client.Write(context.Background(), connect.NewRequest(&v1.WriteRequest{
				Path: "Note.md",
				Body: "typed and never saved\n",
			})); err != nil {
				t.Error(err)
				return
			}
			if _, err := f.client.Flushed(context.Background(), connect.NewRequest(&v1.FlushedRequest{
				Token: stream.Msg().GetToken(),
			})); err != nil {
				t.Error(err)
			}
			return
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

	listening, hangUp := context.WithCancel(context.Background())
	defer hangUp()

	stream, err := f.client.Quitting(listening, connect.NewRequest(&v1.QuittingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatalf("the stream never opened: %v", stream.Err())
	}

	// The page is listening and will say nothing at all.
	bound := 400 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()

	began := time.Now()
	f.opened.Settle(ctx)
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
