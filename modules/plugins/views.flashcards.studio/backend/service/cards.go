package service

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/note"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/service/cardwire"
)

// cardsHandler answers about the decks and stencils a vault is arranged into.
type cardsHandler struct {
	*Backend
}

// ListStencils is every stencil the vault holds, by what it is called and what
// it asks for. A vault holding more than one answer carries is answered with as
// many as it carries, and told how many it holds.
func (h *cardsHandler) ListStencils(
	ctx context.Context, r *connect.Request[v1.ListStencilsRequest],
) (*connect.Response[v1.ListStencilsResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	held, count, err := h.scenarios().cards.List.Execute(ctx, showing, getStencilLimit(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.ListStencilsResponse{Total: int32(count)}
	out.Stencils = make([]*v1.StencilSummary, 0, len(held))
	for _, stencil := range held {
		out.Stencils = append(out.Stencils, cardwire.NewWireSummary(stencil))
	}
	return connect.NewResponse(out), nil
}

// getStencilLimit is how many stencils one answer carries. Nothing asked for takes the
// ceiling, and so does more than the ceiling.
func getStencilLimit(limit int32) int {
	if limit <= 0 || limit > cardwire.MaxStencils {
		return cardwire.MaxStencils
	}
	return int(limit)
}

// CreateStencil puts a stencil declaring these fields in the vault, showing no
// face.
func (h *cardsHandler) CreateStencil(
	ctx context.Context, r *connect.Request[v1.CreateStencilRequest],
) (*connect.Response[v1.CreateStencilResponse], error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		in.Fields = r.Msg.GetFields()
		return h.scenarios().cards.Create.Stencil(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateStencilResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}), nil
}

// CreateDeck puts a deck of no cards in the vault.
func (h *cardsHandler) CreateDeck(
	ctx context.Context, r *connect.Request[v1.CreateDeckRequest],
) (*connect.Response[v1.CreateDeckResponse], error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.scenarios().cards.Create.Deck(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateDeckResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}), nil
}

// createCardNote is what making a deck and making a stencil have in common: the vault
// being shown, the window's hold on writing, and the error codes a file that
// could not be made comes back as.
func (b *Backend) createCardNote(
	ctx context.Context, cut func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, *v1.ErrorCode, bool, error) {
	showing, err := b.getCurrentVault(ctx)
	if err != nil {
		return cards.CreateNoteResult{}, nil, false, err
	}
	made, err := cut(showing, cards.New{Title: title, Path: folder})
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		// The file is on disk under that name and nothing renumbers a second
		// attempt, so the path is the only way back to it.
		return made, nil, isUnlevelled(err), nil
	}
	if errors.Is(err, cards.ErrNoFields) {
		return cards.CreateNoteResult{}, nil, false, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, refused := getErrorCode(err)
	if !refused {
		return cards.CreateNoteResult{}, nil, false, connect.NewError(getCode(err), err)
	}
	return cards.CreateNoteResult{}, &reason, false, nil
}

// RenameStencilField gives one of a stencil's fields a different name, in the
// stencil and in every card of every deck that stencil cuts. A stencil still
// holding what the client read is written; one holding something else is left
// alone and the client is told the stencil changed.
func (h *cardsHandler) RenameStencilField(
	ctx context.Context, r *connect.Request[v1.RenameStencilFieldRequest],
) (*connect.Response[v1.RenameStencilFieldResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	renamed, err := h.scenarios().cards.RenameField.Execute(ctx, showing, cards.Rename{
		Stencil: r.Msg.GetPath(), From: r.Msg.GetFrom(), To: r.Msg.GetTo(),
		Fingerprint: readFingerprint(r.Msg.GetBaseline()),
	})
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		out := cardwire.NewRenameResponse(renamed, writeFingerprint(renamed.Stencil))
		out.IsUnlevelled = isUnlevelled(err)
		return connect.NewResponse(out), nil
	}
	// The name a rename is given is the client's: one the stencil does not
	// declare, and one it already declares, are both a name to correct.
	if errors.Is(err, format.ErrNoSuchField) || errors.Is(err, format.ErrFieldTaken) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, refused := getErrorCode(err)
	if !refused {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&v1.RenameStencilFieldResponse{Error: &reason}), nil
}

// ReadStencil is the fields and the faces of one stencil.
func (h *cardsHandler) ReadStencil(
	ctx context.Context, r *connect.Request[v1.ReadStencilRequest],
) (*connect.Response[v1.ReadStencilResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().cards.Read.Stencil(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &v1.ReadStencilResponse{}
	if code, refused := errorCodeOfStencil(found.Outcome, found.Type); refused {
		out.Error = &code
		return connect.NewResponse(out), nil
	}
	title := h.getTitle(ctx, showing, found.Path)
	out.Stencil = cardwire.NewWireStencil(found.Path, title, found.Body)
	// What the file was when this came out of it, for the client to present
	// when it writes the stencil back.
	out.At = writeFingerprint(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// ReadDeck is the cards of one deck, in the order they stand in the file.
func (h *cardsHandler) ReadDeck(
	ctx context.Context, r *connect.Request[v1.ReadDeckRequest],
) (*connect.Response[v1.ReadDeckResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().cards.Read.Deck(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &v1.ReadDeckResponse{}
	if code, refused := errorCodeOfDeck(found.Outcome, found.Type); refused {
		out.Error = &code
		// The bound travels with the error code, so the interface names it
		// without holding a number of its own.
		if code == v1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE {
			out.Bound = cards.MaxBytes
		}
		return connect.NewResponse(out), nil
	}
	title := h.getTitle(ctx, showing, found.Path)
	out.Deck = cardwire.NewWireDeck(found.Path, title, found.Body, found.Stencils)
	out.At = writeFingerprint(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// WriteDeck puts cards into a deck. A deck still holding what the client read
// is written over; one holding something else is left alone and the client is
// told the deck changed.
func (h *cardsHandler) WriteDeck(
	ctx context.Context, r *connect.Request[v1.WriteDeckRequest],
) (*connect.Response[v1.WriteDeckResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.DeckBody(cardwire.NewDeck(r.Msg))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	wrote, err := h.scenarios().cards.Write.Deck(ctx, showing, r.Msg.GetPath(), body, readFingerprint(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&v1.WriteDeckResponse{
			At: writeFingerprint(wrote.Fingerprint), IsUnlevelled: unlevelled,
		}), nil
	}
	if errors.Is(err, note.ErrTooLarge) {
		code := v1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE
		return connect.NewResponse(&v1.WriteDeckResponse{
			Error: &code, Bound: cards.MaxBytes,
		}), nil
	}
	reason, refused := getErrorCode(err)
	if !refused {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&v1.WriteDeckResponse{Error: &reason}), nil
}

// WriteStencil puts fields and faces into a stencil. A stencil still holding
// what the client read is written over; one holding something else is left
// alone and the client is told the stencil changed.
func (h *cardsHandler) WriteStencil(
	ctx context.Context, r *connect.Request[v1.WriteStencilRequest],
) (*connect.Response[v1.WriteStencilResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.StencilBody(
		r.Msg.GetPreamble(), cardwire.NewFaces(r.Msg.GetFaces()), r.Msg.GetTail())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	at, err := h.scenarios().cards.Write.Stencil(
		ctx, showing, r.Msg.GetPath(), body, r.Msg.GetFields(), readFingerprint(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&v1.WriteStencilResponse{
			At: writeFingerprint(at), IsUnlevelled: unlevelled,
		}), nil
	}
	reason, refused := getErrorCode(err)
	if !refused {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&v1.WriteStencilResponse{Error: &reason}), nil
}

// errorCodeOfDeck is why a deck was not read. What holds for a note holds here,
// and a deck read at a bound of its own is refused at that bound.
func errorCodeOfDeck(o note.ReadOutcome, is domain.NoteType) (v1.ErrorCode, bool) {
	if o == note.TooLarge {
		return v1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE, true
	}
	if o == note.Ok && is != format.TypeDeck {
		return v1.ErrorCode_ERROR_CODE_NOT_A_DECK, true
	}
	return convertOutcomeToErrorCode(o)
}

// errorCodeOfStencil is why a stencil was not read.
func errorCodeOfStencil(o note.ReadOutcome, is domain.NoteType) (v1.ErrorCode, bool) {
	if o == note.Ok && is != format.TypeStencil {
		return v1.ErrorCode_ERROR_CODE_NOT_A_STENCIL, true
	}
	return convertOutcomeToErrorCode(o)
}
