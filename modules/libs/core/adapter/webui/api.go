// Package webui answers what a client may ask about a vault.
//
// The questions and their answers are the schema in modules/libs/protocol; this
// is the half that answers them. The interface asks over HTTP either way, so
// the same handler serves the window the application opens and a browser during
// development.
package webui

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
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
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

	// shut is the door on every question. It is closed before the index and the
	// embedder an answer reaches into are taken away.
	shut atomic.Bool

	// questions are the questions taken and not yet answered. A search, a note
	// and a link are read straight from the index, and Shut stands here until
	// the last of them is off it.
	//
	// A stream is not counted: it lives as long as the page that opened it.
	questions inflight

	// showing is the passes behind the vault the window is showing. It is
	// published as one after the vault, so a run, a cut and a drop reach the
	// vault the request was answered over. Nothing while the vault is being
	// changed.
	showing atomic.Pointer[passes]

	Listeners audience[change]

	// Opens puts another vault in the window: the agents are stopped, the vault
	// is swapped, and the agents are started again against the one that
	// arrived. It is bound by every build that serves the list of vaults.
	Opens func(context.Context, domain.Vault) error

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
	// Playing is the socket a recording is played from. A build without one
	// answers with no address, and the window says the recording cannot be
	// played here.
	Playing *Loopback
	// Wrote is what a save raises: the reading behind the window asks the index
	// what owes a vector, once the vault has been still. Nil for a build with
	// nothing reading behind it, and then a save changes no vectors.
	Wrote func()
	// Drops takes a recording's transcript away, with everything listening to
	// it produced. It is bound by every build that serves what is made from a
	// file.
	Drops *source.DropTranscript

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
	Edits audience[domain.Edit]

	// Places is everyone drawing this vault, for when something asks that a
	// place be put in front of the person.
	Places audience[domain.Place]

	// openTabs is what the person has open, as the window last said. It is
	// replaced while requests are being served, so every reader takes it
	// through Attended.
	openTabs atomic.Pointer[domain.OpenTabs]

	// Attends hears what the person has open each time the window says it,
	// once what it said stands. A build without one takes the report and tells
	// nobody.
	Attends func(domain.OpenTabs)

	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Ready is set when the scan finished, and Failed says why it could not —
	// a vault that could not be read is not an empty one, and the interface has
	// to be able to tell them apart.
	Ready  atomic.Bool
	Failed wire.Reason
	// Unwatched is why the vault is not being followed, when it is not.
	Unwatched wire.Reason
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

	Notes       Notes
	Files       Files
	Vaults      Vaults
	Cards       Cards
	Configuring Configuring
	Indexing    Indexing
}

// Notes is a vault's notes: what is asked of them, and what changes them.
//
// Read and Write are how the window opens a note and puts it back, Create makes
// one and Linking writes a relationship into one, Rename gives one a different
// name and Remove takes it out of the vault. They are bound by every build that
// serves a note.
type Notes struct {
	Queries port.NoteQueries
	Links   port.LinkQueries

	Read    *note.Read
	Write   *note.Write
	Create  *note.Create
	Linking *note.EditLinks
	Rename  *note.Rename
	Remove  *note.Remove
}

// Files is the vault's tree as a person moves things about in it. Writers open
// the vault a folder is made in, Move puts a file or a folder somewhere else,
// and Import copies files a person handed the window into a folder of the
// vault. They are bound by every build that serves the tree.
//
// What a folder holds is read through the API's own Readers, which everything
// else reads through too.
type Files struct {
	Writers port.VaultWriters
	Move    *vaults.Move
	Import  *vaults.Import
}

// Vaults is the list of vaults this installation holds, the one the window is
// showing among them.
//
// Add turns a folder into a vault, Rename is what a person calls one, and
// Forget and Erase take one off the list. They are bound by every build that
// serves the list, and so is the folder dialog below.
type Vaults struct {
	Registry port.VaultRegistry

	// FolderDialog puts this machine's own folder dialog in front of the person.
	// Only an application with a window has one.
	FolderDialog port.FolderDialog

	Add    *vaults.Add
	Rename *vaults.Rename
	Forget *vaults.Forget
	Erase  *vaults.Erase
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

// Configuring is every setting the window reads and writes. Each is a reader
// and the writer beside it, and they are bound by every build that serves the
// settings: the phone serves them to a socket answering any origin at all, so
// it mounts no service about the file the keys are written in.
type Configuring struct {
	// Configured reads every setting as JSON and the file it stands in, Models
	// the models the settings that name one can be set to, and ChoosesSetting
	// writes settings into that file.
	Configured     func() (string, string, error)
	Models         func() []port.Model
	ChoosesSetting func(written []port.Setting) error

	// ConfiguredFile reads that file as its person wrote it, and WritesFile
	// replaces it whole, presenting the file the caller last read.
	ConfiguredFile func() (string, string, error)
	WritesFile     func(written string, seen *string) error

	// PartsUnderANode is how many parts a node may be asked to hang, at each
	// end, and LatestDayStarts how late in the day a day of review may be made
	// to begin, on the clock as `HH:MM`. Each is refused outside, so the window
	// is told them rather than holding a second copy.
	PartsUnderANode Bounds
	LatestDayStarts string
}

// Bounds is how far a setting holding a number goes, at each end.
type Bounds struct{ Least, Most float64 }

// Indexing is how far the vault has been read for meaning.
type Indexing struct {
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

// Showing is the vault the window has open. A window standing on nothing
// answers with no vault at all.
func (a *API) Showing() domain.Vault {
	if v := a.vault.Load(); v != nil {
		return *v
	}
	return domain.Vault{}
}

// show puts a vault in front of whoever asks from now on.
func (a *API) show(v domain.Vault) { a.vault.Store(&v) }

// runs is the passes a run, a cut and a drop are taken through from now on.
// They arrive together, after the vault they belong to.
func (a *API) runs(on *passes) { a.showing.Store(on) }

// recognises reads a scanned document and transcribes hears a recording, each
// for whoever asks. They are the jobs an agent asks through too, so what a
// person started in the window is shown to both. Nothing where the window has
// no vault, and where this build does no such run.
func (a *API) recognises() Runner {
	if on := a.showing.Load(); on != nil {
		return on.recognises
	}
	return nil
}

func (a *API) transcribes() Runner {
	if on := a.showing.Load(); on != nil {
		return on.transcribes
	}
	return nil
}

// proofreads puts a recording's transcript right, for whoever asks.
func (a *API) proofreads() Proofreader {
	if on := a.showing.Load(); on != nil {
		return on.proofreads
	}
	return nil
}

// cuts asks for a source to be cut again from whatever its text now says. A
// window that put a transcript right calls it, so search answers with the words
// as they now read. Nothing while the window has no vault, and then a
// correction is seen in the tab alone.
func (a *API) cuts() func(context.Context, domain.Vault, string) error {
	if on := a.showing.Load(); on != nil {
		return on.cut
	}
	return nil
}

// forgets takes a recording out of what the queue behind the vault has already
// had an answer about, so one that gave no words is offered again.
func (a *API) forgets() func(domain.Vault, string) {
	if on := a.showing.Load(); on != nil {
		return on.forgets
	}
	return nil
}

// say puts one piece of work in the list of what is being done behind the
// window, for a build that keeps one.
func (a *API) say(at task.Task) { a.Window.Say(at) }

// finished takes one piece of work out of that list.
func (a *API) finished(id string) { a.Window.Finished(id) }

// unlevelled says whether a write reached the vault and the index did not
// follow.
// The answer goes on the wire, so the client that saved knows search has not
// caught up with what it saved, and it stands in the list of what is being done
// until a write levels the index again, so a person who is not looking at that
// note is told too.
func (a *API) unlevelled(err error) bool {
	if err == nil {
		a.finished(levellingTheIndex)
		return false
	}
	if !errors.Is(err, note.ErrUnlevelled) {
		return false
	}
	a.say(task.Task{ID: levellingTheIndex, Doing: "Bringing the index level", Failed: err.Error()})
	return true
}

// Shut refuses every question from now on, and there is no opening it again. It
// answers once the questions already taken have been answered, so everything an
// answer reaches into is still there for the whole of it.
func (a *API) Shut() {
	a.shut.Store(true)
	<-a.questions.seal()
}

func (a *API) closed() bool { return a.shut.Load() }

// shown is the vault a question is answered over. A window standing on nothing
// has none, and every question that would reach into a vault is refused there.
func (a *API) shown() (domain.Vault, error) {
	v := a.Showing()
	if v.ID == "" {
		return domain.Vault{}, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	return v, nil
}

// Answering is the agent the panel's tasks go to, and nothing where the vault
// has none.
func (a *API) Answering() port.Agent {
	if taking := a.agent.Load(); taking != nil {
		return *taking
	}
	return nil
}

// Answers is who takes the panel's tasks from now on. Nothing leaves the vault
// with no agent.
func (a *API) Answers(taking port.Agent) {
	if taking == nil {
		a.agent.Store(nil)
		return
	}
	a.agent.Store(&taking)
}

func text(v *atomic.Value) string {
	s, _ := v.Load().(string)
	return s
}

func (a *API) GetVaultState(
	ctx context.Context, _ *connect.Request[v1.GetVaultStateRequest],
) (*connect.Response[v1.GetVaultStateResponse], error) {
	showing := a.Showing()
	out := &v1.GetVaultStateResponse{
		Name:        string(showing.ID),
		DisplayName: showing.Name,
		Path:        showing.Path,
		Ready:       a.Ready.Load(),
		Failed:      a.Failed.Why(),
		Unwatched:   a.Unwatched.Why(),
		Unreachable: a.Unreachable.Why(),
		Embedding:   text(&a.Indexing.Model) != "",
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered as it stands. A window standing on nothing holds no
	// chunks and counts none.
	if a.Indexing.Progress != nil && showing.ID != "" {
		if held, embedded, err := a.Indexing.Progress.Progress(ctx, showing.ID, text(&a.Indexing.Recipe)); err == nil {
			out.Chunks, out.Embedded = held, embedded
		}
	}
	return connect.NewResponse(out), nil
}

func (a *API) GetOpeningNote(
	ctx context.Context, _ *connect.Request[v1.GetOpeningNoteRequest],
) (*connect.Response[v1.GetOpeningNoteResponse], error) {
	showing := a.Showing()
	if showing.ID == "" {
		// A window standing on nothing opens on no note.
		return connect.NewResponse(&v1.GetOpeningNoteResponse{}), nil
	}

	ref, found, err := a.Notes.Queries.Opening(ctx, showing.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.GetOpeningNoteResponse{}
	if found {
		out.Note = noteOf(ref)
	}
	return connect.NewResponse(out), nil
}

func (a *API) GetNeighbourhood(
	ctx context.Context, r *connect.Request[v1.GetNeighbourhoodRequest],
) (*connect.Response[v1.GetNeighbourhoodResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := note.ShowNeighbourhood{Links: a.Notes.Links, Notes: a.Notes.Queries}.
		Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Which of three each note on the picture is, asked once for the whole of
	// it, so a client draws a deck and a stencil as what they are.
	paths := []string{found.Focus.Path}
	for _, related := range found.Related {
		paths = append(paths, related.Path)
	}
	types, err := a.typesAt(ctx, showing, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.GetNeighbourhoodResponse{
		Focus:     noteOf(found.Focus),
		FocusType: typeOf(types[found.Focus.Path]),
	}
	for _, related := range found.Related {
		out.Related = append(out.Related, &v1.Neighbour{
			Note:    noteOf(related.NoteRef),
			Seat:    seatOf(related.Seat),
			Label:   related.Label,
			Through: related.Parent,
			Mutual:  related.Mutual,
			Type:    typeOf(types[related.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// ResolveAddresses answers where addresses written in one note land. An address
// that reaches nothing is left out of the answer.
func (a *API) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Notes.Links.Resolve(ctx, showing.ID, r.Msg.GetFrom(), r.Msg.GetWritten())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// In the order they were asked about, and an address asked about twice is
	// one answer.
	out := &v1.ResolveAddressesResponse{}
	said := make(map[string]bool, len(found))
	for _, written := range r.Msg.GetWritten() {
		one, reached := found[written]
		if !reached || said[written] {
			continue
		}
		said[written] = true
		vault, crossed := one.InVault(showing.ID)
		out.Resolved = append(out.Resolved, &v1.ResolvedAddress{
			Written:   written,
			Path:      one.To,
			Vault:     string(vault),
			Crossed:   crossed,
			Ambiguous: one.Ambiguous,
		})
	}
	return connect.NewResponse(out), nil
}

// WatchVaultChanges reports what moved, for as long as the caller listens.
func (a *API) WatchVaultChanges(
	ctx context.Context,
	_ *connect.Request[v1.WatchVaultChangesRequest],
	out *connect.ServerStream[v1.WatchVaultChangesResponse],
) error {
	line, done := a.Listeners.listen()
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
			renamed := make([]*v1.Move, 0, len(what.renamed))
			for _, went := range what.renamed {
				renamed = append(renamed, &v1.Move{From: went.From, To: went.To})
			}
			if err := out.Send(&v1.WatchVaultChangesResponse{
				Paths: what.paths, Reload: what.reload, Renamed: renamed,
			}); err != nil {
				return err
			}
		}
	}
}

func noteOf(n domain.NoteRef) *v1.Note {
	return &v1.Note{Path: n.Path, Title: n.Title, Identifier: n.ID}
}

func seatOf(s domain.Relation) v1.Seat {
	switch s {
	case domain.SeatParent:
		return v1.Seat_SEAT_PARENT
	case domain.SeatChild:
		return v1.Seat_SEAT_CHILD
	case domain.SeatJump:
		return v1.Seat_SEAT_JUMP
	case domain.SeatSibling:
		return v1.Seat_SEAT_SIBLING
	default:
		return v1.Seat_SEAT_UNSPECIFIED
	}
}
