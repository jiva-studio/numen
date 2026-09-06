package editor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// A recording crosses to the window twice: as the bytes a player is pointed at,
// and as the words a model heard in them.

// errNotARecording is what a facet of a recording answers for a file that is
// not one.
var errNotARecording = errors.New("not a recording this vault holds")

// errNotHeard is a recording nothing has listened to, which holds no words to
// edit, to take away or to put right.
var errNotHeard = errors.New("nothing has listened to this recording")

// errBeingHeard is what an edit to a recording a run holds gets. A run appends
// to the transcript, and what is being appended to is not edited underneath.
var errBeingHeard = errors.New("this recording is being listened to")

// GetRecording answers what the recording at a path is: how far the words
// reach, how much of it a run has written down, and where its bytes are played
// from.
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
	raw, err := a.transcript(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	heard, _ := transcript.Reached(raw)
	_, cues := transcript.Parse(raw)
	// Where a recording is played from and what it is played as are answered
	// here: the socket is opened afresh for every run, and what counts as a
	// recording is this application's to say.
	out := &v1.GetRecordingResponse{
		Length: int32(heard),
		Heard:  int32(heard),
		Media:  a.Playing.Address(showing, ref),
		Type:   domain.MediaType(ref.Path),
	}
	if len(cues) > 0 {
		out.Length = max(out.Length, int32(cues[len(cues)-1].To))
	}
	return connect.NewResponse(out), nil
}

// ReadTranscript answers with the transcript of a recording, each stretch of
// speech against the milliseconds it was spoken in. A recording nothing has
// listened to holds no words, which is an answer.
func (a *API) ReadTranscript(
	ctx context.Context,
	r *connect.Request[v1.ReadTranscriptRequest],
) (*connect.Response[v1.ReadTranscriptResponse], error) {
	showing, ref, err := a.recording(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	said, store, listened, err := a.heard(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}

	out := &v1.ReadTranscriptResponse{}
	var raw []byte
	if listened {
		if raw, err = a.transcribed(ctx, store, said); err != nil {
			return nil, connect.NewError(reaching(err), err)
		}
		// Taken after the words, so a run that began while they were being read
		// is one the caller is told about.
		out.Editable = free(ctx, store, derived.Partial(said.Producer, said.Hash))
	}
	_, cues := transcript.Parse(raw)
	if at := r.Msg.GetAt(); at != nil {
		cues, err = within(at)(cues)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	out.Cues = spoken(cues)
	return connect.NewResponse(out), nil
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
	cues, err := ordered(r.Msg.GetCues())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	showing, ref, err := a.recording(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	said, store, listened, err := a.heard(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	if !listened {
		return nil, connect.NewError(connect.CodeNotFound, errNotHeard)
	}

	// A run holds the recording it is listening to for as long as it takes, by
	// the name it appends to.
	release, err := store.Claim(ctx, derived.Partial(said.Producer, said.Hash))
	if errors.Is(err, port.ErrClaimed) {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errBeingHeard)
	}
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	defer release()

	// The words are a person's, and a proofreader leaves them alone.
	written := append(transcript.Marshal(cues), transcript.Hand()...)
	if err := store.Write(ctx, derived.Corrections(said.Producer, said.Hash), written); err != nil {
		return nil, connect.NewError(reaching(err), err)
	}

	// The chunks in the index hold the words as they were heard, so the source
	// is cut again from what it now says. A write that landed is not refused
	// for a cut that could not be asked for.
	if cut := a.cuts(); cut != nil {
		if err := cut(ctx, showing, ref.Path); err != nil {
			a.say(task.Task{ID: readingBooks, Doing: "Reading books", About: ref.Path, Failed: err.Error()})
		}
	}
	return connect.NewResponse(&v1.WriteTranscriptResponse{
		Cues: spoken(cues), Editable: true,
	}), nil
}

// recording is the vault the window is showing and the recording it holds at a
// path. A file that is not one carries no transcript.
func (a *API) recording(
	ctx context.Context,
	path string,
) (domain.Vault, domain.Fingerprint, error) {
	showing, ref, err := a.held(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(reaching(err), err)
	}
	// A link note pointing at a video has words with times in them, as a
	// recording does, and they are read back the same way.
	if ref.Kind == domain.KindNote && a.points(ctx, showing, ref).IsVideo() {
		return showing, ref, nil
	}
	if ref.Kind != domain.KindRecording {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(
			connect.CodeInvalidArgument, errNotARecording)
	}
	return showing, ref, nil
}

// spoken is a transcript as a caller reads it.
func spoken(cues []transcript.Cue) []*v1.Cue {
	out := make([]*v1.Cue, 0, len(cues))
	for _, one := range cues {
		out = append(out, &v1.Cue{Text: one.Text, From: int32(one.From), To: int32(one.To)})
	}
	return out
}

// ordered is a transcript from the window as the cues it is written down as,
// and why it is not one where it cannot be.
//
// Speech runs forward: a cue ends no earlier than it begins, and begins after
// the one before it ends. A cue whose words trim away is dropped, and its
// timings still bound the cue after it.
//
// A transcript carrying no words at all is refused.
func ordered(cues []*v1.Cue) ([]transcript.Cue, error) {
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

// within cuts the cues down to the run of the words a request named.
//
// The run is a start and a length in the words, which is how a passage is
// addressed everywhere else, and what comes back is the speech those bytes were
// said in. A search hit is played from the first of them.
func within(at *v1.Stretch) func([]transcript.Cue) ([]transcript.Cue, error) {
	return func(cues []transcript.Cue) ([]transcript.Cue, error) {
		start, length := int(at.GetStart()), int(at.GetLength())
		if start < 0 {
			return nil, fmt.Errorf("start: %d is not a place in the words", start)
		}
		if length <= 0 {
			return nil, fmt.Errorf("length: %d is not a run of words", length)
		}
		return transcript.At(cues, start, length), nil
	}
}

// held is the vault the window is showing and what it holds at a path. It is
// the one reading of the vault a request gets, and everything the request goes
// on to do is done to that vault.
//
// Everything from outside reaches the vault through a reader, so a path leaving
// it is refused there.
func (a *API) held(ctx context.Context, path string) (domain.Vault, domain.Fingerprint, error) {
	showing := a.Showing()
	if showing.ID == "" || a.Readers == nil {
		return domain.Vault{}, domain.Fingerprint{}, errNoVault
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, err
	}
	return showing, ref, nil
}

// hearing is what says which model listened to a recording and where what it
// wrote is kept. They are the index and the store a passage is placed from,
// which read the same artifacts.
func (a *API) hearing() (port.SourceQueries, port.DerivedStores, bool) {
	if a.Highlight == nil || a.Highlight.Sources == nil || a.Highlight.Derived == nil {
		return nil, nil, false
	}
	return a.Highlight.Sources, a.Highlight.Derived, true
}

// heard is what listened to the recording at a path and the store holding what
// it wrote. It answers false for a recording nothing has listened to.
func (a *API) heard(
	ctx context.Context,
	v domain.Vault,
	path string,
) (port.SourceText, port.DerivedStore, bool, error) {
	sources, stores, ok := a.hearing()
	if !ok {
		return port.SourceText{}, nil, false, nil
	}
	said, held, err := sources.Reading(ctx, v.ID, path)
	if err != nil || !held || said.Producer == "" {
		return port.SourceText{}, nil, false, err
	}
	store, err := stores.Open(v)
	if err != nil {
		return port.SourceText{}, nil, false, err
	}
	return said, store, true, nil
}

// transcript is what a model wrote down of the recording at a path, and nothing
// where nothing has listened to it. A run still going is read as far as it has
// got.
func (a *API) transcript(ctx context.Context, v domain.Vault, path string) ([]byte, error) {
	said, store, listened, err := a.heard(ctx, v, path)
	if err != nil || !listened {
		return nil, err
	}
	return written(ctx, store,
		derived.Artifact(said.Producer, said.Hash),
		derived.Partial(said.Producer, said.Hash),
	)
}

// transcribed is the transcript of a recording as it now stands: what it was put
// right to, and what was heard where nothing put it right.
func (a *API) transcribed(ctx context.Context, store port.DerivedStore, said port.SourceText) ([]byte, error) {
	return written(ctx, store,
		derived.Corrections(said.Producer, said.Hash),
		derived.Artifact(said.Producer, said.Hash),
		derived.Partial(said.Producer, said.Hash),
	)
}

// written is what the store holds under the first of these names, and nothing
// where it holds none of them. The store is a folder on the person's disk and
// they may empty it.
func written(ctx context.Context, store port.DerivedStore, names ...string) ([]byte, error) {
	for _, name := range names {
		raw, err := store.Read(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return nil, nil
}

// free says whether a name is one nobody holds. It is taken and let go, so what
// it answers is what stood a moment ago.
func free(ctx context.Context, store port.DerivedStore, name string) bool {
	release, err := store.Claim(ctx, name)
	if err != nil {
		return false
	}
	release()
	return true
}
