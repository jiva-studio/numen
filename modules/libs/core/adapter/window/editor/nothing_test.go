package editor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
)

// nothing is an installation holding no vault, the window it opens, and a
// client asking about it the way the window does.
type nothing struct {
	opened    *editor.Installation
	vault     questions
	drawn     numenv1connect.WindowServiceClient
	holds     numenv1connect.VaultsServiceClient
	server    *httptest.Server
	cfg       container.Config
	elsewhere string
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
	opened, err := editor.Open(t.Context(), cfg, "", os.Stderr)
	if err != nil {
		t.Fatalf("an installation holding no vault would not open: %v", err)
	}
	t.Cleanup(func() { opened.Close() })

	// The welcome screen's way into a vault: the list opens one in this window.
	opened.API.Opens = opened.Show

	server := httptest.NewUnstartedServer(opened.API.Serving(http.NotFoundHandler()))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return &nothing{
		opened:    opened,
		vault:     asks(server.Client(), server.URL),
		drawn:     numenv1connect.NewWindowServiceClient(server.Client(), server.URL),
		holds:     numenv1connect.NewVaultsServiceClient(server.Client(), server.URL),
		server:    server,
		cfg:       cfg,
		elsewhere: elsewhere,
	}
}

// TestAnInstallationHoldingNoVaultOpensAWindowStandingOnNothing, and makes
// nothing to stand on. A folder appearing in a person's documents is a folder
// they did not ask for.
func TestAnInstallationHoldingNoVaultOpensAWindowStandingOnNothing(t *testing.T) {
	f := openEmptyWindow(t)

	if got := f.opened.Showing(); got != (domain.Vault{}) {
		t.Errorf("the window opened on %+v, want no vault", got)
	}

	held, err := os.ReadDir(f.elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 0 {
		t.Errorf("%d things were made where this person keeps documents: %v", len(held), held)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), "Documents")); err == nil {
		t.Error("a documents folder was made in this person's home")
	}

	registry, err := f.cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	switch last, found, err := registry.Last(); {
	case err != nil:
		t.Fatal(err)
	case found:
		t.Errorf("the list says a vault was opened: %+v", last)
	}
}

// TestTheWindowStandingOnNothingAnswersWhatItAsksAsItOpens. These run before a
// person can do anything, and one of them refusing is a window that never comes
// up.
func TestTheWindowStandingOnNothingAnswersWhatItAsksAsItOpens(t *testing.T) {
	f := openEmptyWindow(t)

	state, err := f.vault.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
	if err != nil {
		t.Fatalf("the window cannot say what it is showing: %v", err)
	}
	if got := state.Msg; got.GetName() != "" || got.GetPath() != "" {
		t.Errorf("the window says it is showing %q at %q", got.GetName(), got.GetPath())
	}
	if !state.Msg.GetScan().GetReady() {
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

	shown, err := f.drawn.GetShownVault(t.Context(),
		connect.NewRequest(&v1.GetShownVaultRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatalf("the window cannot say which vault it is showing: %v", err)
	}
	if got := shown.Msg.GetVault(); got != "" {
		t.Errorf("the window says it is showing %q", got)
	}

	listed, err := f.holds.ListVaults(t.Context(), connect.NewRequest(&v1.ListVaultsRequest{}))
	if err != nil {
		t.Fatalf("the window cannot list the vaults this installation holds: %v", err)
	}
	if got := listed.Msg.GetVaults(); len(got) != 0 {
		t.Errorf("the list holds %d vaults, and none were added", len(got))
	}
}

// TestTheWindowStandingOnNothingIsFollowedTheWayAnyWindowIs. The window listens
// to four streams as it opens and draws nothing until each has said its first
// word.
func TestTheWindowStandingOnNothingIsFollowedTheWayAnyWindowIs(t *testing.T) {
	f := openEmptyWindow(t)

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := f.vault.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()
	if !changes.Receive() {
		t.Fatalf("the changes stream never opened: %v", changes.Err())
	}

	editing, err := f.vault.WatchEdits(listening, connect.NewRequest(&v1.WatchEditsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer editing.Close()
	if !editing.Receive() {
		t.Fatalf("the editing stream never opened: %v", editing.Err())
	}

	tasks, err := f.drawn.WatchTasks(listening, connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer tasks.Close()
	if !tasks.Receive() {
		t.Fatalf("the tasks stream never opened: %v", tasks.Err())
	}
	if doing := tasks.Msg().GetTasks(); len(doing) != 0 {
		t.Errorf("a window standing on nothing is doing %+v", doing)
	}

	// Focus opens on the first place asked for, and a place reaches only a
	// stream already listening. So it is asked for over and over while the
	// stream opens, and the asking stops with the place that arrived.
	asking, over := context.WithCancel(t.Context())
	defer over()
	go func() {
		for {
			select {
			case <-asking.Done():
				return
			case <-time.After(20 * time.Millisecond):
				_ = f.opened.API.Viewing().Focus(asking, domain.Place{Path: "Somewhere.md"})
			}
		}
	}()

	waiting, spent := context.WithTimeout(listening, 5*time.Second)
	defer spent()

	focus, err := f.vault.WatchFocus(waiting, connect.NewRequest(&v1.WatchFocusRequest{}))
	if err != nil {
		t.Fatalf("the focus stream never opened: %v", err)
	}
	defer focus.Close()
	if !focus.Receive() {
		t.Fatalf("the focus stream said nothing: %v", focus.Err())
	}
	if at := focus.Msg().GetPath(); at != "Somewhere.md" {
		t.Errorf("the window was sent to %q", at)
	}
}

// TestAWindowStandingOnNothingRefusesEveryQuestionAboutAVault.
//
// A vault with no identity is every vault at once in one database, and a vault
// with no path is whichever folder this process happens to be standing in.
func TestAWindowStandingOnNothingRefusesEveryQuestionAboutAVault(t *testing.T) {
	f := openEmptyWindow(t)

	asked := map[string]func() error{
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

	for what, ask := range asked {
		t.Run(what, func(t *testing.T) {
			err := ask()
			if err == nil {
				t.Fatal("the window answered, and it has no vault to answer about")
			}
			if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
				t.Errorf("the refusal is %v: %v", got, err)
			}
			if !strings.Contains(err.Error(), "no vault") {
				t.Errorf("the refusal does not say why: %v", err)
			}
		})
	}
}

// TestAWindowStandingOnNothingSearchesNothing. The palette is a keystroke away
// from a person who has added no vault, and typing in it turns up nothing.
func TestAWindowStandingOnNothingSearchesNothing(t *testing.T) {
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

// TestNoDocumentIsDrawnForAWindowStandingOnNothing. A document is addressed by
// its path in the vault, and there is no vault for a path to be in.
func TestNoDocumentIsDrawnForAWindowStandingOnNothing(t *testing.T) {
	f := openEmptyWindow(t)

	// A file of the folder this process is standing in, which is what a path
	// with no vault under it reaches. The name is a document's, so the ask is
	// one a reader would take.
	const file = "serve.pdf"

	answer, err := f.server.Client().Get(
		f.server.URL + "/assets/" + file + "/pages/0?wide=800&size=1&mtime=1")
	if err != nil {
		t.Fatal(err)
	}
	answer.Body.Close()
	if answer.StatusCode != http.StatusConflict {
		t.Errorf("a page was answered %s, and the window has no vault", answer.Status)
	}

	documents := numenv1connect.NewDocumentServiceClient(f.server.Client(), f.server.URL)
	recordings := numenv1connect.NewRecordingServiceClient(f.server.Client(), f.server.URL)
	readings := numenv1connect.NewOcrServiceClient(f.server.Client(), f.server.URL)
	asked := map[string]func() error{
		"what a document is": func() error {
			_, err := documents.GetDocument(t.Context(), connect.NewRequest(&v1.GetDocumentRequest{
				Path: file,
			}))
			return err
		},
		"what a recording is": func() error {
			_, err := recordings.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{
				Path: file,
			}))
			return err
		},
		"what a run of the text says": func() error {
			_, err := readings.ReadOcr(t.Context(), connect.NewRequest(&v1.ReadOcrRequest{
				Path:  file,
				Spans: []*v1.Span{{From: 0, To: 1}},
			}))
			return err
		},
	}
	for what, asking := range asked {
		if code := connect.CodeOf(asking()); code != connect.CodeFailedPrecondition {
			t.Errorf("%s was refused %s, and the window has no vault", what, code)
		}
	}
}

// TestAVaultAddedToAWindowStandingOnNothingIsShown, which is where the welcome
// screen's two ways in both end.
func TestAVaultAddedToAWindowStandingOnNothingIsShown(t *testing.T) {
	f := openEmptyWindow(t)

	root := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(root, "Entropy.md"), []byte(noteNamed("Entropy")), 0o644,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}

	added, err := f.holds.AddVault(t.Context(), connect.NewRequest(&v1.AddVaultRequest{
		Path: root, Name: "the first one",
	}))
	if err != nil {
		t.Fatalf("a folder could not be added: %v", err)
	}
	if refused := added.Msg.Error; refused != nil {
		t.Fatalf("the folder was refused: %v", *refused)
	}

	if _, err := f.holds.OpenVault(t.Context(), connect.NewRequest(&v1.OpenVaultRequest{
		Id: added.Msg.GetVault().GetId(),
	})); err != nil {
		t.Fatalf("the vault just added would not open: %v", err)
	}

	if got := string(f.opened.Showing().ID); got != added.Msg.GetVault().GetId() {
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
		if state.Msg.GetScan().GetReady() {
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
	switch last, found, err := registry.Last(); {
	case err != nil:
		t.Fatal(err)
	case !found || string(last.ID) != added.Msg.GetVault().GetId():
		t.Errorf("the list says the vault opened last is %+v", last)
	}
}
