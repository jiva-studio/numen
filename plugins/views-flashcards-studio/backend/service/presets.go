package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/service/presetwire"
)

// presetsHandler answers about the presets decks are scheduled by.
type presetsHandler struct {
	*Backend
}

// GetDeckPreset is the preset a deck is scheduled by. A deck naming none is
// answered with the defaults under no path.
func (h *presetsHandler) GetDeckPreset(
	ctx context.Context, req *api.GetDeckPresetRequest,
) (*api.GetDeckPresetResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.GetForDeck(ctx, showing, req.Deck)
	if err != nil {
		return nil, newError(getCode(err), err)
	}

	out := &api.GetDeckPresetResponse{
		Preset: nil, Error: api.ErrorCodeUnspecified, At: nil, Bounds: presetwire.GetSettingsBounds(),
	}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = reason
		return out, nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, h.getTitle(ctx, showing, found.Path))
	// What the file was when this came out of it, for the client to present
	// when it writes the settings back.
	out.At = writeFileStat(found.FileStat)
	return out, nil
}

// ListPresets is every preset the vault holds, by path and by what it is
// called. A deck's preset is chosen from this list, and from the defaults,
// which are no note and are not in it.
func (h *presetsHandler) ListPresets(
	ctx context.Context, _ *api.ListPresetsRequest,
) (*api.ListPresetsResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	target, err := h.scenarios().flash.Presets.List(ctx, showing)
	if err != nil {
		return nil, newError(getCode(err), err)
	}

	out := &api.ListPresetsResponse{Presets: make([]*api.PresetSummary, 0, len(target))}
	for _, one := range target {
		out.Presets = append(out.Presets, &api.PresetSummary{Path: one.Path, Title: one.Title})
	}
	return out, nil
}

// CreatePreset puts a preset naming none of its settings in the vault. A key
func (h *presetsHandler) CreatePreset(
	ctx context.Context, req *api.CreatePresetRequest,
) (*api.CreatePresetResponse, error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.scenarios().cards.Create.Preset(ctx, showing, in)
	}, req.Title, req.Path)
	if err != nil {
		return nil, err
	}
	return &api.CreatePresetResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}, nil
}

// ScheduleDeck assigns a deck to a preset. A deck whose content matches what the
// client read is written; a deck with different content is left unchanged and the
// client is informed that the deck changed.
func (h *presetsHandler) ScheduleDeck(
	ctx context.Context, req *api.ScheduleDeckRequest,
) (*api.ScheduleDeckResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	pos, err := h.scenarios().flash.Presets.Point(
		ctx, showing, req.Deck, req.Preset, readFileStat(req.Baseline))
	// A write that reached the vault has happened, so the client receives the file
	// stat to present at its next save, and is informed where the index did not
	// follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return &api.ScheduleDeckResponse{
			Error: api.ErrorCodeUnspecified, At: writeFileStat(pos), IsUnlevelled: unlevelled,
		}, nil
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := api.ErrorCodeNotAPreset
		return &api.ScheduleDeckResponse{Error: reason, At: nil, IsUnlevelled: false}, nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, newError(getCode(err), err)
	}
	return &api.ScheduleDeckResponse{Error: reason, At: nil, IsUnlevelled: false}, nil
}

// ReadPreset is the settings of one preset.
func (h *presetsHandler) ReadPreset(
	ctx context.Context, req *api.ReadPresetRequest,
) (*api.ReadPresetResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.Read(ctx, showing, req.Path)
	if err != nil {
		return nil, newError(getCode(err), err)
	}

	out := &api.ReadPresetResponse{
		Preset: nil, Error: api.ErrorCodeUnspecified, At: nil, Bounds: presetwire.GetSettingsBounds(),
	}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = reason
		return out, nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, h.getTitle(ctx, showing, found.Path))
	out.At = writeFileStat(found.FileStat)
	return out, nil
}

// WritePreset writes settings into a preset. A note whose content matches what
// the client read is written; a note with different content is left unchanged
// and the client is informed that the note changed.
func (h *presetsHandler) WritePreset(
	ctx context.Context, req *api.WritePresetRequest,
) (*api.WritePresetResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(req.Settings)
	if err != nil {
		return nil, newError(server.CodeInvalidArgument, err)
	}
	pos, err := h.scenarios().flash.Presets.Save(ctx, showing, req.Path, settings, readFileStat(req.Baseline))
	// A write that reached the vault has happened, so the client receives the file
	// stat to present at its next save, and is informed where the index did not
	// follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return &api.WritePresetResponse{
			Error: api.ErrorCodeUnspecified, At: writeFileStat(pos), IsUnlevelled: unlevelled,
		}, nil
	}
	// A value outside what a preset may hold is the client's to correct.
	if errors.Is(err, flashcards.ErrOutOfBounds) {
		return nil, newError(server.CodeInvalidArgument, err)
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := api.ErrorCodeNotAPreset
		return &api.WritePresetResponse{Error: reason, At: nil, IsUnlevelled: false}, nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, newError(getCode(err), err)
	}
	return &api.WritePresetResponse{Error: reason, At: nil, IsUnlevelled: false}, nil
}

// ComputeCurve computes what these settings amount to over the whole range of
// the goal they name. Nothing is written: a curve is requested for a value the
// user is adjusting and has not committed.
func (h *presetsHandler) ComputeCurve(
	ctx context.Context, req *api.ComputeCurveRequest,
) (*api.ComputeCurveResponse, error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(req.Settings)
	if err != nil {
		return nil, newError(server.CodeInvalidArgument, err)
	}
	target, err := h.scenarios().flash.Curves.Execute(ctx, showing, req.Path, settings)
	if err != nil {
		return nil, newError(getCode(err), err)
	}
	return &api.ComputeCurveResponse{Curve: presetwire.ConvertCurveToWire(target)}, nil
}
