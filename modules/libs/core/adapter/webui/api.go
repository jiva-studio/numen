// Package webui answers what a client may ask about a vault.
//
// The questions and their answers are the schema in modules/libs/protocol; this
// is the half that answers them. The interface asks over HTTP either way, so
// the same handler serves the window the application opens and a browser during
// development.
package webui

import (
	"context"
	"sync/atomic"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// API is the vault a client is looking at and what can be asked about it.
type API struct {
	// vault is the vault the window has open. It is replaced while requests are
	// being served, so every reader takes it through Showing.
	vault atomic.Pointer[domain.Vault]

	Notes port.NoteQueries
	Links port.LinkQueries

	// Scan reads the whole vault. Keeping it level afterwards is a use case,
	// and what it produces arrives here through Listeners.
	Scan      func(context.Context, domain.Vault) (usecase.ScanResult, error)
	Listeners audience[changed]

	// Opens puts another vault in the window: the agents are stopped, the vault
	// is swapped, and the agents are started again against the one that
	// arrived. A build without one answers that the window cannot be moved to
	// another vault.
	Opens func(context.Context, domain.Vault) error

	// Vaults is the list of vaults this installation holds, the one the window
	// is showing among them. A build without one answers that it holds no list.
	Vaults port.VaultRegistry

	// Choosing puts this machine's own folder picker in front of the person.
	// Only an application with a window has one, and a build without it answers
	// that a folder cannot be picked here.
	Choosing port.Folders

	// Adding turns a folder into a vault, Renaming is what a person calls one,
	// and Forgetting and Erasing take one off the list. A build without them
	// answers that the list cannot be changed here.
	Adding     *usecase.Add
	Renaming   *usecase.Rename
	Forgetting *usecase.Forget
	Erasing    *usecase.Erase

	// taking is the agent the panel's tasks go to. A vault without one answers
	// that it has none, and the rest of the window works as it did. It is
	// replaced while requests are being served, so it is taken through
	// Answering.
	taking atomic.Pointer[port.Agent]

	// Reads and Saves are how the window opens a note and puts it back. A build
	// without them answers that a note cannot be edited here.
	Reads *note.Read
	Saves *note.Write

	// Readers open the vault a document is drawn from and a folder is listed
	// out of. A path from outside arrives at the vault through them, and one
	// leaving the vault is refused there.
	Readers port.VaultReaders
	// Writers open the vault a folder is made in. A build without them answers
	// that a folder cannot be made here.
	Writers port.VaultWriters
	// Viewer holds the documents the window has open and the pages it has
	// drawn. A build without one answers that it cannot draw a document.
	Viewer *viewer
	// Marking says where a run of a source's text sits on the pages it was read
	// from. A build without one answers that it cannot say where a passage is.
	Marking *source.Marks
	// Playing is the socket a recording is played from. A build without one
	// answers with no address, and the window says the recording cannot be
	// played here.
	Playing *Loopback
	// Wrote is what a save raises: the reading behind the window asks the index
	// what owes a vector, once the vault has been still. Nil for a build with
	// nothing reading behind it, and then a save changes no vectors.
	Wrote func()
	// Recognises reads a scanned document and Transcribes hears a recording,
	// each for whoever asks. They are the jobs an agent asks through too, so
	// what a person started in the window is shown to both. A build without one
	// answers that it cannot do that run.
	Recognises  Run
	Transcribes Run
	// Cut asks for a source to be cut again from whatever its text now says. A
	// window that put a transcript right calls it, so search answers with the
	// words as they now read. Nil for a build with nothing cutting behind it,
	// and then a correction is seen in the tab alone.
	Cut func(context.Context, domain.Vault, string) error
	// Drops takes a recording's transcript away, with everything listening to
	// it produced. A build without one answers that a transcript cannot be
	// dropped here.
	Drops *source.DropTranscript

	// Makes is how the window makes a note, and Joins how it writes a
	// relationship into one. A build without them answers that a note cannot be
	// made here.
	Makes *note.Create
	Joins *note.Linking

	// Cards reads a deck or a stencil, Offered lists the stencils the vault
	// holds, and Cuts puts either back. MakesCards makes a deck, a stencil or a
	// preset, and RenamesField gives one of a stencil's fields a different name
	// everywhere it is written. A build without them answers that cards cannot
	// be worked here.
	Cards        *cards.Read
	Offered      *cards.List
	Cuts         *cards.Write
	MakesCards   *cards.Create
	RenamesField *cards.RenameField

	// Presets is the preset a deck is scheduled by, and how one is read,
	// written and made. Curves is what the one control of a preset comes to
	// over the whole range of its goal. A build without them answers that
	// presets cannot be worked here.
	Presets *flashcards.Presets
	Curves  *flashcards.Curves

	// Renames gives a note a different name, Moves puts a file or a folder
	// somewhere else in the vault, and Removes takes one out of it. A build
	// without them answers that nothing can be renamed, moved or removed here.
	Renames *note.Rename
	Moves   *usecase.Move
	Removes *note.Remove

	// Bringing copies files a person handed the window into a folder of the
	// vault. A build without it takes none.
	Bringing *usecase.Bring

	// Sync reads whether a note's title and its filename are kept as one name,
	// and Chooses writes it. A build with no Chooses answers that it configures
	// nothing; one with no Sync reads what an installation nobody has
	// configured does.
	Sync    note.Syncing
	Chooses func(kept note.Sync) error

	// Hangs reads whether a node hangs the headings of its note under it, Parts
	// how many of them stand there at once, and ChoosesHanging and ChoosesParts
	// write the two. A build with no writer answers that it configures nothing;
	// one with no reader reads what an installation nobody has configured does.
	Hangs          func() bool
	ChoosesHanging func(hangs bool) error
	Parts          func() int
	ChoosesParts   func(parts int) error

	// Reviews reads the hour a day of review begins at, and ChoosesReviewing
	// writes it. A build with no writer answers that it configures nothing; one
	// with no reader reads what an installation nobody has configured does.
	Reviews          func() string
	ChoosesReviewing func(starts string) error

	// Finds is how the window searches the text the vault holds, by the words
	// in it and by what it means. A build without one answers that it cannot be
	// searched, and the names a vault holds are answered all the same.
	Finds *search.Search

	// Drawing is everyone drawing this vault, for a change to a note being made
	// while they may be showing it.
	Drawing audience[domain.Editing]

	// Watching is everyone drawing this vault, for when something asks that a
	// place be put in front of the person.
	Watching audience[domain.Place]

	// attending is what the person has open, as the window last said. It is
	// replaced while requests are being served, so every reader takes it
	// through Attended.
	attending atomic.Pointer[domain.Attention]

	// Attends hears what the person has open each time the window says it,
	// once what it said stands. A build without one takes the report and tells
	// nobody.
	Attends func(domain.Attention)

	// Leaving is everyone drawing this vault, for the moment the window goes:
	// each is asked to write what only it holds, and answers when it has.
	Leaving leaving
	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Ready is set when the scan finished, Failed when it could not — a vault
	// that could not be read is not an empty one, and the interface has to be
	// able to tell them apart.
	Ready  atomic.Bool
	Failed atomic.Value
	// Unwatched is why the vault is not being followed, when it is not.
	Unwatched atomic.Value
	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable atomic.Value

	// Tasking is everything being done behind the window. Whatever does work
	// puts itself there and the window reads the list, so a new kind of work is
	// an entry rather than another field here.
	Tasking *task.Tasks

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

	// Themes are the stylesheets the window may be dressed in. They belong to
	// the installation, so they arrive here from whatever put the window
	// together. Nil answers that this build has none.
	Themes numenv1connect.ThemeServiceHandler
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
	if taking := a.taking.Load(); taking != nil {
		return *taking
	}
	return nil
}

// Answers is who takes the panel's tasks from now on. Nothing leaves the vault
// with no agent.
func (a *API) Answers(taking port.Agent) {
	if taking == nil {
		a.taking.Store(nil)
		return
	}
	a.taking.Store(&taking)
}

// failure is what stopped the scan, or empty while nothing has.
func (a *API) failure() string { return text(&a.Failed) }

func text(v *atomic.Value) string {
	s, _ := v.Load().(string)
	return s
}

func (a *API) State(ctx context.Context, _ *connect.Request[v1.StateRequest]) (*connect.Response[v1.StateResponse], error) {
	showing := a.Showing()
	out := &v1.StateResponse{
		Name:        showing.Name,
		Path:        showing.Path,
		Ready:       a.Ready.Load(),
		Failed:      a.failure(),
		Unwatched:   text(&a.Unwatched),
		Unreachable: text(&a.Unreachable),
		Embedding:   text(&a.Model) != "",
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered as it stands. A window standing on nothing holds no
	// chunks and counts none.
	if a.Progress != nil && showing.ID != "" {
		if held, embedded, err := a.Progress.Progress(ctx, showing.ID, text(&a.Recipe)); err == nil {
			out.Chunks, out.Embedded = held, embedded
		}
	}
	return connect.NewResponse(out), nil
}

func (a *API) Opening(ctx context.Context, _ *connect.Request[v1.OpeningRequest]) (*connect.Response[v1.OpeningResponse], error) {
	showing := a.Showing()
	if showing.ID == "" {
		// A window standing on nothing opens on no note.
		return connect.NewResponse(&v1.OpeningResponse{}), nil
	}

	ref, found, err := a.Notes.Opening(ctx, showing.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.OpeningResponse{}
	if found {
		out.Note = noteOf(ref)
	}
	return connect.NewResponse(out), nil
}

func (a *API) Neighbourhood(ctx context.Context, r *connect.Request[v1.NeighbourhoodRequest]) (*connect.Response[v1.NeighbourhoodResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := note.ShowNeighbourhood{Links: a.Links, Notes: a.Notes}.
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

	out := &v1.NeighbourhoodResponse{
		Focus:     noteOf(found.Focus),
		FocusType: typeOf(types[found.Focus.Path]),
	}
	for _, related := range found.Related {
		out.Related = append(out.Related, &v1.Seated{
			Note:    noteOf(related.NoteRef),
			Seat:    seatOf(related.Seat),
			Label:   related.Label,
			Through: related.Through,
			Mutual:  related.Mutual,
			Type:    typeOf(types[related.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// Changes reports what moved, for as long as the caller listens.
func (a *API) Changes(
	ctx context.Context,
	_ *connect.Request[v1.ChangesRequest],
	out *connect.ServerStream[v1.ChangesResponse],
) error {
	line, done := a.Listeners.listen()
	defer done()

	// Named as following before anything has changed. A stream that says
	// nothing until the vault moves is indistinguishable from one that never
	// opened, and a caller waiting on its first message waits for an edit.
	if err := out.Send(&v1.ChangesResponse{}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case what, open := <-line:
			if !open {
				return nil
			}
			renamed := make([]*v1.Went, 0, len(what.renamed))
			for _, went := range what.renamed {
				renamed = append(renamed, &v1.Went{From: went.From, To: went.To})
			}
			if err := out.Send(&v1.ChangesResponse{
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

func seatOf(s domain.Seat) v1.Seat {
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
