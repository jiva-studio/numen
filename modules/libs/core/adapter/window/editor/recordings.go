package editor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/internal/urlfile"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
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
	raw, err := a.transcript(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	transcribed, _ := transcript.Reached(raw)
	_, cues := transcript.Parse(raw)
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
		at := a.points(ctx, showing, ref)
		out.MediaUrl, out.MediaType = a.copied(ctx, showing, ref)
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

// copied is where the copy fetched for a url is played from, and what a
// player is told it is. One nothing has been fetched a copy for plays from
// nowhere, which is what leaves the tab framing the address.
func (a *API) copied(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint,
) (media, kind string) {
	at := a.points(ctx, v, ref)
	beside, size, held := a.standing(ctx, v, ref.Path, at)
	if !held {
		return "", ""
	}
	// The address names how large the copy was when it was given out, as a
	// recording's names the bytes it was: a copy fetched again is another
	// address. A copy kept in the vault is a file of the vault, played the way
	// every file of it is played.
	played := ref.Path
	if beside != "" {
		played = beside
	}
	return a.Playing.Address(v, domain.Fingerprint{Path: played, Size: size}), derived.CopyType
}

// ReadTranscript answers with the words of a file that carries times: each
// stretch of speech against the milliseconds it was spoken in. A file nothing
// has been heard for holds none, which is an answer.
func (a *API) ReadTranscript(
	ctx context.Context,
	r *connect.Request[v1.ReadTranscriptRequest],
) (*connect.Response[v1.ReadTranscriptResponse], error) {
	showing, ref, err := a.carrying(
		ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT)
	if err != nil {
		return nil, err
	}
	raw, err := a.text(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	_, cues := transcript.Parse(raw)
	if at := r.Msg.GetSpan(); at != nil {
		if cues, err = within(at)(cues); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	return connect.NewResponse(&v1.ReadTranscriptResponse{Cues: spoken(cues)}), nil
}

// ReadArticle answers with the prose a page is written around. A file nothing
// has been fetched for holds none, which is an answer.
func (a *API) ReadArticle(
	ctx context.Context,
	r *connect.Request[v1.ReadArticleRequest],
) (*connect.Response[v1.ReadArticleResponse], error) {
	showing, ref, err := a.carrying(
		ctx, r.Msg.GetPath(), v1.ArtifactKind_ARTIFACT_KIND_ARTICLE)
	if err != nil {
		return nil, err
	}
	raw, err := a.text(ctx, showing, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	whole, _ := transcript.Parse(raw)
	prose, err := stretch(whole, r.Msg.GetSpan())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.ReadArticleResponse{Text: prose}), nil
}

// stretch is the run of the prose a request named, and the whole of it where a
// request named none. A run is held within the prose and stands on whole
// characters.
func stretch(prose string, at *v1.Span) (string, error) {
	if at == nil {
		return prose, nil
	}
	span := domain.Span{From: int(at.GetFrom()), To: int(at.GetTo())}
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

// text is the text of the file at a path as it now stands: what a person put
// right, and what a model wrote or a site published where nothing put it right.
func (a *API) text(ctx context.Context, v domain.Vault, path string) ([]byte, error) {
	said, store, made, err := a.made(ctx, v, path)
	if err != nil || !made {
		return nil, err
	}
	return a.transcribed(ctx, store, said)
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
	said, store, listened, err := a.made(ctx, showing, ref.Path)
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
			a.say(task.Task{ID: readingBooks, Doing: "Reading books", About: ref.Path, Error: err.Error()})
		}
	}
	return connect.NewResponse(&v1.WriteTranscriptResponse{Cues: spoken(cues)}), nil
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
// The run is a span of the words, which is how a passage is addressed
// everywhere else, and what comes back is the speech those bytes were said in.
// A search hit is played from the first of them.
func within(at *v1.Span) func([]transcript.Cue) ([]transcript.Cue, error) {
	return func(cues []transcript.Cue) ([]transcript.Cue, error) {
		span := domain.Span{From: int(at.GetFrom()), To: int(at.GetTo())}
		if span.From < 0 {
			return nil, fmt.Errorf("from: %d is not a place in the words", span.From)
		}
		if span.Empty() {
			return nil, fmt.Errorf("to: %d is not the end of a run of words", span.To)
		}
		return transcript.At(cues, span.From, span.Len()), nil
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

// transcribing is what says which model transcribed a recording and where what
// it wrote is kept. They are the index and the store a passage is placed from,
// which read the same artifacts.
func (a *API) transcribing() (port.SourceQueries, port.DerivedStores, bool) {
	if a.Highlight == nil || a.Highlight.Sources == nil || a.Highlight.Derived == nil {
		return nil, nil, false
	}
	return a.Highlight.Sources, a.Highlight.Derived, true
}

// made is what was made from the file at a path, and the store holding it. It
// answers false for a file nothing has been made from.
//
// A document's reading is named by the bytes that were read, which the index
// holds. What a link note points at was never in the note's bytes: it is named
// by the address, so it is looked for under that name and a walk that has not
// reached the note yet takes nothing away from it.
func (a *API) made(
	ctx context.Context,
	v domain.Vault,
	path string,
) (port.SourceText, port.DerivedStore, bool, error) {
	sources, stores, ok := a.transcribing()
	if !ok {
		return port.SourceText{}, nil, false, nil
	}
	if at := a.pointing(ctx, v, path); string(at) != "" {
		return a.fetchedUnder(ctx, v, at)
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

// pointing is the address the note at a path carries, and nothing where the
// file is not a note or points nowhere.
func (a *API) pointing(ctx context.Context, v domain.Vault, path string) domain.URL {
	reader, err := a.Readers.Open(v)
	if err != nil {
		return domain.URL("")
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.URL("")
	}
	at, err := urlfile.Read(raw)
	if err != nil {
		return domain.URL("")
	}
	return at
}

// fetchedUnder is what stands in the store under an address, and which producer
// wrote it. Nothing fetched is an address nothing has been fetched for, which is
// a link note's ordinary state until something is.
func (a *API) fetchedUnder(
	ctx context.Context, v domain.Vault, at domain.URL,
) (port.SourceText, port.DerivedStore, bool, error) {
	_, stores, ok := a.transcribing()
	if !ok {
		return port.SourceText{}, nil, false, nil
	}
	store, err := stores.Open(v)
	if err != nil {
		return port.SourceText{}, nil, false, err
	}
	hash := derived.Fingerprint([]byte(string(at)))
	for _, from := range derived.Producers() {
		got, err := farUnder(ctx, store, from, hash)
		if err != nil {
			return port.SourceText{}, nil, false, err
		}
		if got.stands != untouched {
			return port.SourceText{Producer: from, Hash: hash}, store, true, nil
		}
	}
	return port.SourceText{}, store, false, nil
}

// transcript is what a model wrote down of the recording at a path, and nothing
// where nothing has listened to it. A run still going is read as far as it has
// got.
func (a *API) transcript(ctx context.Context, v domain.Vault, path string) ([]byte, error) {
	said, store, listened, err := a.made(ctx, v, path)
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
