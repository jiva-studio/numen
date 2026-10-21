package service

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/service/presetwire"
)

// presetsHandler answers about the presets decks are scheduled by.
type presetsHandler struct {
	*Backend
}

// GetDeckPreset is the preset a deck is scheduled by. A deck naming none is
// answered with the defaults under no path.
func (h *presetsHandler) GetDeckPreset(
	ctx context.Context, r *connect.Request[v1.GetDeckPresetRequest],
) (*connect.Response[v1.GetDeckPresetResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.GetForDeck(ctx, showing, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &v1.GetDeckPresetResponse{Bounds: presetwire.GetSettingsBounds()}
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
	ctx context.Context, _ *connect.Request[v1.ListPresetsRequest],
) (*connect.Response[v1.ListPresetsResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	held, err := h.scenarios().flash.Presets.List(ctx, showing)
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &v1.ListPresetsResponse{Presets: make([]*v1.PresetSummary, 0, len(held))}
	for _, one := range held {
		out.Presets = append(out.Presets, &v1.PresetSummary{Path: one.Path, Title: one.Title})
	}
	return connect.NewResponse(out), nil
}

// CreatePreset puts a preset naming none of its settings in the vault. A key
// the file does not carry stands at the default.
func (h *presetsHandler) CreatePreset(
	ctx context.Context, r *connect.Request[v1.CreatePresetRequest],
) (*connect.Response[v1.CreatePresetResponse], error) {
	made, code, unlevelled, err := h.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.scenarios().cards.Create.Preset(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreatePresetResponse{
		Path: made.Path, Error: code, IsUnlevelled: unlevelled,
	}), nil
}

// ScheduleDeck puts a deck on a preset. A deck still holding what the client
// read is written; one holding something else is left alone and the client is
// told the deck changed.
func (h *presetsHandler) ScheduleDeck(
	ctx context.Context, r *connect.Request[v1.ScheduleDeckRequest],
) (*connect.Response[v1.ScheduleDeckResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	at, err := h.scenarios().flash.Presets.Point(
		ctx, showing, r.Msg.GetDeck(), r.Msg.GetPreset(), readFingerprint(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&v1.ScheduleDeckResponse{
			At: writeFingerprint(at), IsUnlevelled: unlevelled,
		}), nil
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := v1.ErrorCode_ERROR_CODE_NOT_A_PRESET
		return connect.NewResponse(&v1.ScheduleDeckResponse{Error: &reason}), nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&v1.ScheduleDeckResponse{Error: &reason}), nil
}

// ReadPreset is the settings of one preset.
func (h *presetsHandler) ReadPreset(
	ctx context.Context, r *connect.Request[v1.ReadPresetRequest],
) (*connect.Response[v1.ReadPresetResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	found, err := h.scenarios().flash.Presets.Read(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &v1.ReadPresetResponse{Bounds: presetwire.GetSettingsBounds()}
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
	ctx context.Context, r *connect.Request[v1.WritePresetRequest],
) (*connect.Response[v1.WritePresetResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	at, err := h.scenarios().flash.Presets.Save(ctx, showing, r.Msg.GetPath(), settings, readFingerprint(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	unlevelled := isUnlevelled(err)
	if err == nil || unlevelled {
		return connect.NewResponse(&v1.WritePresetResponse{
			At: writeFingerprint(at), IsUnlevelled: unlevelled,
		}), nil
	}
	// A value outside what a preset may hold is the client's to correct.
	if errors.Is(err, flashcards.ErrOutOfBounds) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := v1.ErrorCode_ERROR_CODE_NOT_A_PRESET
		return connect.NewResponse(&v1.WritePresetResponse{Error: &reason}), nil
	}
	reason, ok := getErrorCode(err)
	if !ok {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&v1.WritePresetResponse{Error: &reason}), nil
}

// ComputeCurve is what these settings come to over the whole range of the goal
// they name. Nothing is written: what a curve is asked for is the value a
// person is moving and has not settled.
func (h *presetsHandler) ComputeCurve(
	ctx context.Context, r *connect.Request[v1.ComputeCurveRequest],
) (*connect.Response[v1.ComputeCurveResponse], error) {
	showing, err := h.getCurrentVault(ctx)
	if err != nil {
		return nil, err
	}
	settings, err := presetwire.ConvertSettingsFromWire(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	held, err := h.scenarios().flash.Curves.Execute(ctx, showing, r.Msg.GetPath(), settings)
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}
	return connect.NewResponse(&v1.ComputeCurveResponse{Curve: presetwire.ConvertCurveToWire(held)}), nil
}
