package service

import (
	"context"
	"errors"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/cardwire"
)

// cardsHandler answers about the decks and card models a vault is arranged into.
type cardsHandler struct {
	*Backend
}

// ListCardModels is every card model the vault holds, by what it is called and what
// it asks for. A vault holding more than one answer carries is answered with as
// many as it carries, and told how many it holds.
func (h *cardsHandler) ListCardModels(
	ctx context.Context, r *connect.Request[flashcardsv1.ListCardModelsRequest],
) (*connect.Response[flashcardsv1.ListCardModelsResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	target, count, err := h.scenarios().cards.List.Execute(ctx, showing, getCardModelLimit(r.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &flashcardsv1.ListCardModelsResponse{Total: int32(count)}
	out.CardModels = make([]*flashcardsv1.CardModelSummary, 0, len(target))
	for _, cardModel := range target {
		out.CardModels = append(out.CardModels, cardwire.NewWireSummary(cardModel))
	}
	return connect.NewResponse(out), nil
}

// getCardModelLimit is how many card models one answer carries. Nothing asked for takes the
// ceiling, and so does more than the ceiling.
func getCardModelLimit(limit int32) int {
	if limit <= 0 || limit > cardwire.MaxCardModels {
		return cardwire.MaxCardModels
	}
	return int(limit)
}

// CreateCardModel puts a card model declaring these fields in the vault, showing no
// template.
func (h *cardsHandler) CreateCardModel(
	ctx context.Context, r *connect.Request[flashcardsv1.CreateCardModelRequest],
) (*connect.Response[flashcardsv1.CreateCardModelResponse], error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		in.Fields = r.Msg.GetFields()
		return h.scenarios().cards.Create.CardModel(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&flashcardsv1.CreateCardModelResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}), nil
}

// CreateDeck puts a deck of no cards in the vault.
func (h *cardsHandler) CreateDeck(
	ctx context.Context, r *connect.Request[flashcardsv1.CreateDeckRequest],
) (*connect.Response[flashcardsv1.CreateDeckResponse], error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.scenarios().cards.Create.Deck(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&flashcardsv1.CreateDeckResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}), nil
}

// createCardNote is what making a deck and making a card model have in common: the vault
// being shown, the window's hold on writing, and the error codes a file that
// could not be made comes back as.
func (b *Backend) createCardNote(
	ctx context.Context, create func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, *flashcardsv1.ErrorCode, bool, error) {
	showing, err := b.getCurrentVault(ctx)
	if err != nil {
		return cards.CreateNoteResult{}, nil, false, err
	}
	made, err := create(showing, cards.New{Title: title, Path: folder})
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		// The file is on disk under that name and nothing renumbers a second
		// attempt, so the path is the only way back to it.
		return made, nil, isUnlevelled(err), nil
	}
	if errors.Is(err, cards.ErrNoFields) {
		return cards.CreateNoteResult{}, nil, false, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return cards.CreateNoteResult{}, nil, false, connect.NewError(getCode(err), err)
	}
	return cards.CreateNoteResult{}, &reason, false, nil
}

// RenameCardModelField gives one of a card model's fields a different name, in the
// card model and in every card of every deck that card model cuts. A card model still
// holding what the client read is written; one holding something else is left
// alone and the client is told the card model changed.
func (h *cardsHandler) RenameCardModelField(
	ctx context.Context, r *connect.Request[flashcardsv1.RenameCardModelFieldRequest],
) (*connect.Response[flashcardsv1.RenameCardModelFieldResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	renamed, err := h.scenarios().cards.RenameField.Execute(ctx, showing, cards.Rename{
		CardModel: r.Msg.GetPath(), From: r.Msg.GetFrom(), To: r.Msg.GetTo(),
		FileStat: readFileStat(r.Msg.GetBaseline()),
	})
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		out := cardwire.NewRenameResponse(renamed, writeFileStat(renamed.CardModel))
		out.IsUnlevelled = isUnlevelled(err)
		return connect.NewResponse(out), nil
	}
	// The name a rename is given is the client's: one the card model does not
	// declare, and one it already declares, are both a name to correct.
	if errors.Is(err, format.ErrNoSuchField) || errors.Is(err, format.ErrFieldTaken) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&flashcardsv1.RenameCardModelFieldResponse{Error: &reason}), nil
}

// ReadCardModel is the fields and the templates of one card model.
func (h *cardsHandler) ReadCardModel(
	ctx context.Context, r *connect.Request[flashcardsv1.ReadCardModelRequest],
) (*connect.Response[flashcardsv1.ReadCardModelResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().cards.Read.CardModel(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &flashcardsv1.ReadCardModelResponse{}
	if code, ok := errorCodeOfCardModel(found.Outcome, found.Type); ok {
		out.Error = &code
		return connect.NewResponse(out), nil
	}
	title := h.getTitle(ctx, showing, found.Path)
	out.CardModel = cardwire.NewWireCardModel(found.Path, title, found.Body)
	// What the file was when this came out of it, for the client to present
	// when it writes the card model back.
	out.At = writeFileStat(found.FileStat)
	return connect.NewResponse(out), nil
}

// ReadDeck is the cards of one deck, in the order they stand in the file.
func (h *cardsHandler) ReadDeck(
	ctx context.Context, r *connect.Request[flashcardsv1.ReadDeckRequest],
) (*connect.Response[flashcardsv1.ReadDeckResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().cards.Read.Deck(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &flashcardsv1.ReadDeckResponse{}
	if code, ok := errorCodeOfDeck(found.Outcome, found.Type); ok {
		out.Error = &code
		// The bound travels with the error code, so the interface names it
		// without holding a number of its own.
		if code == flashcardsv1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE {
			out.Bound = cards.MaxBytes
		}
		return connect.NewResponse(out), nil
	}
	title := h.getTitle(ctx, showing, found.Path)
	out.Deck = cardwire.NewWireDeck(found.Path, title, found.Body, found.CardModels)
	out.At = writeFileStat(found.FileStat)
	return connect.NewResponse(out), nil
}

// WriteDeck puts cards into a deck. A deck still holding what the client read
// is written over; one holding something else is left alone and the client is
// told the deck changed.
func (h *cardsHandler) WriteDeck(
	ctx context.Context, r *connect.Request[flashcardsv1.WriteDeckRequest],
) (*connect.Response[flashcardsv1.WriteDeckResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.DeckBody(cardwire.NewDeck(r.Msg))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	wrote, err := h.scenarios().cards.Write.Deck(ctx, showing, r.Msg.GetPath(), body, readFileStat(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the file stat it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&flashcardsv1.WriteDeckResponse{
			At: writeFileStat(wrote.FileStat), IsUnlevelled: unlevelled,
		}), nil
	}
	if errors.Is(err, note.ErrTooLarge) {
		code := flashcardsv1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE
		return connect.NewResponse(&flashcardsv1.WriteDeckResponse{
			Error: &code, Bound: cards.MaxBytes,
		}), nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&flashcardsv1.WriteDeckResponse{Error: &reason}), nil
}

// WriteCardModel puts fields and templates into a card model. A card model still holding
// what the client read is written over; one holding something else is left
// alone and the client is told the card model changed.
func (h *cardsHandler) WriteCardModel(
	ctx context.Context, r *connect.Request[flashcardsv1.WriteCardModelRequest],
) (*connect.Response[flashcardsv1.WriteCardModelResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.CardModelBody(
		r.Msg.GetPreamble(), cardwire.NewTemplates(r.Msg.GetTemplates()), r.Msg.GetTail())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	pos, err := h.scenarios().cards.Write.CardModel(
		ctx, showing, r.Msg.GetPath(), body, r.Msg.GetFields(), readFileStat(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the file stat it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&flashcardsv1.WriteCardModelResponse{
			At: writeFileStat(pos), IsUnlevelled: unlevelled,
		}), nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&flashcardsv1.WriteCardModelResponse{Error: &reason}), nil
}

// errorCodeOfDeck is why a deck was not read. What holds for a note holds here,
// and a deck read at a bound of its own fails at that bound.
func errorCodeOfDeck(o note.ReadOutcome, is domain.NoteType) (flashcardsv1.ErrorCode, bool) {
	if o == note.TooLarge {
		return flashcardsv1.ErrorCode_ERROR_CODE_DECK_TOO_LARGE, true
	}
	if o == note.Ok && is != format.TypeDeck {
		return flashcardsv1.ErrorCode_ERROR_CODE_NOT_A_DECK, true
	}
	return convertOutcomeToErrorCode(o)
}

// errorCodeOfCardModel is why a card model was not read.
func errorCodeOfCardModel(o note.ReadOutcome, is domain.NoteType) (flashcardsv1.ErrorCode, bool) {
	if o == note.Ok && is != format.TypeCardModel {
		return flashcardsv1.ErrorCode_ERROR_CODE_NOT_A_CARD_MODEL, true
	}
	return convertOutcomeToErrorCode(o)
}
