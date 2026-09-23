package flashcards

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// GetVaultDeckPreset is the preset a deck of the named vault is scheduled by. A
// deck naming none is answered with the defaults under no path.
func (a *API) GetVaultDeckPreset(
	ctx context.Context, r *connect.Request[v1.GetVaultDeckPresetRequest],
) (*connect.Response[v1.GetVaultDeckPresetResponse], error) {
	v, err := a.Vault(r.Msg.GetVault())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	found, err := a.Presets.GetForDeck(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(wire.GetCode(err), err)
	}

	out := &v1.GetVaultDeckPresetResponse{}
	if reason, refused := wire.ErrorCodeOf(found.Outcome); refused {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = wire.PresetOf(found, wire.GetTitle(ctx, a.Notes, v.ID, found.Path))
	return connect.NewResponse(out), nil
}
