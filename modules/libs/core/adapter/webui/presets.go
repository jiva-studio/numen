package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/refusal"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// errNoPresets is what a build with no path to the presets of a vault answers.
var errNoPresets = errors.New("this build cannot work the presets of a vault")

// Scheduling is the preset a deck is scheduled by. A deck naming none is
// answered with the defaults under no path.
func (a *API) Scheduling(
	ctx context.Context, r *connect.Request[v1.SchedulingRequest],
) (*connect.Response[v1.SchedulingResponse], error) {
	if a.Presets == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPresets)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Presets.Of(ctx, showing, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}

	out := &v1.SchedulingResponse{}
	if reason, refused := refusal.Of(found.Outcome); refused {
		out.Refusal = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.PresetOf(found, a.titled(ctx, showing, found.Path))
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
	if a.Presets == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPresets)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	held, err := a.Presets.List(ctx, showing)
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}

	out := &v1.ListPresetsResponse{Presets: make([]*v1.Listed, 0, len(held))}
	for _, one := range held {
		out.Presets = append(out.Presets, &v1.Listed{Path: one.Path, Title: one.Title})
	}
	return connect.NewResponse(out), nil
}

// MakePreset puts a preset naming none of its settings in the vault. A key the
// file does not carry stands at the default.
func (a *API) MakePreset(
	ctx context.Context, r *connect.Request[v1.MakePresetRequest],
) (*connect.Response[v1.MakePresetResponse], error) {
	made, refused, err := a.makes(ctx, func(showing domain.Vault, in cards.New) (cards.CreateNoteResult, error) {
		return a.MakesCards.Preset(ctx, showing, in)
	}, r.Msg.GetTitle(), r.Msg.GetFolder())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.MakePresetResponse{Path: made.Path, Refusal: refused}), nil
}

// Schedule puts a deck on a preset. A deck still holding what the client read
// is written; one holding something else is left alone and the client is told
// the deck changed.
func (a *API) Schedule(
	ctx context.Context, r *connect.Request[v1.ScheduleRequest],
) (*connect.Response[v1.ScheduleResponse], error) {
	if a.Presets == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPresets)
	}
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
	// handed the fingerprint it presents at its next save.
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.ScheduleResponse{At: fingerprintOf(at)}), nil
	}
	if errors.Is(err, port.ErrChanged) {
		return connect.NewResponse(&v1.ScheduleResponse{Changed: true}), nil
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := v1.Refusal_REFUSAL_NOT_A_PRESET
		return connect.NewResponse(&v1.ScheduleResponse{Refusal: &reason}), nil
	}
	reason, refused := refusal.By(err)
	if !refused {
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.ScheduleResponse{Refusal: &reason}), nil
}

// ReadPreset is the settings of one preset.
func (a *API) ReadPreset(
	ctx context.Context, r *connect.Request[v1.ReadPresetRequest],
) (*connect.Response[v1.ReadPresetResponse], error) {
	if a.Presets == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPresets)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Presets.Read(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}

	out := &v1.ReadPresetResponse{}
	if reason, refused := refusal.Of(found.Outcome); refused {
		out.Refusal = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.PresetOf(found, a.titled(ctx, showing, found.Path))
	out.At = fingerprintOf(found.Fingerprint)
	return connect.NewResponse(out), nil
}

// WritePreset puts settings into a preset. A note still holding what the client
// read is written; one holding something else is left alone and the client is
// told the note changed.
func (a *API) WritePreset(
	ctx context.Context, r *connect.Request[v1.WritePresetRequest],
) (*connect.Response[v1.WritePresetResponse], error) {
	if a.Presets == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPresets)
	}
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
	// handed the fingerprint it presents at its next save.
	if err == nil || errors.Is(err, note.ErrUnlevelled) {
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WritePresetResponse{At: fingerprintOf(at)}), nil
	}
	if errors.Is(err, port.ErrChanged) {
		return connect.NewResponse(&v1.WritePresetResponse{Changed: true}), nil
	}
	// A value outside what a preset may hold is the client's to correct.
	if errors.Is(err, flashcards.ErrOutOfBounds) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, flashcards.ErrNotAPreset) {
		reason := v1.Refusal_REFUSAL_NOT_A_PRESET
		return connect.NewResponse(&v1.WritePresetResponse{Refusal: &reason}), nil
	}
	reason, refused := refusal.By(err)
	if !refused {
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.WritePresetResponse{Refusal: &reason}), nil
}

// Curve is what these settings come to over the whole range of the goal they
// name. Nothing is written: what a curve is asked for is the value a person is
// moving and has not settled.
func (a *API) Curve(
	ctx context.Context, r *connect.Request[v1.CurveRequest],
) (*connect.Response[v1.CurveResponse], error) {
	if a.Curves == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPresets)
	}
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
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.CurveResponse{Curve: wire.CurveOf(held)}), nil
}
