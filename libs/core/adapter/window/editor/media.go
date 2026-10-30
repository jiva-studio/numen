package editor

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// mediaHandler serves MediaService: where a file of the shown vault is loaded from to be played.
type mediaHandler struct {
	state      *State
	playing    *Loopback
	statFile   *file.StatFile
	openCopy   *file.OpenProxy
	mediaTypes *file.GetMediaType
}

// GetMediaSource returns the loopback address of the file's bytes and the media type a player is
// told. A file a plugin keeps a copy of is loaded from the copy.
func (h *mediaHandler) GetMediaSource(
	ctx context.Context, r *connect.Request[v1.GetMediaSourceRequest],
) (*connect.Response[v1.GetMediaSourceResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if h.statFile == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	ref, err := h.statFile.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getListCode(err), err)
	}
	if h.openCopy != nil {
		played, found, err := h.openCopy.Execute(ctx, showing, ref.Path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if found {
			closeAndReport(h.state.reportTask, played.Body)
			return connect.NewResponse(&v1.GetMediaSourceResponse{
				Src:       h.playing.Address(showing, getCopyFileStat(ref.Path, played.Size)),
				MediaType: played.MediaType,
			}), nil
		}
	}
	return connect.NewResponse(&v1.GetMediaSourceResponse{
		Src:       h.playing.Address(showing, ref),
		MediaType: getMediaType(h.mediaTypes, ref.Path),
	}), nil
}

// getCopyFileStat is what an address of a copy names: the file it is played in place of, and the
// size of the copy.
func getCopyFileStat(path string, size int64) domain.FileStat {
	return domain.FileStat{Path: path, Size: size}
}

// GetPageSource returns the loopback address of a page a plugin serves below its route.
func (h *mediaHandler) GetPageSource(
	_ context.Context, r *connect.Request[v1.GetPageSourceRequest],
) (*connect.Response[v1.GetPageSourceResponse], error) {
	if r.Msg.GetPluginId() == "" || strings.ContainsAny(r.Msg.GetPluginId(), "/?#") {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoPlugin)
	}
	return connect.NewResponse(&v1.GetPageSourceResponse{
		Src: h.playing.PageAddress(r.Msg.GetPluginId(), r.Msg.GetPath()),
	}), nil
}

// errNoPlugin is a page asked of no plugin id.
var errNoPlugin = errors.New("the request names no plugin")

// getMediaType is the content type a file is served as, and empty where nothing names one.
func getMediaType(types *file.GetMediaType, path string) string {
	if types == nil {
		return ""
	}
	return types.Execute(path)
}
