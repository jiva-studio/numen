// Package editor answers what a client may ask about a vault.
//
// The questions and their answers are the schema in modules/libs/protocol; this
// is the half that answers them. The interface asks over HTTP either way, so
// the same handler serves the window the application opens and a browser during
// development.
package editor

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/file"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// API is the vault a client is looking at and what can be asked about it, in
// the things a window works a vault as. Every field of every group is a use
// case or a query the rest of the application already has.
type API struct {
	// vault is the vault the window has open. It is replaced while requests are
	// being served, so every reader takes it through Showing.
	vault atomic.Pointer[domain.Vault]

	// closed is the door on every question. It is closed before the index and the
	// embedder an answer reaches into are taken away.
	closed atomic.Bool

	// questions are the questions taken and not yet answered. A search, a note
	// and a link are read straight from the index, and Close waits here until
	// the last of them is off it.
	//
	// A stream is not counted: it lives as long as the page that opened it.
	questions inflight

	// session is the passes behind the vault the window is showing. It is
	// published as one after the vault, so a run, a cut and a drop reach the
	// vault the request was answered over. Nothing while the vault is being
	// changed.
	session atomic.Pointer[vaultSession]

	ChangeEvents broadcaster[change]

	// OpenVault puts another vault in the window: the agents are stopped, the vault
	// is swapped, and the agents are started again against the one that
	// arrived. It is bound by every build that serves the list of vaults.
	OpenVault func(context.Context, domain.Vault) error

	// agent is the agent the panel's tasks go to. A vault without one answers
	// that it has none, and the rest of the window works as it did. It is
	// replaced while requests are being served, so it is taken through
	// Answering.
	agent atomic.Pointer[port.Agent]

	// Readers open the vault a document is drawn from and a folder is listed
	// out of. The notes, the documents, the recordings and the window are all
	// read out of them. A path from outside arrives at the vault through them,
	// and one leaving the vault is refused there.
	Readers port.VaultReaders
	// Viewer holds the documents the window has open and the pages it has
	// drawn, and Highlight says where a run of a source's text sits on the
	// pages it was read from. They are bound by every build that serves what a
	// file of the vault is.
	Viewer    *viewer
	Highlight *source.Highlight
	// Documents reads the pages and books of the vault through the viewer.
	Documents DocumentScenarios
	// Texts reads and writes what was made from a file of the vault.
	Texts source.Texts
	// Playing is the socket a recording is played from. A build without one
	// answers with no address, and the window says the recording cannot be
	// played here.
	Playing *Loopback
	// NotifyNoteWritten is what a save raises: the reading behind the window asks the index
	// what owes a vector, once the vault has been still. Nil for a build with
	// nothing reading behind it, and then a save changes no vectors.
	NotifyNoteWritten func()
	// Drops takes a recording's transcript away, with everything listening to
	// it produced. It is bound by every build that serves what is made from a
	// file.
	Drops *source.DropTranscript

	// Imports fetches what is at the address a link note points at. A build on
	// a machine holding neither of the tools that reach one binds none, and
	// asking for a fetch is answered that this build cannot do it.
	Imports *source.ImportURL

	// Presets is the preset a deck is scheduled by, and how one is read,
	// written and made. Curves is what the one control of a preset comes to
	// over the whole range of its goal. They are bound by every build that
	// serves the presets.
	Presets *flashcards.Presets
	Curves  *flashcards.ProjectCurve

	// Finds is how the window searches the text the vault holds, by the words
	// in it and by what it means. It is bound by every build that serves the
	// search, and the names a vault holds are answered beside it.
	Finds *search.Search

	// Edits is everyone drawing this vault, for a change to a note being made
	// while they may be showing it.
	Edits broadcaster[domain.Edit]

	// Places is everyone drawing this vault, for when something asks that a
	// place be put in front of the person.
	Places broadcaster[domain.Place]

	// openTabs is what the person has open, as the window last said. It is
	// replaced while requests are being served, so every reader takes it
	// through Attended.
	openTabs atomic.Pointer[domain.OpenTabs]

	// NotifyOpenTabs hears what the person has open each time the window says it,
	// once what it said stands. A build without one takes the report and tells
	// nobody.
	NotifyOpenTabs func(domain.OpenTabs)

	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Ready is set when the scan finished, and Error says why it could not.
	Ready atomic.Bool
	Error wire.Reason
	// WatchError is why the vault is not being followed, when it is not.
	WatchError wire.Reason
	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable wire.Reason

	// Window is this window itself: everything being done behind it, which
	// whatever does work puts itself into, and everyone drawing it for the
	// moment it goes.
	Window *wire.Window

	// Themes are the stylesheets the window may be dressed in. They belong to
	// the installation, so they arrive here from whatever put the window
	// together. A build put together without a catalogue serves no ThemeService.
	Themes numenv1connect.ThemeServiceHandler

	Notes    Notes
	Files    Files
	Vaults   Vaults
	Cards    Cards
	Settings Settings
	Indexing IndexState
}

// Notes is a vault's notes: what is asked of them, and what changes them.
//
// Read and Write are how the window opens a note and puts it back, Create makes
// one and EditLinks writes a relationship into one, Rename gives one a different
// name and Remove takes it out of the vault. They are bound by every build that
// serves a note.
type Notes struct {
	Queries port.NoteQueries
	Links   port.LinkQueries

	Read              *note.Read
	Write             *note.Write
	Create            *note.Create
	EditLinks         *note.EditLinks
	Rename            *note.Rename
	Remove            *note.Remove
	ShowNeighbourhood *note.ShowNeighbourhood
	GetTitle          *note.GetTitle
	GetLinkAddress    *note.GetLinkAddress
}

// Files is the vault's tree: what a folder holds, and what a person does to it.
type Files struct {
	GetFolderEntries *file.GetFolderEntries
	GetFileKinds     *file.GetFileKinds
	CreateFolder     *file.CreateFolder
	Move             *vaults.Move
	Import           *vaults.Import
	// URLs makes the file a web address is kept in.
	URLs *source.CreateURL
}

// Vaults is the list of vaults this installation holds and what a person does to it.
type Vaults struct {
	Registry port.VaultRegistry

	List        *vaults.List
	Find        *vaults.Find
	FolderCheck *vaults.FolderCheck
	Add         *vaults.Add
	Rename      *vaults.Rename
	Remove      *vaults.Remove
	Unregister  *vaults.Unregister

	// ChooseFolder is nil where the application has no window to put a dialog in.
	ChooseFolder *vaults.ChooseFolder
}

// SetFolderDialog puts the machine's folder dialog in front of the person when a
// client asks for one.
func (v *Vaults) SetFolderDialog(dialog port.FolderDialog) {
	choose := vaults.NewChooseFolder(dialog)
	v.ChooseFolder = &choose
}

// GetPickFolder is the use case that lets an agent put the same dialog in front
// of the person, or nil where the window has none.
func (v *Vaults) GetPickFolder() *vaults.PickFolder {
	if v.ChooseFolder == nil {
		return nil
	}
	pick := v.ChooseFolder.GetPickFolder()
	return &pick
}

// Cards is the decks and stencils a vault is arranged into. Read takes a deck
// or a stencil, List the stencils the vault holds, and Write puts either back.
// Create makes a deck, a stencil or a preset, and RenameField gives one of a
// stencil's fields a different name everywhere it is written. They are bound by
// every build that serves the cards.
type Cards struct {
	Read        *cards.Read
	List        *cards.List
	Write       *cards.Write
	Create      *cards.Create
	RenameField *cards.RenameField
}

// Settings is every setting read and written, and the settings file read and replaced whole.
type Settings struct {
	Get       *settings.Get
	Write     *settings.Write
	GetFile   *settings.GetFile
	WriteFile *settings.WriteFile
}

// IndexState is how far the vault has been read for meaning.
type IndexState struct {
	// Progress answers how far cutting and embedding have got. Nil for a vault
	// nothing is reading for meaning, and the window then says nothing about it.
	Progress port.IndexProgress
	// Model is the identity vectors are being made under, which is what decides
	// whether a chunk already carries one. Empty for an installation with no
	// model, and then nothing is going to embed anything. It is named by the
	// goroutine reading the vault and asked for by every request.
	Model atomic.Value
	// Recipe is everything that decides what a vector is, which is what a
	// vector is found by.
	Recipe atomic.Value
}

// GetCurrentVault is the vault the window has open. A window standing on nothing
// answers with no vault at all.
func (a *API) GetCurrentVault() domain.Vault {
	if v := a.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

// setVault puts a vault in front of whoever asks from now on.
func (a *API) setVault(v domain.Vault) { a.vault.Store(&v) }

// setSession sets the session a run, a rechunk and a drop are taken through from now
// on. They arrive together, after the vault they belong to.
func (a *API) setSession(on *vaultSession) { a.session.Store(on) }

// getRecogniser reads a scanned document and getTranscriber hears a recording,
// each for whoever asks. They are the jobs an agent asks through too, so what a
// person started in the window is shown to both. Nothing where the window has
// no vault, and where this build does no such run.
func (a *API) getRecogniser() Runner {
	if on := a.session.Load(); on != nil {
		return on.recognitionRunner
	}
	return nil
}

func (a *API) getTranscriber() Runner {
	if on := a.session.Load(); on != nil {
		return on.transcriptionRunner
	}
	return nil
}

// getProofreader puts a recording's transcript right, for whoever asks.
func (a *API) getProofreader() Proofreader {
	if on := a.session.Load(); on != nil {
		return on.transcriptProofreader
	}
	return nil
}

// getRecognitionProofreader puts a document's recognition right, for whoever asks.
func (a *API) getRecognitionProofreader() RecognitionProofreader {
	if on := a.session.Load(); on != nil {
		return on.proofreadsRecognition
	}
	return nil
}

// getRechunker asks for a source to be cut again from whatever its text now says. A
// window that put a transcript right calls it, so search answers with the words
// as they now read. Nothing while the window has no vault, and then a
// correction is baseline in the tab alone.
func (a *API) getRechunker() func(context.Context, domain.Vault, string) error {
	if on := a.session.Load(); on != nil {
		return on.rechunk
	}
	return nil
}

// getRecordingForgetter takes a recording out of what the queue behind the vault has
// already had an answer about, so one that gave no words is offered again.
func (a *API) getRecordingForgetter() func(domain.Vault, string) {
	if on := a.session.Load(); on != nil {
		return on.forgetRecording
	}
	return nil
}

// reportTask puts one piece of work in the list of what is being done behind the
// window, for a build that keeps one.
func (a *API) reportTask(at task.Task) { a.Window.AddTask(at) }

// finishTask takes one piece of work out of that list.
func (a *API) finishTask(id string) { a.Window.RemoveTask(id) }

// isUnlevelled says whether a write reached the vault and the index did not
// follow.
// The answer goes on the wire, so the client that saved knows search has not
// caught up with what it saved, and it stands in the list of what is being done
// until a write levels the index again, so a person who is not looking at that
// note is told too.
func (a *API) isUnlevelled(err error) bool {
	if err == nil {
		a.finishTask(levellingTheIndex)
		return false
	}
	if !errors.Is(err, note.ErrUnlevelled) {
		return false
	}
	a.reportTask(task.Task{ID: levellingTheIndex, Title: "Bringing the index level", Error: err.Error()})
	return true
}

// Close refuses every question from now on, and there is no opening it again. It
// answers once the questions already taken have been answered, so everything an
// answer reaches into is still there for the whole of it.
func (a *API) Close() {
	a.closed.Store(true)
	<-a.questions.seal()
}

func (a *API) isClosed() bool { return a.closed.Load() }

// getCurrentVault is the vault a question is answered over. A window standing on nothing
// has none, and every question that would reach into a vault is refused there.
func (a *API) getCurrentVault() (domain.Vault, error) {
	v := a.GetCurrentVault()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return v, nil
}

// GetAgent is the agent the panel's tasks go to, and nothing where the vault
// has none.
func (a *API) GetAgent() port.Agent {
	if taking := a.agent.Load(); taking != nil {
		return *taking
	}
	return nil
}

// SetAgent names who takes the panel's tasks from now on. Nil leaves the vault
// with no agent.
func (a *API) SetAgent(taking port.Agent) {
	a.agent.Store(&taking)
}

func text(v *atomic.Value) string {
	s, _ := v.Load().(string)
	return s
}

func (a *API) GetVaultState(
	ctx context.Context, _ *connect.Request[v1.GetVaultStateRequest],
) (*connect.Response[v1.GetVaultStateResponse], error) {
	showing := a.GetCurrentVault()
	out := &v1.GetVaultStateResponse{
		Id:   string(showing.ID),
		Name: showing.Name,
		Path: showing.Path,
		Scan: &v1.Scan{
			IsReady:    a.Ready.Load(),
			Error:      a.Error.Why(),
			WatchError: a.WatchError.Why(),
		},
		Coverage: &v1.IndexCoverage{IsEmbedding: text(&a.Indexing.Model) != ""},
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered as it stands. A window standing on nothing holds no
	// chunks and counts none.
	if a.Indexing.Progress != nil && showing.ID != "" {
		if held, embedded, err := a.Indexing.Progress.GetProgress(ctx, showing.ID, text(&a.Indexing.Recipe)); err == nil {
			out.Coverage.ChunkCount, out.Coverage.EmbeddedCount = held, embedded
		}
	}
	return connect.NewResponse(out), nil
}

// WatchVaultChanges reports what moved, for as long as the caller listens.
func (a *API) WatchVaultChanges(
	ctx context.Context,
	_ *connect.Request[v1.WatchVaultChangesRequest],
	out *connect.ServerStream[v1.WatchVaultChangesResponse],
) error {
	line, done := a.ChangeEvents.subscribe()
	defer done()

	// Named as following before anything has changed. A stream that says
	// nothing until the vault moves is indistinguishable from one that never
	// opened, and a caller waiting on its first message waits for an edit.
	if err := out.Send(&v1.WatchVaultChangesResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := out.Send(&v1.WatchVaultChangesResponse{}); err != nil {
				return err
			}
		case what, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(newChangeResponse(what)); err != nil {
				return err
			}
		}
	}
}

// newChangeResponse is the wire message that reports one change.
func newChangeResponse(what change) *v1.WatchVaultChangesResponse {
	renamed := make([]*v1.Move, 0, len(what.renamed))
	for _, went := range what.renamed {
		renamed = append(renamed, &v1.Move{From: went.From, To: went.To})
	}
	return &v1.WatchVaultChangesResponse{Paths: what.paths, Reload: what.shouldReload, Renamed: renamed}
}
