package service

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/service/presetwire"
)

// GetVaultDeckPreset is the preset a deck of the named vault is scheduled by. A
// deck naming none is answered with the defaults under no path.
func (a *reviewHandler) GetVaultDeckPreset(
	ctx context.Context, r *connect.Request[v1.GetVaultDeckPresetRequest],
) (*connect.Response[v1.GetVaultDeckPresetResponse], error) {
	v, err := a.getVault(ctx, r.Msg.GetVault())
	if err != nil {
		return nil, err
	}
	found, err := a.scenarios().flash.Presets.GetForDeck(ctx, v, r.Msg.GetDeck())
	if err != nil {
		return nil, connect.NewError(getCode(err), err)
	}

	out := &v1.GetVaultDeckPresetResponse{}
	if reason, ok := convertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = &reason
		return connect.NewResponse(out), nil
	}
	out.Preset = presetwire.ConvertPresetToWire(found, a.getTitle(ctx, v, found.Path))
	return connect.NewResponse(out), nil
}
