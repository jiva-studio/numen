package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
)

// mediaHandler serves MediaService: where a file of the shown vault or of a plugin's store of it is
// loaded from.
type mediaHandler struct {
	state          *State
	playing        *Loopback
	resolveAddress *file.ResolveAddress
	mediaTypes     *file.GetMediaType
}

// ResolveAddress returns the loopback address of the bytes of the file at the address and the media
// type for a player.
func (h *mediaHandler) ResolveAddress(
	ctx context.Context, r *connect.Request[v1.ResolveAddressRequest],
) (*connect.Response[v1.ResolveAddressResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if h.resolveAddress == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	resolved, err := h.resolveAddress.Execute(ctx, showing, r.Msg.GetAddress())
	switch {
	case errors.Is(err, file.ErrInvalidAddress):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, file.ErrNoPluginFiles):
		return nil, connect.NewError(connect.CodeUnavailable, err)
	case err != nil:
		return nil, connect.NewError(getListCode(err), err)
	}
	src := h.playing.Address(showing, resolved.Stat)
	if resolved.Address.IsPluginFile() {
		src = h.playing.PluginFileAddress(showing, resolved.Address, resolved.Size)
	}
	return connect.NewResponse(&v1.ResolveAddressResponse{
		Src:       src,
		MediaType: getMediaType(h.mediaTypes, resolved.Address.Path),
	}), nil
}

func getMediaType(types *file.GetMediaType, path string) string {
	if types == nil {
		return ""
	}
	return types.Execute(path)
}
