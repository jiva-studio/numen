package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/cardwire"
)

// cardsHandler answers about the decks and card models a vault is arranged into.
type cardsHandler struct {
	*Backend
}

// ListCardModels returns the card models of the vault, with each title and field
// list. A vault with more card models than the limit returns that many, and
// reports the total.
func (h *cardsHandler) ListCardModels(
	ctx context.Context, req *api.ListCardModelsRequest,
) (*api.ListCardModelsResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	target, count, err := h.scenarios().cards.List.Execute(ctx, showing, getCardModelLimit(req.Limit))
	if err != nil {
		return nil, newError(server.CodeInternal, err)
	}

	summaries := make([]*api.CardModelSummary, 0, len(target))
	for _, cardModel := range target {
		summaries = append(summaries, cardwire.NewWireSummary(cardModel))
	}
	return &api.ListCardModelsResponse{CardModels: summaries, Total: int32(count)}, nil
}

// getCardModelLimit returns how many card models one response contains. A limit
// of zero or less, or above the ceiling, yields the ceiling.
func getCardModelLimit(limit int32) int {
	if limit <= 0 || limit > cardwire.MaxCardModels {
		return cardwire.MaxCardModels
	}
	return int(limit)
}

// CreateCardModel puts a card model declaring these fields in the vault, showing no
// template.
func (h *cardsHandler) CreateCardModel(
	ctx context.Context, req *api.CreateCardModelRequest,
) (*api.CreateCardModelResponse, error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		in.Fields = req.Fields
		return h.scenarios().cards.Create.CardModel(ctx, showing, in)
	}, req.Title, req.Path)
	if err != nil {
		return nil, err
	}
	return &api.CreateCardModelResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}, nil
}

// CreateDeck puts a deck of no cards in the vault.
func (h *cardsHandler) CreateDeck(
	ctx context.Context, req *api.CreateDeckRequest,
) (*api.CreateDeckResponse, error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.scenarios().cards.Create.Deck(ctx, showing, in)
	}, req.Title, req.Path)
	if err != nil {
		return nil, err
	}
	return &api.CreateDeckResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}, nil
}

// createCardNote is what making a deck and making a card model have in common: the vault
// being shown, the window's hold on writing, and the error codes a file that
// could not be made comes back as.
func (b *Backend) createCardNote(
	ctx context.Context, create func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, api.ErrorCode, bool, error) {
	showing, err := b.getCurrentVault(ctx)
	if err != nil {
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, err
	}
	made, err := create(showing, cards.New{Title: title, Body: "", Path: folder, Fields: nil, Templates: nil})
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		// The file is on disk under that name and nothing renumbers a second
		// attempt, so the path is the only way back to it.
		return made, api.ErrorCodeUnspecified, isUnlevelled(err), nil
	}
	if errors.Is(err, cards.ErrNoFields) {
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, newError(server.CodeInvalidArgument, err)
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, newError(getCode(err), err)
	}
	return cards.CreateNoteResult{Path: "", ID: "", Title: ""}, reason, false, nil
}

// RenameCardModelField renames one field of a card model, in the card model and
// in every card of every deck that the card model generates. A card model that
// still matches what the client read is written; a card model that changed is
// left unchanged and the response reports the change.
func (h *cardsHandler) RenameCardModelField(
	ctx context.Context, req *api.RenameCardModelFieldRequest,
) (*api.RenameCardModelFieldResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	renamed, err := h.scenarios().cards.RenameField.Execute(ctx, showing, cards.Rename{
		CardModel: req.Path, From: req.From, To: req.To,
		FileStat: readFileStat(req.Baseline),
	})
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		out := cardwire.NewRenameResponse(renamed, writeFileStat(renamed.CardModel))
		out.IsUnlevelled = isUnlevelled(err)
		return out, nil
	}
	// The name a rename is given is the client's: one the card model does not
	// declare, and one it already declares, are both a name to correct.
	if errors.Is(err, format.ErrNoSuchField) || errors.Is(err, format.ErrFieldTaken) {
		return nil, newError(server.CodeInvalidArgument, err)
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, newError(getCode(err), err)
	}
	return &api.RenameCardModelFieldResponse{
		Decks: nil, Cards: 0, NotWritten: nil, Error: reason, At: nil, IsUnlevelled: false,
	}, nil
}

// ReadCardModel is the fields and the templates of one card model.
func (h *cardsHandler) ReadCardModel(
	ctx context.Context, req *api.ReadCardModelRequest,
) (*api.ReadCardModelResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().cards.Read.CardModel(ctx, showing, req.Path)
	if err != nil {
		return nil, newError(getCode(err), err)
	}

	out := &api.ReadCardModelResponse{CardModel: nil, Error: api.ErrorCodeUnspecified, At: nil}
	if code, ok := errorCodeOfCardModel(found.Outcome, found.Type); ok {
		out.Error = code
		return out, nil
	}
	title := h.getTitle(ctx, showing, found.Path)
	out.CardModel = cardwire.NewWireCardModel(found.Path, title, found.Body)
	// What the file was when this came out of it, for the client to present
	// when it writes the card model back.
	out.At = writeFileStat(found.FileStat)
	return out, nil
}

// ReadDeck is the cards of one deck, in the order they stand in the file.
func (h *cardsHandler) ReadDeck(
	ctx context.Context, req *api.ReadDeckRequest,
) (*api.ReadDeckResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().cards.Read.Deck(ctx, showing, req.Path)
	if err != nil {
		return nil, newError(getCode(err), err)
	}

	out := &api.ReadDeckResponse{Deck: nil, Error: api.ErrorCodeUnspecified, At: nil, Bound: 0}
	if code, ok := errorCodeOfDeck(found.Outcome, found.Type); ok {
		out.Error = code
		// 		// The bound is sent with the error code, so the interface names it
		// 		// without storing a number of its own.
		if code == api.ErrorCodeDeckTooLarge {
			out.Bound = cards.MaxBytes
		}
		return out, nil
	}
	title := h.getTitle(ctx, showing, found.Path)
	out.Deck = cardwire.NewWireDeck(found.Path, title, found.Body, found.CardModels)
	out.At = writeFileStat(found.FileStat)
	return out, nil
}

// WriteDeck writes cards into a deck. A deck that still matches what the client
// read is overwritten; a deck that changed is left unchanged and the response
// reports the change.
func (h *cardsHandler) WriteDeck(
	ctx context.Context, req *api.WriteDeckRequest,
) (*api.WriteDeckResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.DeckBody(cardwire.NewDeck(req))
	if err != nil {
		return nil, newError(server.CodeInvalidArgument, err)
	}
	wrote, err := h.scenarios().cards.Write.Deck(ctx, showing, req.Path, body, readFileStat(req.Baseline))
	// 	// A write that reached the vault has happened, so the response carries the
	// 	// file stat the client presents at its next save, and reports where the
	// 	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return &api.WriteDeckResponse{
			Error: api.ErrorCodeUnspecified, At: writeFileStat(wrote.FileStat), Bound: 0, IsUnlevelled: unlevelled,
		}, nil
	}
	if errors.Is(err, note.ErrTooLarge) {
		code := api.ErrorCodeDeckTooLarge
		return &api.WriteDeckResponse{
			Error: code, At: nil, Bound: cards.MaxBytes, IsUnlevelled: false,
		}, nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, newError(getCode(err), err)
	}
	return &api.WriteDeckResponse{Error: reason, At: nil, Bound: 0, IsUnlevelled: false}, nil
}

// WriteCardModel writes fields and templates into a card model. A card model that
// still matches what the client read is overwritten; a card model that changed is
// left unchanged and the response reports the change.
func (h *cardsHandler) WriteCardModel(
	ctx context.Context, req *api.WriteCardModelRequest,
) (*api.WriteCardModelResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.CardModelBody(
		req.Preamble, cardwire.NewTemplates(req.Templates), req.Tail)
	if err != nil {
		return nil, newError(server.CodeInvalidArgument, err)
	}
	pos, err := h.scenarios().cards.Write.CardModel(
		ctx, showing, req.Path, body, req.Fields, readFileStat(req.Baseline))
	// 	// A write that reached the vault has happened, so the response carries the
	// 	// file stat the client presents at its next save, and reports where the
	// 	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return &api.WriteCardModelResponse{
			Error: api.ErrorCodeUnspecified, At: writeFileStat(pos), IsUnlevelled: unlevelled,
		}, nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, newError(getCode(err), err)
	}
	return &api.WriteCardModelResponse{Error: reason, At: nil, IsUnlevelled: false}, nil
}

// errorCodeOfDeck is why a deck was not read. What holds for a note holds here,
// and a deck read at a bound of its own fails at that bound.
func errorCodeOfDeck(o note.ReadOutcome, is domain.NoteType) (api.ErrorCode, bool) {
	if o == note.TooLarge {
		return api.ErrorCodeDeckTooLarge, true
	}
	if o == note.Ok && is != format.TypeDeck {
		return api.ErrorCodeNotADeck, true
	}
	return convertOutcomeToErrorCode(o)
}

// errorCodeOfCardModel is why a card model was not read.
func errorCodeOfCardModel(o note.ReadOutcome, is domain.NoteType) (api.ErrorCode, bool) {
	if o == note.Ok && is != format.TypeCardModel {
		return api.ErrorCodeNotACardModel, true
	}
	return convertOutcomeToErrorCode(o)
}
