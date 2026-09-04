package webui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// The two vaults a swap is asked about, and one note in each.
const (
	entropy  = "Entropy.md"
	enthalpy = "Enthalpy.md"
)

func noteNamed(title string) string {
	return "---\ntitle: " + title + "\n---\n\n# " + title + "\n"
}

// showing is one installation holding two vaults, with a window open on the
// first and a client talking to it the way the window does.
type showing struct {
	client questions
	// drawn is the window itself, which the drain that holds it back is asked
	// of.
	drawn  numenv1connect.WindowServiceClient
	opened *webui.Installation
	cfg    container.Config
	first  domain.Vault
	second domain.Vault
}

// swapping opens a window on the first of two vaults, both on the list this
// installation keeps.
func swapping(t *testing.T) *showing {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	first := listed(t, cfg, registry, "one", map[string]string{entropy: noteNamed("Entropy")})
	second := listed(t, cfg, registry, "two", map[string]string{enthalpy: noteNamed("Enthalpy")})

	opened, err := webui.Open(t.Context(), cfg, "one", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { opened.Close() })

	going, itself := numenv1connect.NewWindowServiceHandler(opened.API.Window)
	mux := http.NewServeMux()
	answers(mux, opened.API)
	mux.Handle(going, itself)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	f := &showing{
		client: asks(server.Client(), server.URL),
		drawn:  numenv1connect.NewWindowServiceClient(server.Client(), server.URL),
		opened: opened,
		cfg:    cfg,
		first:  first,
		second: second,
	}
	f.read(t)
	return f
}

// listed writes a folder of notes, makes it a vault and puts it on the list.
func listed(
	t *testing.T,
	cfg container.Config,
	registry port.VaultRegistry,
	name string,
	notes map[string]string,
) domain.Vault {
	t.Helper()

	root := t.TempDir()
	for at, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(at))
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
	v, err := usecase.Add{
		Registry: registry,
		Identity: cfg.VaultIdentity(),
		Now:      time.Now,
	}.Execute(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// read waits for the vault in the window to have been read through.
func (f *showing) read(t *testing.T) {
	t.Helper()

	for range 400 {
		if state := f.state(t); state.GetReady() {
			return
		} else if reason := state.GetFailed(); reason != "" {
			t.Fatalf("the vault could not be read: %s", reason)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the vault in the window was never read")
}

func (f *showing) state(t *testing.T) *v1.GetVaultStateResponse {
	t.Helper()

	out, err := f.client.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg
}

// named is the notes the window answers with under a name.
func (f *showing) named(t *testing.T, query string) []string {
	t.Helper()

	found, err := f.client.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: query}))
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(found.Msg.GetFound()))
	for _, one := range found.Msg.GetFound() {
		out = append(out, one.GetNote().GetPath())
	}
	return out
}

// TestTheWindowSaysWhichVaultItIsShowing. Two windows are open on one
// installation, so the identity of the vault in front of the person is the
// window's answer and not the list's.
func TestTheWindowSaysWhichVaultItIsShowing(t *testing.T) {
	f := swapping(t)

	if got := f.shows(t, wire.Editor); got != string(f.first.ID) {
		t.Errorf("the window says it is showing %q, want %s", got, string(f.first.ID))
	}

	if err := f.opened.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	f.read(t)

	if got := f.shows(t, wire.Editor); got != string(f.second.ID) {
		t.Errorf("after the swap it says %q, want %s", got, string(f.second.ID))
	}

	// And the question is the window's, so another window's name reaches
	// nothing here.
	_, err := f.drawn.GetShownVault(t.Context(),
		connect.NewRequest(&v1.GetShownVaultRequest{Window: wire.Review}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("err = %v, want the question about another window turned away", err)
	}
}

// shows is the identity of the vault the named window says it is showing.
func (f *showing) shows(t *testing.T, window string) string {
	t.Helper()

	out, err := f.drawn.GetShownVault(t.Context(),
		connect.NewRequest(&v1.GetShownVaultRequest{Window: window}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg.GetVault()
}

// TestAnotherVaultOpensInTheWindowThatIsOpen. What the window answers about
// afterwards is the vault that arrived and nothing of the one that went, and
// every page is told that what it holds was read somewhere else.
func TestAnotherVaultOpensInTheWindowThatIsOpen(t *testing.T) {
	f := swapping(t)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := f.client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()
	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}

	told := make(chan bool, 1)
	go func() {
		for changes.Receive() {
			if changes.Msg().GetReload() {
				told <- true
				return
			}
		}
	}()

	if err := f.opened.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	f.read(t)

	if name := f.state(t).GetName(); name != "two" {
		t.Errorf("the window says it is showing %q", name)
	}
	if got := f.named(t, "Enthalpy"); len(got) == 0 {
		t.Error("the vault that arrived does not answer for the note it holds")
	}
	if got := f.named(t, "Entropy"); len(got) != 0 {
		t.Errorf("the vault that went still answers for %v", got)
	}

	// The settling shut the door on writes, and the vault that arrived is one
	// a person types in.
	if _, err := f.client.Write(t.Context(), connect.NewRequest(&v1.WriteRequest{
		Path: enthalpy,
		Body: "what a person typed in the vault that arrived\n",
	})); err != nil {
		t.Errorf("the vault that arrived cannot be written in: %v", err)
	}

	select {
	case <-told:
	case <-time.After(5 * time.Second):
		t.Error("no page was told that what it holds is from another vault")
	}

	// The next window opens on it.
	registry, err := f.cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	switch last, found, err := registry.Last(); {
	case err != nil:
		t.Fatal(err)
	case !found || last.ID != f.second.ID:
		t.Errorf("the list says the vault opened last is %+v", last)
	}
}

// TestTheWindowOpensTheVaultItShowedLast, so that a person who changed vault
// finds it there the next time.
func TestTheWindowOpensTheVaultItShowedLast(t *testing.T) {
	f := swapping(t)

	if err := f.opened.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	if err := f.opened.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := webui.Open(t.Context(), f.cfg, "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { again.Close() })

	if got := again.Showing(); got.ID != f.second.ID {
		t.Errorf("the window opened on %s, want the vault it showed last", got.Name)
	}
}

// TestTheVaultThatWentIsNoLongerFollowed. The watch and the passes behind a
// vault end with it, so what happens in that vault afterwards reaches nobody.
func TestTheVaultThatWentIsNoLongerFollowed(t *testing.T) {
	f := swapping(t)

	if err := f.opened.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	f.read(t)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := f.client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()
	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}

	reported := make(chan []string, 8)
	go func() {
		for changes.Receive() {
			if paths := changes.Msg().GetPaths(); len(paths) > 0 {
				reported <- paths
			}
		}
	}()

	write(t, f.first, entropy, noteNamed("Entropy, written again"))
	time.Sleep(200 * time.Millisecond)
	write(t, f.second, enthalpy, noteNamed("Enthalpy, written again"))

	until := time.After(5 * time.Second)
	for {
		select {
		case paths := <-reported:
			if slices.Contains(paths, entropy) {
				t.Fatalf("the vault that went is still being followed: %v", paths)
			}
			if slices.Contains(paths, enthalpy) {
				return
			}
		case <-until:
			t.Fatal("the vault in the window is not being followed")
		}
	}
}

func write(t *testing.T, v domain.Vault, path, body string) {
	t.Helper()

	at := filepath.Join(v.Path, filepath.FromSlash(path))
	if err := os.WriteFile(at, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTheVaultAlreadyShownIsNotOpenedAgain. Nothing is taken down and nothing
// is read again, and the window is where it was.
func TestTheVaultAlreadyShownIsNotOpenedAgain(t *testing.T) {
	f := swapping(t)

	if err := f.opened.Show(t.Context(), f.first); err != nil {
		t.Fatalf("the vault already shown was refused: %v", err)
	}
	if state := f.state(t); !state.GetReady() {
		t.Error("the vault already shown is being read again")
	}
	if got := f.named(t, "Entropy"); len(got) == 0 {
		t.Error("the window stopped answering for the vault it is showing")
	}
}

// TestAVaultThatCannotBeShownIsRefusedAndTheWindowStays. Nothing is taken away
// until the vault asked for is one the window can stand on.
func TestAVaultThatCannotBeShownIsRefusedAndTheWindowStays(t *testing.T) {
	for _, one := range []struct {
		name  string
		spoil func(t *testing.T, v domain.Vault)
	}{
		{"the folder is gone", func(t *testing.T, v domain.Vault) {
			if err := os.RemoveAll(v.Path); err != nil {
				t.Fatal(err)
			}
		}},
		{"the folder is no longer that vault", func(t *testing.T, v domain.Vault) {
			if err := os.RemoveAll(filepath.Join(v.Path, filesystem.DefaultServiceDir)); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(one.name, func(t *testing.T) {
			f := swapping(t)
			one.spoil(t, f.second)

			if err := f.opened.Show(t.Context(), f.second); err == nil {
				t.Fatal("the window opened a vault it cannot read")
			}
			state := f.state(t)
			if state.GetName() != "one" || !state.GetReady() {
				t.Errorf("the window is on %+v", state)
			}
			if got := f.named(t, "Entropy"); len(got) == 0 {
				t.Error("the window stopped answering for the vault it had")
			}
		})
	}
}

// listens opens a stream saying the window is going, and answers with the token
// this page will flush under.
func (f *showing) listens(t *testing.T) (*connect.ServerStreamForClient[v1.WatchQuitResponse], func()) {
	t.Helper()

	listening, hangUp := context.WithCancel(context.Background())
	stream, err := f.drawn.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{
		Window: wire.Editor,
	}))
	if err != nil {
		hangUp()
		t.Fatal(err)
	}
	// The token before anything is asked for.
	if !stream.Receive() {
		hangUp()
		t.Fatalf("the stream never opened: %v", stream.Err())
	}
	return stream, func() {
		stream.Close()
		hangUp()
	}
}

// TestAPageThatSaysNothingCostsTheSwapItsBound. A page whose script has stopped
// answers never, and the vault asked for arrives once the bound is spent.
func TestAPageThatSaysNothingCostsTheSwapItsBound(t *testing.T) {
	f := swapping(t)

	// A page that listens and answers nothing.
	_, done := f.listens(t)
	defer done()

	swapped := make(chan error, 1)
	go func() { swapped <- f.opened.Show(context.Background(), f.second) }()

	select {
	case err := <-swapped:
		if err != nil {
			t.Fatalf("the second vault would not open: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("a page that says nothing held the swap with nothing to wait for")
	}
	if got := f.opened.Showing(); got.ID != f.second.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

// TestAPageHoldingAnUnansweredQuestionCallsTheSwapOff. What a person is being
// asked about is theirs to answer, and the vault it was typed in stays in front
// of them.
func TestAPageHoldingAnUnansweredQuestionCallsTheSwapOff(t *testing.T) {
	f := swapping(t)

	stream, done := f.listens(t)
	defer done()

	asked := make(chan struct{})
	go func() {
		defer close(asked)
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			if _, err := f.drawn.ReportFlush(context.Background(), connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor,
				Token:  stream.Msg().GetToken(),
				Owed:   v1.Owed_OWED_ASKING,
			})); err != nil {
				t.Error(err)
			}
			return
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := f.opened.Show(ctx, f.second); err == nil {
		t.Fatal("a page holding work a person has to answer for did not call the swap off")
	}

	select {
	case <-asked:
	case <-time.After(5 * time.Second):
		t.Fatal("the page was never asked for what it holds")
	}
	if name := f.state(t).GetName(); name != "one" {
		t.Errorf("the window is showing %q", name)
	}
	if got := f.named(t, "Entropy"); len(got) == 0 {
		t.Error("the window stopped answering for the vault it had")
	}
}

// TestASwapAndACloseAskedForAtOnceDoNotCancelEachOther. One settling runs at a
// time. The second to arrive is refused, and the first runs to its end.
func TestASwapAndACloseAskedForAtOnceDoNotCancelEachOther(t *testing.T) {
	f := swapping(t)

	stream, done := f.listens(t)
	defer done()

	told := make(chan struct{})
	release := make(chan struct{})
	wrote := make(chan struct{})
	go func() {
		defer close(wrote)
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			close(told)
			<-release
			if _, err := f.drawn.ReportFlush(context.Background(), connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor,
				Token:  stream.Msg().GetToken(),
				Owed:   v1.Owed_OWED_NOTHING,
			})); err != nil {
				t.Error(err)
			}
			return
		}
	}()

	settled := make(chan bool, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		settled <- f.opened.Settle(ctx)
	}()

	select {
	case <-told:
	case <-time.After(5 * time.Second):
		t.Fatal("the close never asked the page for what it holds")
	}

	if err := f.opened.Show(t.Context(), f.second); err == nil {
		t.Fatal("a vault was opened while the window was settling to close")
	}
	close(release)

	select {
	case ok := <-settled:
		if !ok {
			t.Error("the close was called off by the vault asked for beside it")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the close never settled")
	}
	select {
	case <-wrote:
	case <-time.After(5 * time.Second):
		t.Fatal("the page never wrote what it holds")
	}
	if got := f.opened.Showing(); got.ID != f.first.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

// TestARunReachesTheVaultTheWindowIsShowing. The passes behind a vault are
// taken down and built again while requests are being served, so a handler that
// reads one reads it where the swap publishes it. A request that arrives in the
// middle is answered by a vault or refused, and never by a pass that stopped.
func TestARunReachesTheVaultTheWindowIsShowing(t *testing.T) {
	f := swapping(t)

	asking, stop := context.WithCancel(t.Context())
	var asked sync.WaitGroup
	for _, id := range []string{"asr.corrected", "ocr", "asr"} {
		asked.Add(1)
		go func() {
			defer asked.Done()
			for asking.Err() == nil {
				f.opened.API.CreateArtifact(asking, connect.NewRequest(&v1.CreateArtifactRequest{
					Path: entropy, ArtifactId: id,
				}))
			}
		}()
	}

	for range 4 {
		if err := f.opened.Show(t.Context(), f.second); err != nil {
			t.Errorf("the second vault would not open: %v", err)
			break
		}
		if err := f.opened.Show(t.Context(), f.first); err != nil {
			t.Errorf("the first vault would not open again: %v", err)
			break
		}
	}

	stop()
	asked.Wait()
}
