package webui

import (
	"context"
	"errors"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// errNoCards is what a build with no path to the decks and the stencils
// answers.
var errNoCards = errors.New("this build cannot work the cards of a vault")

// Stencils is every stencil the vault holds, by what it is called and what it
// asks for. A vault holding more than one answer carries is answered with as
// many as it carries, and told how many it holds.
func (a *API) Stencils(
	ctx context.Context, r *connect.Request[v1.StencilsRequest],
) (*connect.Response[v1.StencilsResponse], error) {
	if a.Offered == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	held, err := a.Offered.Execute(ctx, showing)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.StencilsResponse{Held: int32(len(held))}
	held = held[:min(len(held), offering(r.Msg.GetLimit()))]
	out.Stencils = make([]*v1.Offered, 0, len(held))
	for _, stencil := range held {
		out.Stencils = append(out.Stencils, offeredOf(stencil))
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

// ReadStencil is the fields and the faces of one stencil.
func (a *API) ReadStencil(
	ctx context.Context, r *connect.Request[v1.ReadStencilRequest],
) (*connect.Response[v1.ReadStencilResponse], error) {
	if a.Cards == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Cards.Stencil(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.ReadStencilResponse{}
	if refusal, refused := refusedStencil(found.Outcome, found.Type); refused {
		out.Refusal = &refusal
		return connect.NewResponse(out), nil
	}
	out.Stencil = stencilOf(found.Path, a.titled(ctx, showing, found.Path), found.Stencil)
	// What the file was when this came out of it, for the client to present
	// when it writes the stencil back.
	out.At = fingerprintOf(found.Ref)
	return connect.NewResponse(out), nil
}

// ReadDeck is the cards of one deck, in the order they stand in the file.
func (a *API) ReadDeck(
	ctx context.Context, r *connect.Request[v1.ReadDeckRequest],
) (*connect.Response[v1.ReadDeckResponse], error) {
	if a.Cards == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Cards.Deck(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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
	out.Deck = deckOf(found.Path, a.titled(ctx, showing, found.Path), found.Deck, found.Stencils)
	out.At = fingerprintOf(found.Ref)
	return connect.NewResponse(out), nil
}

// WriteDeck puts cards into a deck. A deck still holding what the client read
// is written over; one holding something else is left alone and the client is
// told the deck changed.
func (a *API) WriteDeck(
	ctx context.Context, r *connect.Request[v1.WriteDeckRequest],
) (*connect.Response[v1.WriteDeckResponse], error) {
	if a.Cuts == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	body, err := container.DeckBody(r.Msg.GetPreamble(), cardsOf(r.Msg.GetCards()), r.Msg.GetTail())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	at, err := a.Cuts.Deck(ctx, showing, r.Msg.GetPath(), body, refOf(r.Msg.GetSeen()))
	if err == nil {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WriteDeckResponse{At: fingerprintOf(at)}), nil
	}
	if errors.Is(err, port.ErrChanged) {
		return connect.NewResponse(&v1.WriteDeckResponse{Changed: true}), nil
	}
	if errors.Is(err, note.ErrTooLarge) {
		refusal := v1.Refusal_REFUSAL_DECK_TOO_LARGE
		return connect.NewResponse(&v1.WriteDeckResponse{
			Refusal: &refusal, Bound: cards.MaxBytes,
		}), nil
	}
	refusal, refused := refusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteDeckResponse{Refusal: &refusal}), nil
}

// WriteStencil puts fields and faces into a stencil.
//
// The faces are the body and the fields are one key of the frontmatter, so the
// two are written one after the other, the second standing on the fingerprint
// the first produced.
func (a *API) WriteStencil(
	ctx context.Context, r *connect.Request[v1.WriteStencilRequest],
) (*connect.Response[v1.WriteStencilResponse], error) {
	if a.Cuts == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoCards)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	body, err := container.StencilBody(facesOf(r.Msg.GetFaces()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	at, err := a.Cuts.Stencil(ctx, showing, r.Msg.GetPath(), body, refOf(r.Msg.GetSeen()))
	if err == nil {
		at, err = a.declares(ctx, showing, r.Msg.GetPath(), r.Msg.GetFields(), at)
	}
	if err == nil {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WriteStencilResponse{At: fingerprintOf(at)}), nil
	}
	if errors.Is(err, port.ErrChanged) {
		return connect.NewResponse(&v1.WriteStencilResponse{Changed: true}), nil
	}
	refusal, refused := refusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteStencilResponse{Refusal: &refusal}), nil
}

// declares writes the fields a stencil asks a person for. A stencil already
// declaring them is not written, so a caller that changed nothing but the faces
// leaves one file behind and not two.
func (a *API) declares(
	ctx context.Context, showing domain.Vault, path string, fields []string, at domain.FileRef,
) (domain.FileRef, error) {
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return domain.FileRef{}, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.FileRef{}, err
	}
	f, err := format.OpenStencil(raw)
	if err != nil {
		return domain.FileRef{}, err
	}
	if slices.Equal(f.Fields(), fields) {
		return at, nil
	}
	if err := f.SetFields(fields); err != nil {
		return domain.FileRef{}, err
	}
	writer, err := a.Writers.Open(showing)
	if err != nil {
		return domain.FileRef{}, err
	}
	written, err := writer.Write(ctx, path, f.Bytes(), at)
	if err != nil {
		return domain.FileRef{}, err
	}
	if a.Cuts.Index == nil {
		return written, nil
	}
	return written, a.Cuts.Index(ctx, showing, []string{path})
}

// titled is what the vault calls the note at a path. A build with no index, and
// a path the index holds no note at, are answered with no name.
func (a *API) titled(ctx context.Context, showing domain.Vault, path string) string {
	if a.Notes == nil {
		return ""
	}
	found, err := a.Notes.Notes(ctx, showing.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}

// refusedDeck is why a deck was not read. What holds for a note holds here, and
// a deck read at a bound of its own is refused at that bound.
func refusedDeck(o note.Outcome, is domain.NoteType) (v1.Refusal, bool) {
	if o == note.TooLarge {
		return v1.Refusal_REFUSAL_DECK_TOO_LARGE, true
	}
	if o == note.Ok && is != domain.TypeDeck {
		return v1.Refusal_REFUSAL_NOT_A_DECK, true
	}
	return refusalOf(o)
}

// refusedStencil is why a stencil was not read.
func refusedStencil(o note.Outcome, is domain.NoteType) (v1.Refusal, bool) {
	if o == note.Ok && is != domain.TypeStencil {
		return v1.Refusal_REFUSAL_NOT_A_STENCIL, true
	}
	return refusalOf(o)
}
