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
	v, err := a.Vault(r.Msg.GetVault())
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

// titled is what the vault calls the note at a path. A build with no index, and
// a path the index holds no note at, are answered with no name.
func (a *API) titled(ctx context.Context, v domain.Vault, path string) string {
	if a.Notes == nil || path == "" {
		return ""
	}
	found, err := a.Notes.Notes(ctx, string(v.ID), []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}
