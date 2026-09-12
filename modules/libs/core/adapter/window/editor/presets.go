package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// GetDeckPreset is the preset a deck is scheduled by. A deck naming none is
// answered with the defaults under no path.
func (a *API) GetDeckPreset(
	ctx context.Context, r *connect.Request[v1.GetDeckPresetRequest],
) (*connect.Response[v1.GetDeckPresetResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Presets.Of(ctx, showing, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(wire.Coded(err), err)
	}

	out := &v1.GetDeckPresetResponse{Bounds: wire.SettingsBounds()}
	if reason, refused := wire.ErrorCodeOf(found.Outcome); refused {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.PresetOf(found, wire.Titled(ctx, a.Notes.Queries, showing.ID, found.Path))
	// What the file was when this came out of it, for the client to present
	// when it writes the settings back.
	out.At = fingerprintOf(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// ListPresets is every preset the vault holds, by path and by what it is
// called. A deck's preset is chosen from this list, and from the defaults,
// which are no note and are not in it.
func (a *API) ListPresets(
	ctx context.Context, _ *connect.Request[v1.ListPresetsRequest],
) (*connect.Response[v1.ListPresetsResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	held, err := a.Presets.List(ctx, showing)
	if err != nil {
		return nil, connect.NewError(wire.Coded(err), err)
	}

	out := &v1.ListPresetsResponse{Presets: make([]*v1.PresetSummary, 0, len(held))}
	for _, one := range held {
		out.Presets = append(out.Presets, &v1.PresetSummary{Path: one.Path, Title: one.Title})
	}
	return connect.NewResponse(out), nil
}

// CreatePreset puts a preset naming none of its settings in the vault. A key
// the file does not carry stands at the default.
func (a *API) CreatePreset(
	ctx context.Context, r *connect.Request[v1.CreatePresetRequest],
) (*connect.Response[v1.CreatePresetResponse], error) {
	made, code, unlevelled, err := a.makes(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return a.Cards.Create.Preset(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreatePresetResponse{
		Path: made.Path, Error: code, Unlevelled: unlevelled,
	}), nil
}

// ScheduleDeck puts a deck on a preset. A deck still holding what the client
// read is written; one holding something else is left alone and the client is
// told the deck changed.
func (a *API) ScheduleDeck(
	ctx context.Context, r *connect.Request[v1.ScheduleDeckRequest],
) (*connect.Response[v1.ScheduleDeckResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	at, err := a.Presets.Point(
		ctx, showing, r.Msg.GetDeck(), r.Msg.GetPreset(), refOf(r.Msg.GetSeen()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	behind := a.unlevelled(err)
	if err == nil || behind {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.ScheduleDeckResponse{
			At: fingerprintOf(at), Unlevelled: behind,
		}), nil
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := v1.ErrorCode_ERROR_CODE_NOT_A_PRESET
		return connect.NewResponse(&v1.ScheduleDeckResponse{Error: &reason}), nil
	}
	reason, refused := wire.ErrorCodeBy(err)
	if !refused {
		return nil, connect.NewError(wire.Coded(err), err)
	}
	return connect.NewResponse(&v1.ScheduleDeckResponse{Error: &reason}), nil
}

// ReadPreset is the settings of one preset.
func (a *API) ReadPreset(
	ctx context.Context, r *connect.Request[v1.ReadPresetRequest],
) (*connect.Response[v1.ReadPresetResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Presets.Read(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(wire.Coded(err), err)
	}

	out := &v1.ReadPresetResponse{Bounds: wire.SettingsBounds()}
	if reason, refused := wire.ErrorCodeOf(found.Outcome); refused {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.PresetOf(found, wire.Titled(ctx, a.Notes.Queries, showing.ID, found.Path))
	out.At = fingerprintOf(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// WritePreset puts settings into a preset. A note still holding what the client
// read is written; one holding something else is left alone and the client is
// told the note changed.
func (a *API) WritePreset(
	ctx context.Context, r *connect.Request[v1.WritePresetRequest],
) (*connect.Response[v1.WritePresetResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	settings, err := wire.SettingsIn(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	at, err := a.Presets.Save(ctx, showing, r.Msg.GetPath(), settings, refOf(r.Msg.GetSeen()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save, and told where the
	// index did not follow.
	behind := a.unlevelled(err)
	if err == nil || behind {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WritePresetResponse{
			At: fingerprintOf(at), Unlevelled: behind,
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
	reason, refused := wire.ErrorCodeBy(err)
	if !refused {
		return nil, connect.NewError(wire.Coded(err), err)
	}
	return connect.NewResponse(&v1.WritePresetResponse{Error: &reason}), nil
}

// ComputeCurve is what these settings come to over the whole range of the goal
// they name. Nothing is written: what a curve is asked for is the value a
// person is moving and has not settled.
func (a *API) ComputeCurve(
	ctx context.Context, r *connect.Request[v1.ComputeCurveRequest],
) (*connect.Response[v1.ComputeCurveResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	settings, err := wire.SettingsIn(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	held, err := a.Curves.Execute(ctx, showing, r.Msg.GetPath(), settings)
	if err != nil {
		return nil, connect.NewError(wire.Coded(err), err)
	}
	return connect.NewResponse(&v1.ComputeCurveResponse{Curve: wire.CurveOf(held)}), nil
}
