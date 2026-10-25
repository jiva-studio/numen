package editor

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// recordingHandler answers where a file of the shown vault is played from.
type recordingHandler struct {
	state          *State
	playing        *Loopback
	getFingerprint *file.GetFingerprint
}

// GetRecording answers the address the loopback socket serves the file's bytes at, and the type a
// player is told. The duration is left to the player.
func (h *recordingHandler) GetRecording(
	ctx context.Context, r *connect.Request[v1.GetRecordingRequest],
) (*connect.Response[v1.GetRecordingResponse], error) {
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
	return connect.NewResponse(&v1.GetRecordingResponse{
		MediaUrl:  h.playing.Address(showing, ref),
		MediaType: domain.MediaType(ref.Path),
	}), nil
}
