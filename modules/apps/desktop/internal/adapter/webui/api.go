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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/agent"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// API is the vault a client is looking at and what can be asked about it.
type API struct {
	Vault domain.Vault
	Notes port.NoteQueries
	Links port.LinkQueries

	// Scan reads the whole vault. Keeping it level afterwards is a use case,
	// and what it produces arrives here through Listeners.
	Scan      func(context.Context, domain.Vault) (usecase.ScanResult, error)
	Listeners audience[changed]

	// Agent takes the tasks the panel sends. A vault without one answers that
	// it has none, and the rest of the window works as it did.
	Agent agent.Agent

	// Reads and Saves are how the window opens a note and puts it back. A build
	// without them answers that a note cannot be edited here.
	Reads *note.Read
	Saves *note.Write
	// Wrote is what a save raises: the reading behind the window asks the index
	// what owes a vector, once the vault has been still. Nil for a build with
	// nothing reading behind it, and then a save changes no vectors.
	Wrote func()

	// Makes is how the window makes a note, and Joins how it writes a
	// relationship into one. A build without them answers that a note cannot be
	// made here.
	Makes *note.Create
	Joins *note.Linking

	// Finds is how the window searches the text the vault holds, by the words
	// in it and by what it means. A build without one answers that it cannot be
	// searched, and the names a vault holds are answered all the same.
	Finds *search.Search

	// Drawing is everyone drawing this vault, for a change to a note being made
	// while they may be showing it.
	Drawing audience[domain.Editing]

	// Watching is everyone drawing this vault, for when something asks that a
	// note be put in front of the person.
	Watching audience[string]

	// Leaving is everyone drawing this vault, for the moment the window goes:
	// each is asked to write what only it holds, and answers when it has.
	Leaving leaving
	// Writing is the writes taken and not yet finished.
	Writing inflight

	// Indexed counts what the scan has stored so far. Ready is set when it
	// finished, Failed when it could not — a vault that could not be read is
	// not an empty one, and the interface has to be able to tell them apart.
	Indexed atomic.Int64
	Ready   atomic.Bool
	Failed  atomic.Value
	// Unwatched is why the vault is not being followed, when it is not.
	Unwatched atomic.Value
	// Unreachable is why an agent cannot be reached, when one cannot.
	Unreachable atomic.Value

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
	// Reading is the source being read now, empty between sources and after the
	// last one.
	Reading atomic.Value
	// Books counts what the vault holds and BooksRead how far cutting has got
	// through them. Cutting opens files, so a book is its unit; embedding works
	// on what cutting produced, and Learning says which of the two is running.
	Books     atomic.Int64
	BooksRead atomic.Int64
	Learning  atomic.Bool
	// Owed is what the pass now running found to do and how much of it it has
	// done. The two are one fact and are stored together: read one at a time,
	// they can be seen in a state neither of them was ever in.
	Owed atomic.Pointer[Owed]
	// Busy is set for as long as this vault is being read: its notes, then its
	// books, then their vectors. Reading a book and embedding one change no
	// file, so a client asks again for as long as it holds.
	Busy atomic.Bool
}

// failure is what stopped the scan, or empty while nothing has.
func (a *API) failure() string { return text(&a.Failed) }

func text(v *atomic.Value) string {
	s, _ := v.Load().(string)
	return s
}

func (a *API) State(ctx context.Context, _ *connect.Request[v1.StateRequest]) (*connect.Response[v1.StateResponse], error) {
	owed := a.Owed.Load()
	if owed == nil {
		owed = &Owed{}
	}
	out := &v1.StateResponse{
		Name:        a.Vault.Name,
		Path:        a.Vault.Path,
		Indexed:     a.Indexed.Load(),
		Ready:       a.Ready.Load(),
		Failed:      a.failure(),
		Unwatched:   text(&a.Unwatched),
		Unreachable: text(&a.Unreachable),
		Reading:     text(&a.Reading),
		Embedding:   text(&a.Model) != "",
		Books:       a.Books.Load(),
		BooksRead:   a.BooksRead.Load(),
		Learning:    a.Learning.Load(),
		Owing:       owed.Owing,
		Made:        owed.Made,
		Busy:        a.Busy.Load(),
	}
	// A count that cannot be taken leaves the pair at nothing, and the rest of
	// the state is answered as it stands.
	if a.Progress != nil {
		if held, embedded, err := a.Progress.Progress(ctx, a.Vault.ID, text(&a.Recipe)); err == nil {
			out.Chunks, out.Embedded = held, embedded
		}
	}
	return connect.NewResponse(out), nil
}

func (a *API) Opening(ctx context.Context, _ *connect.Request[v1.OpeningRequest]) (*connect.Response[v1.OpeningResponse], error) {
	ref, found, err := a.Notes.Opening(ctx, a.Vault.ID)
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
	found, err := note.ShowNeighbourhood{Links: a.Links, Notes: a.Notes}.
		Execute(ctx, a.Vault, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.NeighbourhoodResponse{Focus: noteOf(found.Focus)}
	for _, related := range found.Related {
		out.Related = append(out.Related, &v1.Seated{
			Note:    noteOf(related.NoteRef),
			Seat:    seatOf(related.Seat),
			Label:   related.Label,
			Through: related.Through,
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
			renamed := make([]*v1.Renamed, 0, len(what.renamed))
			for _, went := range what.renamed {
				renamed = append(renamed, &v1.Renamed{From: went.From, To: went.To})
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

// Owed is the work one pass has in hand: what it found to do, and how much of
// it it has done.
type Owed struct {
	Owing int64
	Made  int64
}
