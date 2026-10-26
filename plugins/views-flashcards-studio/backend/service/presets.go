package service

import (
	"context"
	"errors"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/notes/domain"
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
	ctx context.Context, r *connect.Request[flashcardsv1.GetDeckPresetRequest],
) (*connect.Response[flashcardsv1.GetDeckPresetResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.GetForDeck(ctx, showing, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &flashcardsv1.GetDeckPresetResponse{Bounds: presetwire.GetSettingsBounds()}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, h.getTitle(ctx, showing, found.Path))
	// What the file was when this came out of it, for the client to present
	// when it writes the settings back.
	out.At = writeFingerprint(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// ListPresets is every preset the vault holds, by path and by what it is
// called. A deck's preset is chosen from this list, and from the defaults,
// which are no note and are not in it.
func (h *presetsHandler) ListPresets(
	ctx context.Context, _ *connect.Request[flashcardsv1.ListPresetsRequest],
) (*connect.Response[flashcardsv1.ListPresetsResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	target, err := h.scenarios().flash.Presets.List(ctx, showing)
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &flashcardsv1.ListPresetsResponse{Presets: make([]*flashcardsv1.PresetSummary, 0, len(target))}
	for _, one := range target {
		out.Presets = append(out.Presets, &flashcardsv1.PresetSummary{Path: one.Path, Title: one.Title})
	}
	return connect.NewResponse(out), nil
}

// CreatePreset puts a preset naming none of its settings in the vault. A key
func (h *presetsHandler) CreatePreset(
	ctx context.Context, r *connect.Request[flashcardsv1.CreatePresetRequest],
) (*connect.Response[flashcardsv1.CreatePresetResponse], error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.scenarios().cards.Create.Preset(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&flashcardsv1.CreatePresetResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}), nil
}

// ScheduleDeck puts a deck on a preset. A deck still holding what the client
// read is written; one holding something else is left alone and the client is
// told the deck changed.
func (h *presetsHandler) ScheduleDeck(
	ctx context.Context, r *connect.Request[flashcardsv1.ScheduleDeckRequest],
) (*connect.Response[flashcardsv1.ScheduleDeckResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	pos, err := h.scenarios().flash.Presets.Point(
		ctx, showing, r.Msg.GetDeck(), r.Msg.GetPreset(), readFingerprint(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&flashcardsv1.ScheduleDeckResponse{
			At: writeFingerprint(pos), IsUnlevelled: unlevelled,
		}), nil
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := flashcardsv1.ErrorCode_ERROR_CODE_NOT_A_PRESET
		return connect.NewResponse(&flashcardsv1.ScheduleDeckResponse{Error: &reason}), nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&flashcardsv1.ScheduleDeckResponse{Error: &reason}), nil
}

// ReadPreset is the settings of one preset.
func (h *presetsHandler) ReadPreset(
	ctx context.Context, r *connect.Request[flashcardsv1.ReadPresetRequest],
) (*connect.Response[flashcardsv1.ReadPresetResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.Read(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &flashcardsv1.ReadPresetResponse{Bounds: presetwire.GetSettingsBounds()}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, h.getTitle(ctx, showing, found.Path))
	out.At = writeFingerprint(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// WritePreset puts settings into a preset. A note still holding what the client
// read is written; one holding something else is left alone and the client is
// told the note changed.
func (h *presetsHandler) WritePreset(
	ctx context.Context, r *connect.Request[flashcardsv1.WritePresetRequest],
) (*connect.Response[flashcardsv1.WritePresetResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	pos, err := h.scenarios().flash.Presets.Save(ctx, showing, r.Msg.GetPath(), settings, readFingerprint(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&flashcardsv1.WritePresetResponse{
			At: writeFingerprint(pos), IsUnlevelled: unlevelled,
		}), nil
	}
	// A value outside what a preset may hold is the client's to correct.
	if errors.Is(err, flashcards.ErrOutOfBounds) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := flashcardsv1.ErrorCode_ERROR_CODE_NOT_A_PRESET
		return connect.NewResponse(&flashcardsv1.WritePresetResponse{Error: &reason}), nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&flashcardsv1.WritePresetResponse{Error: &reason}), nil
}

// ComputeCurve is what these settings come to over the whole range of the goal
// they name. Nothing is written: what a curve is asked for is the value a
// person is moving and has not settled.
func (h *presetsHandler) ComputeCurve(
	ctx context.Context, r *connect.Request[flashcardsv1.ComputeCurveRequest],
) (*connect.Response[flashcardsv1.ComputeCurveResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	target, err := h.scenarios().flash.Curves.Execute(ctx, showing, r.Msg.GetPath(), settings)
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&flashcardsv1.ComputeCurveResponse{Curve: presetwire.ConvertCurveToWire(target)}), nil
}
