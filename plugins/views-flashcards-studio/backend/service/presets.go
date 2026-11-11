package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/filewire"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presets"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/presetwire"
)

// presetsHandler answers about the presets decks are scheduled by.
type presetsHandler struct {
	getCurrentVault func(context.Context) (domain.Vault, error)
	createCardNote  func(context.Context, func(domain.Vault, cards.New) (cards.CreateNoteResult, error), string, string) (cards.CreateNoteResult, api.ErrorCode, bool, error)
	newTitler       func(context.Context, domain.Vault) func(string) string
	create          func() cards.Create
	presets         func() presets.Presets
	curves          func() flashcards.ProjectCurve
}

func (b *Backend) newPresetsHandler() *presetsHandler {
	return &presetsHandler{
		getCurrentVault: b.getCurrentVault,
		createCardNote:  b.createCardNote,
		newTitler:       b.newTitler,
		create:          func() cards.Create { return b.scenarios().Cards.Create },
		presets:         func() presets.Presets { return b.scenarios().Flash.Presets },
		curves:          func() flashcards.ProjectCurve { return b.scenarios().Flash.Curves },
	}
}

func (h *presetsHandler) GetDeckPreset(ctx context.Context, req *api.GetDeckPresetRequest) (*api.GetDeckPresetResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.presets().GetForDeck(ctx, showing, req.Deck)
	return presetwire.NewDeckPresetResponse(found, err, h.newTitler(ctx, showing))
}

func (h *presetsHandler) ListPresets(ctx context.Context, _ *api.ListPresetsRequest) (*api.ListPresetsResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.presets().List(ctx, showing)
	return presetwire.NewListResponse(found, err)
}

func (h *presetsHandler) CreatePreset(ctx context.Context, req *api.CreatePresetRequest) (*api.CreatePresetResponse, error) {
	made, code, indexOutdated, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.create().Preset(ctx, showing, in)
	}, req.Title, req.Path)
	if err != nil {
		return nil, err
	}
	return &api.CreatePresetResponse{Path: made.Path, Error: code, IsIndexOutdated: indexOutdated}, nil
}

func (h *presetsHandler) ScheduleDeck(ctx context.Context, req *api.ScheduleDeckRequest) (*api.ScheduleDeckResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	pos, err := h.presets().Point(ctx, showing, req.Deck, req.Preset, filewire.GetStat(req.Baseline))
	return presetwire.NewScheduleDeckResponse(pos, err)
}

func (h *presetsHandler) ReadPreset(ctx context.Context, req *api.ReadPresetRequest) (*api.ReadPresetResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.presets().Read(ctx, showing, req.Path)
	return presetwire.NewReadPresetResponse(found, err, h.newTitler(ctx, showing))
}

func (h *presetsHandler) WritePreset(ctx context.Context, req *api.WritePresetRequest) (*api.WritePresetResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(req.Settings)
	if err != nil {
		return nil, apierror.NewError(server.CodeInvalidArgument, err)
	}
	pos, err := h.presets().Save(ctx, showing, req.Path, settings, filewire.GetStat(req.Baseline))
	return presetwire.NewWritePresetResponse(pos, err)
}

func (h *presetsHandler) ComputeCurve(ctx context.Context, req *api.ComputeCurveRequest) (*api.ComputeCurveResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(req.Settings)
	if err != nil {
		return nil, apierror.NewError(server.CodeInvalidArgument, err)
	}
	curve, err := h.curves().Execute(ctx, showing, req.Path, settings)
	return presetwire.NewCurveResponse(curve, err)
}
