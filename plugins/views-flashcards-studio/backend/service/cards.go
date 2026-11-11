package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cardwire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/filewire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
)

// cardsHandler answers about the decks and card models a vault is arranged into.
type cardsHandler struct {
	getCurrentVault func(context.Context) (domain.Vault, error)
	createCardNote  func(context.Context, func(domain.Vault, cards.New) (cards.CreateNoteResult, error), string, string) (cards.CreateNoteResult, api.ErrorCode, bool, error)
	newTitler       func(context.Context, domain.Vault) func(string) string
	cards           func() cards.Scenarios
}

func (b *Backend) newCardsHandler() *cardsHandler {
	return &cardsHandler{
		getCurrentVault: b.getCurrentVault,
		createCardNote:  b.createCardNote,
		newTitler:       b.newTitler,
		cards:           func() cards.Scenarios { return b.scenarios().Cards },
	}
}

func (h *cardsHandler) ListCardModels(ctx context.Context, req *api.ListCardModelsRequest) (*api.ListCardModelsResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, total, err := h.cards().List.Execute(ctx, showing, cardwire.GetLimit(req.Limit))
	return cardwire.NewListResponse(found, total, err)
}

func (h *cardsHandler) CreateCardModel(ctx context.Context, req *api.CreateCardModelRequest) (*api.CreateCardModelResponse, error) {
	made, code, indexOutdated, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		in.Fields = req.Fields
		return h.cards().Create.CardModel(ctx, showing, in)
	}, req.Title, req.Path)
	if err != nil {
		return nil, err
	}
	return &api.CreateCardModelResponse{Path: made.Path, Error: code, IsIndexOutdated: indexOutdated}, nil
}

func (h *cardsHandler) CreateDeck(ctx context.Context, req *api.CreateDeckRequest) (*api.CreateDeckResponse, error) {
	made, code, indexOutdated, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.cards().Create.Deck(ctx, showing, in)
	}, req.Title, req.Path)
	if err != nil {
		return nil, err
	}
	return &api.CreateDeckResponse{Path: made.Path, Error: code, IsIndexOutdated: indexOutdated}, nil
}

func (b *Backend) createCardNote(
	ctx context.Context, create func(domain.Vault, cards.New) (cards.CreateNoteResult, error), title, folder string,
) (cards.CreateNoteResult, api.ErrorCode, bool, error) {
	showing, err := b.getCurrentVault(ctx)
	if err != nil {
		return cards.CreateNoteResult{}, api.ErrorCodeUnspecified, false, err
	}
	return apierror.ClassifyCreate(create(showing, cards.New{Title: title, Body: "", Path: folder, Fields: nil, Templates: nil}))
}

func (h *cardsHandler) RenameCardModelField(
	ctx context.Context, req *api.RenameCardModelFieldRequest,
) (*api.RenameCardModelFieldResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	renamed, err := h.cards().RenameField.Execute(ctx, showing, cards.Rename{
		CardModel: req.Path, From: req.From, To: req.To, FileStat: filewire.GetStat(req.Baseline),
	})
	return cardwire.ConvertRename(renamed, err)
}

func (h *cardsHandler) ReadCardModel(ctx context.Context, req *api.ReadCardModelRequest) (*api.ReadCardModelResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.cards().Read.CardModel(ctx, showing, req.Path)
	return cardwire.NewReadCardModelResponse(found, err, h.newTitler(ctx, showing))
}

func (h *cardsHandler) ReadDeck(ctx context.Context, req *api.ReadDeckRequest) (*api.ReadDeckResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.cards().Read.Deck(ctx, showing, req.Path)
	return cardwire.NewReadDeckResponse(found, err, h.newTitler(ctx, showing))
}

func (h *cardsHandler) WriteDeck(ctx context.Context, req *api.WriteDeckRequest) (*api.WriteDeckResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.DeckBody(cardwire.NewDeck(req))
	if err != nil {
		return nil, apierror.NewError(server.CodeInvalidArgument, err)
	}
	wrote, err := h.cards().Write.Deck(ctx, showing, req.Path, body, filewire.GetStat(req.Baseline))
	return cardwire.NewWriteDeckResponse(wrote.FileStat, err)
}

func (h *cardsHandler) WriteCardModel(ctx context.Context, req *api.WriteCardModelRequest) (*api.WriteCardModelResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	body, err := format.CardModelBody(req.Preamble, cardwire.NewTemplates(req.Templates), req.Tail)
	// gomutants:disable-next-line BRANCH_IF reason="CardModelBody normalizes template names to a single heading line."
	if err != nil {
		return nil, apierror.NewError(server.CodeInvalidArgument, err)
	}
	pos, err := h.cards().Write.CardModel(ctx, showing, req.Path, body, req.Fields, filewire.GetStat(req.Baseline))
	return cardwire.NewWriteCardModelResponse(pos, err)
}
