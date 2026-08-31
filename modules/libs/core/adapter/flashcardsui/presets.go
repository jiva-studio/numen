package flashcardsui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/refusal"
)

// Scheduling is the preset a deck of the named vault is scheduled by. A deck
// naming none is answered with the defaults under no path.
func (a *API) Scheduling(
	ctx context.Context, r *connect.Request[v1.FlashcardsServiceSchedulingRequest],
) (*connect.Response[v1.FlashcardsServiceSchedulingResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	found, err := a.Presets.Of(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}

	out := &v1.FlashcardsServiceSchedulingResponse{}
	if reason, refused := refusal.Of(found.Outcome); refused {
		out.Refusal = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.PresetOf(found, a.titled(ctx, v, found.Path))
	return connect.NewResponse(out), nil
}

// Curve is what these settings come to over the whole range of the goal they
// name. Nothing is written: a curve is asked for the value a person is moving
// and has not settled.
func (a *API) Curve(
	ctx context.Context, r *connect.Request[v1.FlashcardsServiceCurveRequest],
) (*connect.Response[v1.FlashcardsServiceCurveResponse], error) {
	v, err := a.Vault(r.Msg.GetVaultId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	settings, err := wire.SettingsIn(r.Msg.GetSettings())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	held, err := a.Curves.Execute(ctx, v, r.Msg.GetPath(), settings)
	if err != nil {
		return nil, connect.NewError(refusal.Coded(err), err)
	}
	return connect.NewResponse(&v1.FlashcardsServiceCurveResponse{Curve: wire.CurveOf(held)}), nil
}

// titled is what the vault calls the note at a path. A build with no index, and
// a path the index holds no note at, are answered with no name.
func (a *API) titled(ctx context.Context, v domain.Vault, path string) string {
	if a.Notes == nil || path == "" {
		return ""
	}
	found, err := a.Notes.Notes(ctx, v.ID, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}
