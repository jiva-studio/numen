package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// presetsHandler answers about the presets decks are scheduled by.
type presetsHandler struct {
	state    *State
	presets  *flashcards.Presets
	curves   *flashcards.ProjectCurve
	create   *cards.Create
	getTitle *note.GetTitle
}

// GetDeckPreset is the preset a deck is scheduled by. A deck naming none is
// answered with the defaults under no path.
func (h *presetsHandler) GetDeckPreset(
	ctx context.Context, r *connect.Request[v1.GetDeckPresetRequest],
) (*connect.Response[v1.GetDeckPresetResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	found, err := h.presets.GetForDeck(ctx, showing, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(wire.GetCode(err), err)
	}

	out := &v1.GetDeckPresetResponse{Bounds: wire.GetSettingsBounds()}
	if reason, refused := wire.ConvertOutcomeToErrorCode(found.Outcome); refused {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.ConvertPresetToWire(found, h.getTitle.Execute(ctx, showing, found.Path))
	// What the file was when this came out of it, for the client to present
	// when it writes the settings back.
	out.At = convertFingerprintToWire(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// ListPresets is every preset the vault holds, by path and by what it is
// called. A deck's preset is chosen from this list, and from the defaults,
// which are no note and are not in it.
func (h *presetsHandler) ListPresets(
	ctx context.Context, _ *connect.Request[v1.ListPresetsRequest],
) (*connect.Response[v1.ListPresetsResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	held, err := h.presets.List(ctx, showing)
	if err != nil {
		return nil, connect.NewError(wire.GetCode(err), err)
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
	made, code, unlevelled, err := h.state.createCardNote(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return h.create.Preset(ctx, showing, in)
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
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	at, err := h.presets.Point(
		ctx, showing, r.Msg.GetDeck(), r.Msg.GetPreset(), convertFingerprintFromWire(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	behind := h.state.isUnlevelled(err)
	if err == nil || behind {
		if h.state.NotifyNoteWritten != nil {
			h.state.NotifyNoteWritten()
		}
		return connect.NewResponse(&v1.ScheduleDeckResponse{
			At: convertFingerprintToWire(at), IsUnlevelled: behind,
		}), nil
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := v1.ErrorCode_ERROR_CODE_NOT_A_PRESET
		return connect.NewResponse(&v1.ScheduleDeckResponse{Error: &reason}), nil
	}
	reason, refused := wire.GetErrorCode(err)
	if !refused {
		return nil, connect.NewError(wire.GetCode(err), err)
	}
	return connect.NewResponse(&v1.ScheduleDeckResponse{Error: &reason}), nil
}

// ReadPreset is the settings of one preset.
func (h *presetsHandler) ReadPreset(
	ctx context.Context, r *connect.Request[v1.ReadPresetRequest],
) (*connect.Response[v1.ReadPresetResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	found, err := h.presets.Read(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(wire.GetCode(err), err)
	}

	out := &v1.ReadPresetResponse{Bounds: wire.GetSettingsBounds()}
	if reason, refused := wire.ConvertOutcomeToErrorCode(found.Outcome); refused {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.ConvertPresetToWire(found, h.getTitle.Execute(ctx, showing, found.Path))
	out.At = convertFingerprintToWire(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// WritePreset puts settings into a preset. A note still holding what the client
// read is written; one holding something else is left alone and the client is
// told the note changed.
func (h *presetsHandler) WritePreset(
	ctx context.Context, r *connect.Request[v1.WritePresetRequest],
) (*connect.Response[v1.WritePresetResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	settings, err := wire.ConvertSettingsFromWire(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	at, err := h.presets.Save(ctx, showing, r.Msg.GetPath(), settings, convertFingerprintFromWire(r.Msg.GetBaseline()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	behind := h.state.isUnlevelled(err)
	if err == nil || behind {
		if h.state.NotifyNoteWritten != nil {
			h.state.NotifyNoteWritten()
		}
		return connect.NewResponse(&v1.WritePresetResponse{
			At: convertFingerprintToWire(at), IsUnlevelled: behind,
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
	reason, refused := wire.GetErrorCode(err)
	if !refused {
		return nil, connect.NewError(wire.GetCode(err), err)
	}
	return connect.NewResponse(&v1.WritePresetResponse{Error: &reason}), nil
}

// ComputeCurve is what these settings come to over the whole range of the goal
// they name. Nothing is written: what a curve is asked for is the value a
// person is moving and has not settled.
func (h *presetsHandler) ComputeCurve(
	ctx context.Context, r *connect.Request[v1.ComputeCurveRequest],
) (*connect.Response[v1.ComputeCurveResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	settings, err := wire.ConvertSettingsFromWire(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	held, err := h.curves.Execute(ctx, showing, r.Msg.GetPath(), settings)
	if err != nil {
		return nil, connect.NewError(wire.GetCode(err), err)
	}
	return connect.NewResponse(&v1.ComputeCurveResponse{Curve: wire.ConvertCurveToWire(held)}), nil
}
