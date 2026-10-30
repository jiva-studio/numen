package editor_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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
	canvas       numenv1connect.WindowServiceClient
	installation *editor.Installation
	cfg          container.Config
	first        domain.Vault
	second       domain.Vault
}

// openTwoVaults opens a window on the first of two vaults, both on the list this
// installation keeps.
func openTwoVaults(t *testing.T) *showing {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	first := newListedVault(t, cfg, registry, "one", map[string]string{entropy: noteNamed("Entropy")})
	second := newListedVault(t, cfg, registry, "two", map[string]string{enthalpy: noteNamed("Enthalpy")})

	installation, err := editor.Open(t.Context(), editor.NewAssembly(t, cfg), "one", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)

	going, itself := numenv1connect.NewWindowServiceHandler(installation.State.Window)
	mux := http.NewServeMux()
	mountQuestionsServices(mux, installation.Services)
	mux.Handle(going, itself)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	f := &showing{
		client:       newQuestionsClient(server.Client(), server.URL),
		canvas:       numenv1connect.NewWindowServiceClient(server.Client(), server.URL),
		installation: installation,
		cfg:          cfg,
		first:        first,
		second:       second,
	}
	f.read(t)
	return f
}

// newListedVault writes a folder of notes, makes it a vault and puts it on the list.
func newListedVault(
	t *testing.T,
	cfg container.Config,
	registry vaultport.VaultRegistry,
	name string,
	notes map[string]string,
) domain.Vault {
	t.Helper()

	root := t.TempDir()
	for item, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(item))
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
	v, err := vaults.NewAdd(cfg.VaultIdentity(), registry, time.Now).Execute(root, name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// read waits for the vault in the window to have been read through.
func (f *showing) read(t *testing.T) {
	t.Helper()

	for range 400 {
		if state := f.state(t); state.GetScan().GetIsReady() {
			return
		} else if reason := state.GetScan().GetError(); reason != "" {
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

// getFoundNames is the notes the window answers with under a name.
func (f *showing) getFoundNames(t *testing.T, query string) []string {
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
func TestTheWindowReturnsWhichVaultItIsShowing(t *testing.T) {
	f := openTwoVaults(t)

	if got := f.getShownVault(t, wire.Editor); got != string(f.first.ID) {
		t.Errorf("the window says it is showing %q, want %s", got, string(f.first.ID))
	}

	if err := f.installation.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	f.read(t)

	if got := f.getShownVault(t, wire.Editor); got != string(f.second.ID) {
		t.Errorf("after the swap it says %q, want %s", got, string(f.second.ID))
	}

	// And the question is the window's, so another window's name reaches
	// nothing here.
	_, err := f.canvas.GetCurrentVault(t.Context(),
		connect.NewRequest(&v1.GetCurrentVaultRequest{Window: "another"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("err = %v, want the question about another window turned away", err)
	}
}

// getShownVault returns the identity of the vault the named window says it is showing.
func (f *showing) getShownVault(t *testing.T, window string) string {
	t.Helper()

	out, err := f.canvas.GetCurrentVault(t.Context(),
		connect.NewRequest(&v1.GetCurrentVaultRequest{Window: window}))
	if err != nil {
		t.Fatal(err)
	}
	return out.Msg.GetVault()
}

// TestAnotherVaultOpensInTheWindowThatIsOpen. What the window answers about
// afterwards is the vault that arrived and nothing of the one that went, and
// every page is told that what it holds was read somewhere else.
func TestAnotherVaultOpensInTheWindowThatIsOpen(t *testing.T) {
	f := openTwoVaults(t)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := f.client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, changes)
	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}

	signalCh := make(chan bool, 1)
	go func() {
		for changes.Receive() {
			if changes.Msg().GetReload() {
				signalCh <- true
				return
			}
		}
	}()

	if err := f.installation.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	f.read(t)

	if name := f.state(t).GetName(); name != "two" {
		t.Errorf("the window says it is showing %q", name)
	}
	if got := f.getFoundNames(t, "Enthalpy"); len(got) == 0 {
		t.Error("the vault that arrived does not answer for the note it holds")
	}
	if got := f.getFoundNames(t, "Entropy"); len(got) != 0 {
		t.Errorf("the vault that went still answers for %v", got)
	}

	// The settling shut the door on writes, and the vault that arrived is one
	// a person types in.
	if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: enthalpy,
		Body: "what a person typed in the vault that arrived\n",
	})); err != nil {
		t.Errorf("the vault that arrived cannot be written in: %v", err)
	}

	select {
	case <-signalCh:
	case <-time.After(5 * time.Second):
		t.Error("no page received that what it holds is from another vault")
	}

	// The next window opens on it.
	registry, err := f.cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	switch last, found, err := registry.GetLastVault(); {
	case err != nil:
		t.Fatal(err)
	case !found || last.ID != f.second.ID:
		t.Errorf("the list says the vault opened last is %+v", last)
	}
}

// TestTheWindowOpensTheVaultItShowedLast, so that a person who changed vault
// finds it there the next time.
func TestTheWindowOpensTheVaultItShowedLast(t *testing.T) {
	f := openTwoVaults(t)

	if err := f.installation.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	if err := f.installation.Close(); err != nil {
		t.Fatal(err)
	}

	again, err := editor.Open(t.Context(), editor.NewAssembly(t, f.cfg), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, again)

	if got := again.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window opened on %s, want the vault it showed last", got.Name)
	}
}

// TestTheVaultThatWentIsNoLongerFollowed. The watch and the passes behind a
// vault end with it, so what happens in that vault afterwards reaches nobody.
func TestTheVaultThatWentIsNoLongerFollowed(t *testing.T) {
	f := openTwoVaults(t)

	if err := f.installation.Show(t.Context(), f.second); err != nil {
		t.Fatalf("the second vault would not open: %v", err)
	}
	f.read(t)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := f.client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, changes)
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

	pos := filepath.Join(v.Path, filepath.FromSlash(path))
	if err := os.WriteFile(pos, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTheVaultAlreadyShownIsNotOpenedAgain. Nothing is taken down and nothing
// is read again, and the window is where it was.
func TestTheVaultAlreadyShownIsNotOpenedAgain(t *testing.T) {
	f := openTwoVaults(t)

	if err := f.installation.Show(t.Context(), f.first); err != nil {
		t.Fatalf("the vault already shown failed: %v", err)
	}
	if state := f.state(t); !state.GetScan().GetIsReady() {
		t.Error("the vault already shown is being read again")
	}
	if got := f.getFoundNames(t, "Entropy"); len(got) == 0 {
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
			if err := os.RemoveAll(filepath.Join(v.Path, vaultconfig.DefaultServiceDir)); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(one.name, func(t *testing.T) {
			f := openTwoVaults(t)
			one.spoil(t, f.second)

			if err := f.installation.Show(t.Context(), f.second); err == nil {
				t.Fatal("the window opened a vault it cannot read")
			}
			state := f.state(t)
			if state.GetName() != "one" || !state.GetScan().GetIsReady() {
				t.Errorf("the window is on %+v", state)
			}
			if got := f.getFoundNames(t, "Entropy"); len(got) == 0 {
				t.Error("the window stopped answering for the vault it had")
			}
		})
	}
}

// watchQuit opens a stream saying the window is going, and answers with the token
// this page will flush under.
func (f *showing) watchQuit(t *testing.T) (*connect.ServerStreamForClient[v1.WatchQuitResponse], func()) {
	t.Helper()

	listening, hangUp := context.WithCancel(t.Context())
	stream, err := f.canvas.WatchQuit(listening, connect.NewRequest(&v1.WatchQuitRequest{
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
		testsupport.Close(t, stream)
		hangUp()
	}
}

// TestAPageThatSaysNothingCostsTheSwapItsBound. A page whose script has stopped
// answers never, and the vault asked for arrives once the bound is spent.
func TestAPageThatReturnsNothingCostsTheSwapItsBound(t *testing.T) {
	f := openTwoVaults(t)

	// A page that listens and answers nothing.
	_, isDone := f.watchQuit(t)
	defer isDone()

	swapped := make(chan error, 1)
	go func() { swapped <- f.installation.Show(t.Context(), f.second) }()

	select {
	case err := <-swapped:
		if err != nil {
			t.Fatalf("the second vault would not open: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("a page that says nothing held the swap with nothing to wait for")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

// TestAPageHoldingAnUnansweredQuestionCallsTheSwapOff. What a person is being
// asked about is theirs to answer, and the vault it was typed in stays in front
// of them.
func TestAPageContainingAnUnansweredQuestionCallsTheSwapOff(t *testing.T) {
	f := openTwoVaults(t)

	stream, isDone := f.watchQuit(t)
	defer isDone()

	request := make(chan struct{})
	go func() {
		defer close(request)
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			if _, err := f.canvas.ReportFlush(t.Context(), connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor,
				Token:  stream.Msg().GetToken(),
				Result: v1.FlushResult_FLUSH_RESULT_ASKING,
			})); err != nil {
				t.Error(err)
			}
			return
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if err := f.installation.Show(ctx, f.second); err == nil {
		t.Fatal("a page holding work a person has to answer for did not call the swap off")
	}

	select {
	case <-request:
	case <-time.After(5 * time.Second):
		t.Fatal("the page was never asked for what it holds")
	}
	if name := f.state(t).GetName(); name != "one" {
		t.Errorf("the window is showing %q", name)
	}
	if got := f.getFoundNames(t, "Entropy"); len(got) == 0 {
		t.Error("the window stopped answering for the vault it had")
	}
}

// TestASwapAndACloseAskedForAtOnceDoNotCancelEachOther. One settling runs at a
// time. The second to arrive fails, and the first runs to its end.
func TestASwapAndACloseAskedForAtOnceDoNotCancelEachOther(t *testing.T) {
	f := openTwoVaults(t)

	stream, isDone := f.watchQuit(t)
	defer isDone()

	signalCh := make(chan struct{})
	release := make(chan struct{})
	wrote := make(chan struct{})
	go func() {
		defer close(wrote)
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			close(signalCh)
			<-release
			if _, err := f.canvas.ReportFlush(t.Context(), connect.NewRequest(&v1.ReportFlushRequest{
				Window: wire.Editor,
				Token:  stream.Msg().GetToken(),
				Result: v1.FlushResult_FLUSH_RESULT_NOTHING,
			})); err != nil {
				t.Error(err)
			}
			return
		}
	}()

	resolved := make(chan bool, 1)
	go func() {
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
		defer cancel()
		resolved <- f.installation.FlushPending(ctx)
	}()

	select {
	case <-signalCh:
	case <-time.After(5 * time.Second):
		t.Fatal("the close never asked the page for what it holds")
	}

	if err := f.installation.Show(t.Context(), f.second); err == nil {
		t.Fatal("a vault was opened while the window was settling to close")
	}
	close(release)

	select {
	case ok := <-resolved:
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
	if got := f.installation.GetCurrentVault(); got.ID != f.first.ID {
		t.Errorf("the window is showing %s", got.Name)
	}
}

// Two processes write the index, so a write that reached the vault may find the
// the index is behind: the file is on disk and search does not hold what it now
// says.

// refuseToLevel is an index that will not come level with what was written.
func refuseToLevel(context.Context, domain.Vault, []string) error {
	return errors.New("the index is held open elsewhere")
}

// unwritten is a vault with those notes in it and an index that will not come level with anything
// written to it, asked the way the editor's window does.
func unwritten(t *testing.T, notes map[string]string) *going {
	t.Helper()

	f := openWindow(t, nil, notes)
	waitForScan(t, f)
	*f.installation.Services.NoteWrite() = f.installation.Services.NoteWrite().WithIndex(refuseToLevel)
	return f
}

func TestASaveTheIndexWouldNotComeLevelWithIsAnswered(t *testing.T) {
	f := unwritten(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	answer, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("saving answered %+v", answer.Msg)
	}
	// Without it the next save has nothing to present and is answered as a note
	// somebody else touched.
	if answer.Msg.GetAt() == nil {
		t.Error("the save landed and the client was handed no fingerprint")
	}
	if !answer.Msg.GetIsUnlevelled() {
		t.Error("the save is answered as findable and search does not hold it")
	}
	if target := onDisk(t, f.root, "Note.md"); !strings.Contains(target, "what the person typed") {
		t.Errorf("the note on disk is %q", target)
	}
}

// TestASaveTheIndexCameLevelWithSaysNothingIsBehind is the control. A flag
// every save carries says nothing, and a client shown a warning after every
// save learns to ignore it.
func TestASaveTheIndexCameLevelWithReturnsNothingIsPrior(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	waitForScan(t, f)

	answer, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetIsUnlevelled() {
		t.Error("a save the index came level with is answered as behind")
	}
	if pos := findLevellingTask(t, f); pos != nil {
		t.Errorf("a save that went through is in the list as %+v", pos)
	}
}

// TestASaveTheIndexWouldNotComeLevelWithStandsInTheList. A person saving a
// second note has no reason to look at what the first one answered, and the
// list of what is being done is where they are.
func TestASaveTheIndexWouldNotComeLevelWithExistsInTheList(t *testing.T) {
	f := unwritten(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	save := func() {
		t.Helper()
		if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
			Path: "Note.md",
			Body: "what the person typed\n",
		})); err != nil {
			t.Fatal(err)
		}
	}

	save()
	pos := findLevellingTask(t, f)
	if pos == nil || pos.GetError() == "" {
		t.Fatalf("what a person is shown after a save search cannot see is %+v", pos)
	}

	// The index is behind until something puts it right, and the save that does
	// is what says so. A list nothing leaves is a list nobody reads.
	*f.installation.Services.NoteWrite() = f.installation.Services.NoteWrite().WithIndex(func(context.Context, domain.Vault, []string) error { return nil })
	save()
	if offset := findLevellingTask(t, f); offset != nil {
		t.Errorf("the index came level and a person is still shown %+v", offset)
	}
}

// findLevellingTask is what the list of what is being done says about the index, asked
// for the way the window asks for it. The first message of the stream is the
// list currently, so nothing here waits on anything.
func findLevellingTask(t *testing.T, f *going) *v1.Task {
	t.Helper()

	listening, cancel := context.WithCancel(t.Context())
	defer cancel()
	tasks, err := f.canvas.WatchTasks(listening, connect.NewRequest(&v1.WatchTasksRequest{
		Window: wire.Editor,
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, tasks)
	if !tasks.Receive() {
		t.Fatalf("the tasks stream never opened: %v", tasks.Err())
	}
	for _, item := range tasks.Msg().GetTasks() {
		if item.GetId() == "levelling the index" {
			return item
		}
	}
	return nil
}

// TestANoteMadeWhenTheIndexWouldNotComeLevelIsAnswered. A note is made through
// its own use case, and it reports the same thing a save does.
func TestANoteMadeWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	f := openWindow(t, nil, nil)
	waitForScan(t, f)
	*f.installation.Services.NoteCreate() = f.installation.Services.NoteCreate().WithIndex(refuseToLevel)

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
	}))
	if err != nil {
		t.Fatalf("the note is on disk and the client received %v", err)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note is filed at %q", path)
	}
	if !answer.Msg.GetIsUnlevelled() {
		t.Error("the note is answered as findable and search does not hold it")
	}
}

// TestAFileRemovedWhenTheIndexWouldNotComeLevelIsAnswered. The file is in the
// trash and search still answers about it, and what its links now reach is
// reported all the same.
func TestAFileRemovedWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
		"Entropy.md": "---\ntitle: Entropy\nlinks:\n  - to: Ontology\n    role: parent\n---\n\n" +
			"# Entropy\n",
	})
	waitForScan(t, f)
	*f.installation.Services.FileRemove() = f.installation.Services.FileRemove().WithIndex(refuseToLevel)

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Ontology.md",
	}))
	if err != nil {
		t.Fatalf("the note is in the trash and the client received %v", err)
	}
	if answer.Msg.GetTrashed() == "" {
		t.Error("the note went to the trash and the client received nothing of where")
	}
	if !answer.Msg.GetIsUnlevelled() {
		t.Error("the removal is answered as followed and search still holds the note")
	}
	if len(answer.Msg.GetDangling()) != 1 {
		t.Errorf("the links now reaching nothing are %q", answer.Msg.GetDangling())
	}
}

// openInstallation is a window put together the way a person's window is:
// through the composition root, on a vault of this test's own.
func openInstallation(t *testing.T) *editor.Installation {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	add := vaults.NewAdd(cfg.VaultIdentity(), registry, time.Now)
	if _, err := add.Execute(root, "one"); err != nil {
		t.Fatal(err)
	}

	installation, err := editor.Open(t.Context(), editor.NewAssembly(t, cfg), "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, installation)
	return installation
}

// A note saved through the window is in the index when the save is answered.
func TestANoteSavedThroughTheWindowIsFindableAtOnce(t *testing.T) {
	installation := openInstallation(t)

	const body = "tetragrammaton is a word nothing else in this vault holds"
	content, err := installation.Services.Note.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Kept.md",
		Body: body,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if rejected := content.Msg.GetError(); rejected != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the save failed: %v", rejected)
	}

	found, err := installation.Passages().FindLexical(
		t.Context(), installation.GetCurrentVault().ID, "tetragrammaton",
		[]domain.SourceKind{domain.KindNote}, 10, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) == 0 {
		t.Error("the note saved through the window is not found")
	}
	for _, p := range found {
		if p.Source != "Kept.md" {
			t.Errorf("the search answered with %s, and only Kept.md holds the word", p.Source)
		}
	}
}

// nothing is an installation holding no vault, the window it opens, and a
// client asking about it the way the window does.
type nothing struct {
	installation *editor.Installation
	vault        questions
	canvas       numenv1connect.WindowServiceClient
	holds        numenv1connect.VaultsServiceClient
	server       *httptest.Server
	cfg          container.Config
	elsewhere    string
}

// openEmptyWindow opens a window on an installation that has added nothing.
//
// Both the folder this system keeps documents in and the home it would fall
// back on are somewhere a test owns, so anything made for a person to write in
// is made where this can see it.
func openEmptyWindow(t *testing.T) *nothing {
	t.Helper()

	elsewhere := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DOCUMENTS_DIR", elsewhere)

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	installation, err := editor.Open(t.Context(), editor.NewAssembly(t, cfg), "", os.Stderr)
	if err != nil {
		t.Fatalf("an installation holding no vault would not open: %v", err)
	}
	testsupport.CloseAfter(t, installation)

	// The welcome screen's way into a vault: the list opens one in this window.
	installation.State.OpenVault = installation.Show

	server := httptest.NewUnstartedServer(installation.Services.NewHandler(http.NotFoundHandler()))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &nothing{
		installation: installation,
		vault:        newQuestionsClient(server.Client(), server.URL),
		canvas:       numenv1connect.NewWindowServiceClient(server.Client(), server.URL),
		holds:        numenv1connect.NewVaultsServiceClient(server.Client(), server.URL),
		server:       server,
		cfg:          cfg,
		elsewhere:    elsewhere,
	}
}

// TestAnInstallationHoldingNoVaultOpensAWindowStandingOnNothing, and makes
// nothing to stand on. A folder appearing in a person's documents is a folder
// they did not ask for.
func TestAnInstallationContainingNoVaultOpensAWindowActiveOnNothing(t *testing.T) {
	f := openEmptyWindow(t)

	if got := f.installation.GetCurrentVault(); got != (domain.Vault{}) {
		t.Errorf("the window opened on %+v, want no vault", got)
	}

	target, err := os.ReadDir(f.elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 0 {
		t.Errorf("%d things were made where this person keeps documents: %v", len(target), target)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), "Documents")); err == nil {
		t.Error("a documents folder was made in this person's home")
	}

	registry, err := f.cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	switch last, found, err := registry.GetLastVault(); {
	case err != nil:
		t.Fatal(err)
	case found:
		t.Errorf("the list says a vault was opened: %+v", last)
	}
}

// TestTheWindowStandingOnNothingAnswersWhatItAsksAsItOpens. These run before a
// person can do anything, and one of them refusing is a window that never comes
// up.
func TestTheWindowActiveOnNothingAnswersWhatItAsksAsItOpens(t *testing.T) {
	f := openEmptyWindow(t)

	state, err := f.vault.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
	if err != nil {
		t.Fatalf("the window cannot say what it is showing: %v", err)
	}
	if got := state.Msg; got.GetName() != "" || got.GetPath() != "" {
		t.Errorf("the window says it is showing %q at %q", got.GetName(), got.GetPath())
	}
	if !state.Msg.GetScan().GetIsReady() {
		t.Error("the window says it is still being read, and nothing is reading")
	}
	if reason := state.Msg.GetScan().GetError(); reason != "" {
		t.Errorf("the window says it could not be read: %s", reason)
	}
	if state.Msg.GetCoverage().GetChunkCount() != 0 || state.Msg.GetCoverage().GetEmbeddedCount() != 0 {
		t.Errorf("the window counted %d chunks and %d of them embedded",
			state.Msg.GetCoverage().GetChunkCount(), state.Msg.GetCoverage().GetEmbeddedCount())
	}

	opening, err := f.vault.GetOpeningNote(t.Context(), connect.NewRequest(&v1.GetOpeningNoteRequest{}))
	if err != nil {
		t.Fatalf("the window cannot say which note it opens on: %v", err)
	}
	if note := opening.Msg.GetNote(); note != nil {
		t.Errorf("the window opens on %q, and holds no vault to hold it", note.GetPath())
	}

	displayed, err := f.canvas.GetCurrentVault(t.Context(),
		connect.NewRequest(&v1.GetCurrentVaultRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatalf("the window cannot say which vault it is showing: %v", err)
	}
	if got := displayed.Msg.GetVault(); got != "" {
		t.Errorf("the window says it is showing %q", got)
	}

	items, err := f.holds.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if err != nil {
		t.Fatalf("the window cannot list the vaults this installation holds: %v", err)
	}
	if got := items.Msg.GetVaults(); len(got) != 0 {
		t.Errorf("the list holds %d vaults, and none were added", len(got))
	}
}

// TestTheWindowStandingOnNothingIsFollowedTheWayAnyWindowIs. The window listens
// to four streams as it opens and draws nothing until each has said its first
// word.
func TestTheWindowActiveOnNothingIsFollowedTheWayAnyWindowIs(t *testing.T) {
	f := openEmptyWindow(t)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := f.vault.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, changes)
	if !changes.Receive() {
		t.Fatalf("the changes stream never opened: %v", changes.Err())
	}

	editing, err := f.vault.WatchEdits(listening, connect.NewRequest(&v1.WatchEditsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, editing)
	if !editing.Receive() {
		t.Fatalf("the editing stream never opened: %v", editing.Err())
	}

	tasks, err := f.canvas.WatchTasks(listening, connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, tasks)
	if !tasks.Receive() {
		t.Fatalf("the tasks stream never opened: %v", tasks.Err())
	}
	if executing := tasks.Msg().GetTasks(); len(executing) != 0 {
		t.Errorf("a window standing on nothing is doing %+v", executing)
	}

	// Focus opens on the first place asked for, and a place reaches only a
	// stream already listening. So it is asked for over and over while the
	// stream opens, and the asking stops with the place that arrived.
	asking, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		for {
			select {
			case <-asking.Done():
				return
			case <-time.After(20 * time.Millisecond):
				if err := f.installation.State.GetWindow().FocusPlace(asking, domain.Place{Path: "Somewhere.md"}); err != nil && asking.Err() == nil {
					t.Error(err)
				}
			}
		}
	}()

	waiting, elapsed := context.WithTimeout(listening, 5*time.Second)
	defer elapsed()

	focus, err := f.vault.WatchFocus(waiting, connect.NewRequest(&v1.WatchFocusRequest{}))
	if err != nil {
		t.Fatalf("the focus stream never opened: %v", err)
	}
	defer testsupport.Close(t, focus)
	if !focus.Receive() {
		t.Fatalf("the focus stream said nothing: %v", focus.Err())
	}
	if pos := focus.Msg().GetPath(); pos != "Somewhere.md" {
		t.Errorf("the window was sent to %q", pos)
	}
}

// TestAWindowStandingOnNothingRefusesEveryQuestionAboutAVault.
//
// A vault with no identity is every vault at once in one database, and a vault
// with no path is whichever folder this process happens to be standing in.
func TestAWindowActiveOnNothingRefusesEveryQuestionAboutAVault(t *testing.T) {
	f := openEmptyWindow(t)

	request := map[string]func() error{
		"list a folder of the vault": func() error {
			_, err := f.vault.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{}))
			return err
		},
		"read a note": func() error {
			_, err := f.vault.ReadNote(t.Context(), connect.NewRequest(&v1.ReadNoteRequest{Path: "One.md"}))
			return err
		},
		"write a note": func() error {
			_, err := f.vault.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
				Path: "One.md", Body: "what nobody asked to keep\n",
			}))
			return err
		},
		"make a note": func() error {
			_, err := f.vault.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{Title: "One"}))
			return err
		},
		"join a note to another": func() error {
			_, err := f.vault.WriteLink(t.Context(), connect.NewRequest(&v1.WriteLinkRequest{
				Path: "One.md",
				Link: &v1.Link{To: "Two.md", Role: v1.Role_ROLE_JUMP},
			}))
			return err
		},
		"rename a note": func() error {
			_, err := f.vault.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: "One.md", Title: "Two",
			}))
			return err
		},
		"move a file": func() error {
			_, err := f.vault.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
				From: "One.md", To: "Two.md",
			}))
			return err
		},
		"remove a file": func() error {
			_, err := f.vault.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{Path: "One.md"}))
			return err
		},
		"make a folder": func() error {
			_, err := f.vault.CreateFolder(t.Context(), connect.NewRequest(&v1.CreateFolderRequest{
				Path: "somewhere",
			}))
			return err
		},
		"show what a note is joined to": func() error {
			_, err := f.vault.GetNeighbourhood(t.Context(), connect.NewRequest(&v1.GetNeighbourhoodRequest{
				Path: "One.md",
			}))
			return err
		},
	}

	for what, ask := range request {
		t.Run(what, func(t *testing.T) {
			err := ask()
			if err == nil {
				t.Fatal("the window answered, and it has no vault to answer about")
			}
			if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
				t.Errorf("the error is %v: %v", got, err)
			}
			if !strings.Contains(err.Error(), "no vault") {
				t.Errorf("the rejection does not say why: %v", err)
			}
		})
	}
}

// TestAWindowStandingOnNothingSearchesNothing. The palette is a keystroke away
// from a person who has added no vault, and typing in it turns up nothing.
func TestAWindowActiveOnNothingSearchesNothing(t *testing.T) {
	f := openEmptyWindow(t)

	named, err := f.vault.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "one"}))
	if err != nil {
		t.Fatalf("the palette cannot be typed in: %v", err)
	}
	if found := named.Msg.GetFound(); len(found) != 0 {
		t.Errorf("a window standing on nothing knows the names %+v", found)
	}

	found, err := f.vault.SearchPassages(t.Context(), connect.NewRequest(&v1.SearchPassagesRequest{
		Query: "one", Mode: v1.SearchMode_SEARCH_MODE_HYBRID,
	}))
	if err != nil {
		t.Fatalf("the vault cannot be searched: %v", err)
	}
	if passages := found.Msg.GetFound(); len(passages) != 0 {
		t.Errorf("a window standing on nothing holds the text %+v", passages)
	}
}

// TestAVaultAddedToAWindowStandingOnNothingIsShown, which is where the welcome
// screen's two ways in both end.
func TestAVaultAddedToAWindowActiveOnNothingIsShown(t *testing.T) {
	f := openEmptyWindow(t)

	root := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(root, "Entropy.md"), []byte(noteNamed("Entropy")), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultconfig.Initialize(root, vaultconfig.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}

	added, err := f.holds.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{
		Path: root, Name: "the first one",
	}))
	if err != nil {
		t.Fatalf("a folder could not be added: %v", err)
	}
	if rejected := added.Msg.Error; rejected != nil {
		t.Fatalf("the folder failed: %v", *rejected)
	}

	if _, err := f.holds.OpenVault(t.Context(), connect.NewRequest(&v1.OpenVaultRequest{
		Id: added.Msg.GetVault().GetId(),
	})); err != nil {
		t.Fatalf("the vault just added would not open: %v", err)
	}

	if got := string(f.installation.GetCurrentVault().ID); got != added.Msg.GetVault().GetId() {
		t.Fatalf("the window is showing %q, want the vault just added", got)
	}

	// The vault is read, and answers about the note it holds.
	for range 400 {
		state, err := f.vault.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if reason := state.Msg.GetScan().GetError(); reason != "" {
			t.Fatalf("the vault could not be read: %s", reason)
		}
		if state.Msg.GetScan().GetIsReady() {
			if got := state.Msg.GetName(); got != "the first one" {
				t.Errorf("the window says it is showing %q", got)
			}
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	named, err := f.vault.SearchNames(t.Context(), connect.NewRequest(&v1.SearchNamesRequest{Query: "Entropy"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(named.Msg.GetFound()) == 0 {
		t.Error("the vault that arrived does not answer for the note it holds")
	}

	// The next window opens on it.
	registry, err := f.cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	switch last, found, err := registry.GetLastVault(); {
	case err != nil:
		t.Fatal(err)
	case !found || string(last.ID) != added.Msg.GetVault().GetId():
		t.Errorf("the list says the vault opened last is %+v", last)
	}
}
