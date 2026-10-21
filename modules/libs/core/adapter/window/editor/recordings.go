package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// recordingHandler answers what a recording is and where it is played from.
type recordingHandler struct {
	files              *artifactFiles
	playing            *Loopback
	getHeardTranscript source.GetHeardTranscript
	getCopyLocation    source.GetCopyLocation
}

// A recording crosses to the window twice: as the bytes a player is pointed at,
// and as the words a model heard in them.

// errNotARecording is what a facet of a recording answers for a file that is
// not one.
var errNotARecording = errors.New("not a recording this vault holds")

// GetRecording answers what the file at a path is: how long it runs and where
// its bytes are played from. What has been made from it is ListArtifacts.
//
// A recording nothing has listened to reaches nowhere, and the player it is
// loaded into is what then says how long it runs.
func (h *recordingHandler) GetRecording(
	ctx context.Context,
	r *connect.Request[v1.GetRecordingRequest],
) (*connect.Response[v1.GetRecordingResponse], error) {
	showing, ref, err := h.files.recording(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	received, err := h.getHeardTranscript.Execute(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	transcribed, _ := transcript.ReadReached(received.Words)
	_, cues := transcript.Parse(received.Words)
	// Where a recording is played from and what it is played as are answered
	// here: the socket is opened afresh for every run, and what counts as a
	// recording is this application's to say.
	out := &v1.GetRecordingResponse{
		DurationMs: int32(transcribed),
		MediaUrl:   h.playing.Address(showing, ref),
		MediaType:  domain.MediaType(ref.Path),
	}
	// A url plays the copy fetched for it, where one stands. One with none is
	// framed at the address instead, from the socket this run opened, and the
	// frame is a page whatever is inside it.
	if ref.Kind == domain.KindURL {
		at := h.files.getArtifactAddress(ctx, showing, ref)
		out.MediaUrl, out.MediaType = h.getCopyMedia(ctx, showing, ref)
		out.Url = string(at)
		if out.MediaUrl == "" {
			out.MediaUrl, out.MediaType = h.playing.Embed(at), asAPage
		}
	}
	if len(cues) > 0 {
		out.DurationMs = max(out.DurationMs, int32(cues[len(cues)-1].To))
	}
	return connect.NewResponse(out), nil
}

// asAPage is what a url with no copy is played as: a page, in a frame, whatever
// the site puts inside it.
const asAPage = "text/html"

// getCopyMedia is where the copy fetched for a url is played from, and what a
// player is told it is. One nothing has been fetched a copy for plays from
// nowhere, which is what leaves the tab framing the address.
func (h *recordingHandler) getCopyMedia(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint,
) (media, kind string) {
	at := h.files.getArtifactAddress(ctx, v, ref)
	location := h.getCopyLocation.Execute(ctx, v, ref.Path, at)
	if !location.Exists {
		return "", ""
	}
	// The address names how large the copy was when it was given out, as a
	// recording's names the bytes it was: a copy fetched again is another
	// address. A copy kept in the vault is a file of the vault, played the way
	// every file of it is played.
	played := ref.Path
	if location.VaultPath != "" {
		played = location.VaultPath
	}
	return h.playing.Address(v, domain.Fingerprint{Path: played, Size: location.Size}), domain.CopyType
}
