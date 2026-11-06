package editor

import (
	"context"
	"errors"
	"io/fs"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// mediaHandler serves MediaService: where a file of the shown vault or of a plugin's store of it is
// loaded from.
type mediaHandler struct {
	state          *State
	playing        *Loopback
	statFile       *file.StatFile
	openPluginFile *file.OpenPluginFile
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
	address, err := file.ParseAddress(r.Msg.GetAddress())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if address.IsPluginFile() {
		return h.resolvePluginFile(ctx, showing, address)
	}
	if h.statFile == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	ref, err := h.statFile.Execute(ctx, showing, address.Path)
	if err != nil {
		return nil, connect.NewError(getListCode(err), err)
	}
	return connect.NewResponse(&v1.ResolveAddressResponse{
		Src:       h.playing.Address(showing, ref),
		MediaType: getMediaType(h.mediaTypes, ref.Path),
	}), nil
}

// resolvePluginFile is the address of a file in a plugin's store of the shown vault, which names the
// size the file has now.
func (h *mediaHandler) resolvePluginFile(
	ctx context.Context, showing domain.Vault, address file.Address,
) (*connect.Response[v1.ResolveAddressResponse], error) {
	if h.openPluginFile == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	opened, err := h.openPluginFile.Execute(ctx, showing, address.PluginID, address.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	closeAndReport(h.state.reportTask, opened.Body)
	return connect.NewResponse(&v1.ResolveAddressResponse{
		Src:       h.playing.PluginFileAddress(showing, address, opened.Size),
		MediaType: getMediaType(h.mediaTypes, address.Path),
	}), nil
}

// getMediaType is the content type a file is served as, and empty where nothing names one.
func getMediaType(types *file.GetMediaType, path string) string {
	if types == nil {
		return ""
	}
	return types.Execute(path)
}
