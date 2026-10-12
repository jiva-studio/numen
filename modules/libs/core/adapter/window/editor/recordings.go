package editor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// A recording crosses to the window twice: as the bytes a player is pointed at,
// and as the words a model heard in them.

// errNotARecording is what a facet of a recording answers for a file that is
// not one.
var errNotARecording = errors.New("not a recording this vault holds")

// errBeingHeard is what an edit to a recording a run holds gets.
var errBeingHeard = errors.New("this recording is being listened to")

// GetRecording answers what the file at a path is: how long it runs and where
// its bytes are played from. What has been made from it is ListArtifacts.
//
// A recording nothing has listened to reaches nowhere, and the player it is
// loaded into is what then says how long it runs.
func (a *API) GetRecording(
	ctx context.Context,
	r *connect.Request[v1.GetRecordingRequest],
) (*connect.Response[v1.GetRecordingResponse], error) {
	showing, ref, err := a.recording(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	heard, err := a.Texts.GetHeardTranscript.Execute(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	transcribed, _ := transcript.ReadReached(heard.Words)
	_, cues := transcript.Parse(heard.Words)
	// Where a recording is played from and what it is played as are answered
	// here: the socket is opened afresh for every run, and what counts as a
	// recording is this application's to say.
	out := &v1.GetRecordingResponse{
		DurationMs: int32(transcribed),
		MediaUrl:   a.Playing.Address(showing, ref),
		MediaType:  domain.MediaType(ref.Path),
	}
	// A url plays the copy fetched for it, where one stands. One with none is
	// framed at the address instead, from the socket this run opened, and the
	// frame is a page whatever is inside it.
	if ref.Kind == domain.KindURL {
		at := a.getArtifactAddress(ctx, showing, ref)
		out.MediaUrl, out.MediaType = a.getCopyMedia(ctx, showing, ref)
		out.Url = string(at)
		if out.MediaUrl == "" {
			out.MediaUrl, out.MediaType = a.Playing.Embed(at), asAPage
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
func (a *API) getCopyMedia(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint,
) (media, kind string) {
	at := a.getArtifactAddress(ctx, v, ref)
	location := a.Texts.GetCopyLocation.Execute(ctx, v, ref.Path, at)
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
	return a.Playing.Address(v, domain.Fingerprint{Path: played, Size: location.Size}), derived.CopyType
}

// ReadTranscript answers with the words of a file that carries times: each
// stretch of speech against the milliseconds it was spoken in. A file nothing
// has been heard for holds none, which is an answer.
func (a *API) ReadTranscript(
	ctx context.Context,
	r *connect.Request[v1.ReadTranscriptRequest],
) (*connect.Response[v1.ReadTranscriptResponse], error) {
	showing, ref, err := a.getArtifactFile(
		ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT)
	if err != nil {
		return nil, err
	}
	standing, err := a.Texts.GetTranscript.Execute(ctx, showing, ref.Path)
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

// ReadArticle answers with the prose a page is written around. A file nothing
// has been fetched for holds none, which is an answer.
func (a *API) ReadArticle(
	ctx context.Context,
	r *connect.Request[v1.ReadArticleRequest],
) (*connect.Response[v1.ReadArticleResponse], error) {
	showing, ref, err := a.getArtifactFile(
		ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_ARTICLE)
	if err != nil {
		return nil, err
	}
	standing, err := a.Texts.GetTranscript.Execute(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	whole, _ := transcript.Parse(standing.Words)
	prose, err := readSpan(whole, r.Msg.GetSpan())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.ReadArticleResponse{Text: prose}), nil
}

// readSpan is the run of the prose a request named, and the whole of it where a
// request named none. A run is held within the prose and stands on whole
// characters.
func readSpan(prose string, at *v1.Span) (string, error) {
	if at == nil {
		return prose, nil
	}
	span := domain.ByteSpan{From: int(at.GetFrom()), To: int(at.GetTo())}
	if span.From < 0 {
		return "", fmt.Errorf("from: %d is not a place in the prose", span.From)
	}
	if span.Empty() {
		return "", fmt.Errorf("to: %d is not the end of a run of the prose", span.To)
	}
	start, end := min(span.From, len(prose)), min(span.To, len(prose))
	for start > 0 && !utf8.RuneStart(prose[start]) {
		start--
	}
	for end < len(prose) && !utf8.RuneStart(prose[end]) {
		end++
	}
	return prose[start:end], nil
}

// WriteTranscript writes the transcript of a recording as a person left it.
//
// What the model heard stays under its own name and the words as they now stand
// go beside it, so a transcript edited into nonsense is corrections that can be
// taken away and what was heard comes back.
func (a *API) WriteTranscript(
	ctx context.Context,
	r *connect.Request[v1.WriteTranscriptRequest],
) (*connect.Response[v1.WriteTranscriptResponse], error) {
	cues, err := parseCues(r.Msg.GetCues())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	showing, ref, err := a.recording(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	written, err := a.Texts.WriteTranscript.Execute(ctx, showing, ref.Path, cues, a.getRechunker())
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
		a.reportTask(task.Task{ID: extractingBooks, Title: "Extracting books", About: ref.Path, Error: written.CutError.Error()})
	}
	return connect.NewResponse(&v1.WriteTranscriptResponse{Cues: newWireCues(cues)}), nil
}

// recording is the vault the window is showing and the recording it holds at a
// path. A file that is not one carries no transcript.
func (a *API) recording(
	ctx context.Context,
	path string,
) (domain.Vault, domain.Fingerprint, error) {
	showing, ref, err := a.getShownFile(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(getReachCode(err), err)
	}
	// A url pointing at a video has words with times in them, as a
	// recording does, and they are read back the same way.
	if ref.Kind == domain.KindURL {
		return showing, ref, nil
	}
	if ref.Kind != domain.KindRecording {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(
			connect.CodeInvalidArgument, errNotARecording)
	}
	return showing, ref, nil
}

// newWireCues is a transcript as a caller reads it.
func newWireCues(cues []transcript.Cue) []*v1.Cue {
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
func parseCues(cues []*v1.Cue) ([]transcript.Cue, error) {
	out := make([]transcript.Cue, 0, len(cues))
	last := transcript.Cue{From: -1, To: -1}
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
		last = transcript.Cue{Text: one.GetText(), From: from, To: to}
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
func selectSpan(at *v1.Span) func([]transcript.Cue) ([]transcript.Cue, error) {
	return func(cues []transcript.Cue) ([]transcript.Cue, error) {
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

// getShownFile is the vault the window is showing and what it holds at a path. It is
// the one read of the vault a request gets, and everything the request goes
// on to do is done to that vault.
//
// Everything from outside reaches the vault through a reader, so a path leaving
// it is refused there.
func (a *API) getShownFile(ctx context.Context, path string) (domain.Vault, domain.Fingerprint, error) {
	showing := a.GetCurrentVault()
	if showing.ID == "" || a.Readers == nil {
		return domain.Vault{}, domain.Fingerprint{}, errNoVault
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, err
	}
	ref, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, err
	}
	return showing, ref, nil
}
