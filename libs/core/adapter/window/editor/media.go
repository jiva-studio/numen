package editor

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// mediaHandler serves MediaService: where a file of the shown vault is loaded from to be played.
type mediaHandler struct {
	state          *State
	playing        *Loopback
	getFingerprint *file.GetFingerprint
	openCopy       *file.OpenCopy
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
	if h.getFingerprint == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	ref, err := h.getFingerprint.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getListCode(err), err)
	}
	if h.openCopy != nil {
		played, found, err := h.openCopy.Execute(ctx, showing, ref.Path)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if found {
			_ = played.Body.Close()
			return connect.NewResponse(&v1.GetMediaSourceResponse{
				Src:       h.playing.Address(showing, getCopyFingerprint(ref.Path, played.Size)),
				MediaType: played.MediaType,
			}), nil
		}
	}
	return connect.NewResponse(&v1.GetMediaSourceResponse{
		Src:       h.playing.Address(showing, ref),
		MediaType: domain.MediaType(ref.Path),
	}), nil
}

// getCopyFingerprint is what an address of a copy names: the file it is played in place of, and the
// size of the copy.
func getCopyFingerprint(path string, size int64) domain.Fingerprint {
	return domain.Fingerprint{Path: path, Size: size}
}
