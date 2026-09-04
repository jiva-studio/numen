package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/refusal"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// errNoCards is what a build with no path to the decks and the stencils
// answers.
var errNoCards = errors.New("this build cannot work the cards of a vault")

// ListStencils is every stencil the vault holds, by what it is called and what
// it asks for. A vault holding more than one answer carries is answered with as
// many as it carries, and told how many it holds.
func (a *API) ListStencils(
	ctx context.Context, r *connect.Request[v1.ListStencilsRequest],
) (*connect.Response[v1.ListStencilsResponse], error) {
	if a.Cards.List == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	held, count, err := a.Cards.List.Execute(ctx, showing, offering(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.ListStencilsResponse{Held: int32(count)}
	out.Stencils = make([]*v1.StencilSummary, 0, len(held))
	for _, stencil := range held {
		out.Stencils = append(out.Stencils, summaryOf(stencil))
	}
	return connect.NewResponse(out), nil
}

// offering is how many stencils one answer carries. Nothing asked for takes the
// ceiling, and so does more than the ceiling.
func offering(limit int32) int {
	if limit <= 0 || limit > maxStencils {
		return maxStencils
	}
	return int(limit)
}

// CreateStencil puts a stencil declaring these fields in the vault, showing no
// face.
func (a *API) CreateStencil(
	ctx context.Context, r *connect.Request[v1.CreateStencilRequest],
) (*connect.Response[v1.CreateStencilResponse], error) {
	made, refusal, err := a.makes(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		in.Fields = r.Msg.GetFields()
		return a.Cards.Create.Stencil(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetFolder())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateStencilResponse{Path: made.Path, Refusal: refusal}), nil
}

// CreateDeck puts a deck of no cards in the vault.
func (a *API) CreateDeck(
	ctx context.Context, r *connect.Request[v1.CreateDeckRequest],
) (*connect.Response[v1.CreateDeckResponse], error) {
	made, refusal, err := a.makes(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return a.Cards.Create.Deck(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetFolder())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateDeckResponse{Path: made.Path, Refusal: refusal}), nil
}

// makes is what making a deck and making a stencil have in common: the vault
// being shown, the window's hold on writing, and the refusals a file that could
// not be made comes back as.
func (a *API) makes(
	ctx context.Context, cut func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, *v1.Refusal, error) {
	if a.Cards.Create == nil {
		return cards.CreateNoteResult{}, nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return cards.CreateNoteResult{}, nil, err
	}
	if !a.Writing.begin() {
		return cards.CreateNoteResult{}, nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	made, err := cut(showing, cards.New{Title: title, Folder: folder})
	if err == nil {
		if a.Wrote != nil {
			a.Wrote()
		}
		return made, nil, nil
	}
	if errors.Is(err, cards.ErrNoFields) {
		return cards.CreateNoteResult{}, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, refused := refusal.By(err)
	if !refused {
		return cards.CreateNoteResult{}, nil, connect.NewError(refusal.Coded(err), err)
	}
	return cards.CreateNoteResult{}, &reason, nil
}

// RenameStencilField gives one of a stencil's fields a different name, in the
// stencil and in every card of every deck that stencil cuts. A stencil still
// holding what the client read is written; one holding something else is left
// alone and the client is told the stencil changed.
func (a *API) RenameStencilField(
	ctx context.Context, r *connect.Request[v1.RenameStencilFieldRequest],
) (*connect.Response[v1.RenameStencilFieldResponse], error) {
	if a.Cards.RenameField == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	renamed, err := a.Cards.RenameField.Execute(ctx, showing, cards.Rename{
		Stencil: r.Msg.GetPath(), From: r.Msg.GetFrom(), To: r.Msg.GetTo(),
		Fingerprint: refOf(r.Msg.GetSeen()),
	})
	if err == nil {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(renamedOf(renamed)), nil
	}
	// The name a rename is given is the client's: one the stencil does not
	// declare, and one it already declares, are both a name to correct.
	if errors.Is(err, format.ErrNoSuchField) || errors.Is(err, format.ErrFieldTaken) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, refused := refusal.By(err)
	if !refused {
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.RenameStencilFieldResponse{Refusal: &reason}), nil
}

// ReadStencil is the fields and the faces of one stencil.
func (a *API) ReadStencil(
	ctx context.Context, r *connect.Request[v1.ReadStencilRequest],
) (*connect.Response[v1.ReadStencilResponse], error) {
	if a.Cards.Read == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Cards.Read.Stencil(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}

	out := &v1.ReadStencilResponse{}
	if refusal, refused := refusedStencil(found.Outcome, found.Type); refused {
		out.Refusal = &refusal
		return connect.NewResponse(out), nil
	}
	out.Stencil = stencilOf(found.Path, a.titled(ctx, showing, found.Path), found.Body)
	// What the file was when this came out of it, for the client to present
	// when it writes the stencil back.
	out.At = fingerprintOf(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// ReadDeck is the cards of one deck, in the order they stand in the file.
func (a *API) ReadDeck(
	ctx context.Context, r *connect.Request[v1.ReadDeckRequest],
) (*connect.Response[v1.ReadDeckResponse], error) {
	if a.Cards.Read == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Cards.Read.Deck(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}

	out := &v1.ReadDeckResponse{}
	if refusal, refused := refusedDeck(found.Outcome, found.Type); refused {
		out.Refusal = &refusal
		// The bound travels with the refusal, so the interface names it
		// without holding a number of its own.
		if refusal == v1.Refusal_REFUSAL_DECK_TOO_LARGE {
			out.Bound = cards.MaxBytes
		}
		return connect.NewResponse(out), nil
	}
	out.Deck = deckOf(found.Path, a.titled(ctx, showing, found.Path), found.Body, found.Stencils)
	out.At = fingerprintOf(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// WriteDeck puts cards into a deck. A deck still holding what the client read
// is written over; one holding something else is left alone and the client is
// told the deck changed.
func (a *API) WriteDeck(
	ctx context.Context, r *connect.Request[v1.WriteDeckRequest],
) (*connect.Response[v1.WriteDeckResponse], error) {
	if a.Cards.Write == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	body, err := format.DeckBody(writtenDeck(r.Msg))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	wrote, err := a.Cards.Write.Deck(ctx, showing, r.Msg.GetPath(), body, refOf(r.Msg.GetSeen()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save.
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WriteDeckResponse{At: fingerprintOf(wrote.Fingerprint)}), nil
	}
	if errors.Is(err, note.ErrTooLarge) {
		refusal := v1.Refusal_REFUSAL_DECK_TOO_LARGE
		return connect.NewResponse(&v1.WriteDeckResponse{
			Refusal: &refusal, Bound: cards.MaxBytes,
		}), nil
	}
	reason, refused := refusal.By(err)
	if !refused {
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.WriteDeckResponse{Refusal: &reason}), nil
}

// WriteStencil puts fields and faces into a stencil. A stencil still holding
// what the client read is written over; one holding something else is left
// alone and the client is told the stencil changed.
func (a *API) WriteStencil(
	ctx context.Context, r *connect.Request[v1.WriteStencilRequest],
) (*connect.Response[v1.WriteStencilResponse], error) {
	if a.Cards.Write == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	body, err := container.StencilBody(
		r.Msg.GetPreamble(), facesOf(r.Msg.GetFaces()), r.Msg.GetTail())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	at, err := a.Cards.Write.Stencil(
		ctx, showing, r.Msg.GetPath(), body, r.Msg.GetFields(), refOf(r.Msg.GetSeen()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save.
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WriteStencilResponse{At: fingerprintOf(at)}), nil
	}
	reason, refused := refusal.By(err)
	if !refused {
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.WriteStencilResponse{Refusal: &reason}), nil
}

// titled is what the vault calls the note at a path. A build with no index, and
// a path the index holds no note at, are answered with no name.
func (a *API) titled(ctx context.Context, showing domain.Vault, path string) string {
	if a.Notes.Queries == nil {
		return ""
	}
	found, err := a.Notes.Queries.Notes(ctx, string(showing.ID), []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}

// refusedDeck is why a deck was not read. What holds for a note holds here, and
// a deck read at a bound of its own is refused at that bound.
func refusedDeck(o note.ReadOutcome, is domain.NoteType) (v1.Refusal, bool) {
	if o == note.TooLarge {
		return v1.Refusal_REFUSAL_DECK_TOO_LARGE, true
	}
	if o == note.Ok && is != domain.TypeDeck {
		return v1.Refusal_REFUSAL_NOT_A_DECK, true
	}
	return refusal.Of(o)
}

// refusedStencil is why a stencil was not read.
func refusedStencil(o note.ReadOutcome, is domain.NoteType) (v1.Refusal, bool) {
	if o == note.Ok && is != domain.TypeStencil {
		return v1.Refusal_REFUSAL_NOT_A_STENCIL, true
	}
	return refusal.Of(o)
}
