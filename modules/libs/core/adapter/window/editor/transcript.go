package editor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// transcriptHandler answers with the words of a file that carries times, and
// writes them back as a person left them.
type transcriptHandler struct {
	state           *State
	files           *artifactFiles
	getTranscript   source.GetTranscript
	writeTranscript source.WriteTranscript
}

// errBeingHeard is what an edit to a recording a run holds gets.
var errBeingHeard = errors.New("this recording is being listened to")

// ReadTranscript answers with the words of a file that carries times: each
// stretch of speech against the milliseconds it was spoken in. A file nothing
// has been heard for holds none, which is an answer.
func (h *transcriptHandler) ReadTranscript(
	ctx context.Context,
	r *connect.Request[v1.ReadTranscriptRequest],
) (*connect.Response[v1.ReadTranscriptResponse], error) {
	showing, ref, err := h.files.getArtifactFile(
		ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT)
	if err != nil {
		return nil, err
	}
	standing, err := h.getTranscript.Execute(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	_, cues := transcript.Parse(standing.Words)
	if at := r.Msg.GetSpan(); at != nil {
		if cues, err = selectSpan(at)(cues); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return connect.NewResponse(&v1.ReadTranscriptResponse{Cues: newWireCues(cues)}), nil
}

// WriteTranscript writes the transcript of a recording as a person left it.
//
// What the model heard stays under its own name and the words as they now stand
// go beside it, so a transcript edited into nonsense is corrections that can be
// taken away and what was heard comes back.
func (h *transcriptHandler) WriteTranscript(
	ctx context.Context,
	r *connect.Request[v1.WriteTranscriptRequest],
) (*connect.Response[v1.WriteTranscriptResponse], error) {
	cues, err := parseCues(r.Msg.GetCues())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	showing, ref, err := h.files.recording(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	written, err := h.writeTranscript.Execute(ctx, showing, ref.Path, cues, h.state.getRechunker())
	switch {
	case errors.Is(err, source.ErrNoTranscript):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, source.ErrTranscriptBusy):
		return nil, connect.NewError(connect.CodeFailedPrecondition, errBeingHeard)
	case err != nil:
		return nil, connect.NewError(getReachCode(err), err)
	}
	// A write that landed is not refused for a cut that could not be asked for.
	if written.CutError != nil {
		h.state.reportTask(run.Task{ID: extractingBooks, Title: "Extracting books", About: ref.Path, Error: written.CutError.Error()})
	}
	return connect.NewResponse(&v1.WriteTranscriptResponse{Cues: newWireCues(cues)}), nil
}

// newWireCues is a transcript as a caller reads it.
func newWireCues(cues []domain.TranscriptCue) []*v1.Cue {
	out := make([]*v1.Cue, 0, len(cues))
	for _, one := range cues {
		out = append(out, &v1.Cue{Text: one.Text, From: int32(one.From), To: int32(one.To)})
	}
	return out
}

// parseCues is a transcript from the window as the cues it is written down as,
// and why it is not one where it cannot be.
//
// Speech runs forward: a cue ends no earlier than it begins, and begins after
// the one before it ends. A cue whose words trim away is dropped, and its
// timings still bound the cue after it.
//
// A transcript carrying no words at all is refused.
func parseCues(cues []*v1.Cue) ([]domain.TranscriptCue, error) {
	out := make([]domain.TranscriptCue, 0, len(cues))
	last := domain.TranscriptCue{From: -1, To: -1}
	for at, one := range cues {
		from, to := int(one.GetFrom()), int(one.GetTo())
		switch {
		case from < 0 || to < from:
			return nil, fmt.Errorf("cue %d: %d to %d is not a stretch of a recording", at, from, to)
		case from < last.From:
			return nil, fmt.Errorf("cue %d: begins at %d, before the cue above it at %d", at, from, last.From)
		case from < last.To:
			return nil, fmt.Errorf("cue %d: begins at %d, inside the cue above it ending at %d", at, from, last.To)
		case strings.ContainsAny(one.GetText(), "\r\n"):
			// One cue is one line of the transcript, and the window edits it as
			// one. A cue broken over two lines is two the window would offer to
			// edit and one the recording would play.
			return nil, fmt.Errorf("cue %d: a cue stands on one line", at)
		}
		last = domain.TranscriptCue{Text: one.GetText(), From: from, To: to}
		if strings.TrimSpace(one.GetText()) == "" {
			continue
		}
		out = append(out, last)
	}
	if len(out) == 0 {
		return nil, errors.New("a transcript of no words is not one this recording was put right to")
	}
	return out, nil
}

// selectSpan cuts the cues down to the run of the words a request named.
//
// The run is a span of the words, which is how a passage is addressed
// everywhere else, and what comes back is the speech those bytes were said in.
// A search hit is played from the first of them.
func selectSpan(at *v1.Span) func([]domain.TranscriptCue) ([]domain.TranscriptCue, error) {
	return func(cues []domain.TranscriptCue) ([]domain.TranscriptCue, error) {
		span := domain.ByteSpan{From: int(at.GetFrom()), To: int(at.GetTo())}
		if span.From < 0 {
			return nil, fmt.Errorf("from: %d is not a place in the words", span.From)
		}
		if span.Empty() {
			return nil, fmt.Errorf("to: %d is not the end of a run of words", span.To)
		}
		return transcript.GetCuesAt(cues, span.From, span.Len()), nil
	}
}
