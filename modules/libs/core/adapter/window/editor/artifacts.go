package editor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// What a model wrote about a file of the vault is asked for as a resource: it
// is listed, made and taken away, and what has become of it is a state.
//
// Which model does the work follows from the file, so the caller names the
// artifact and never the producer.

// The artifacts a file of the vault can carry, by the name each stands under.
// A name is the last part of an artifact's resource name.
const (
	// An artifact is named by what it is. What made it is the store's to say,
	// and a caller asking for one asks for a kind and not for a producer.
	readingID             = derived.Reading
	readingCorrectedID    = derived.Reading + ".corrected"
	transcriptID          = derived.Transcript
	transcriptCorrectedID = derived.Transcript + ".corrected"
	articleID             = derived.Article
	copiedID              = derived.Copies
)

// errNoArtifact is a kind no file of the vault carries, and errNotCarried one
// this file's kind does not carry.
var (
	errNoArtifact = errors.New("nothing of that name is made from a file")
	errNotCarried = errors.New("this file carries no artifact of that name")
)

// carried is every artifact a file can carry, in the order they are made. A
// file carrying none is one nothing is made from.
//
// A note carries what is at the address it points at, and what that is follows
// from the address: a video is words with the times they were said at, and
// every other page is the prose it is written around.
func carried(kind domain.SourceKind, produces string) []v1.ArtifactKind {
	switch {
	case kind == domain.KindBook:
		return []v1.ArtifactKind{
			v1.ArtifactKind_ARTIFACT_KIND_OCR,
			v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED,
		}
	case kind == domain.KindRecording:
		return []v1.ArtifactKind{
			v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT,
			v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED,
		}
	case kind != domain.KindURL || produces == "":
		return nil
	case produces == derived.Captions:
		// A site that publishes words against a clock is one a copy can be
		// taken from: the same tool answers for both.
		return []v1.ArtifactKind{
			v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT,
			v1.ArtifactKind_ARTIFACT_KIND_COPY,
		}
	default:
		return []v1.ArtifactKind{v1.ArtifactKind_ARTIFACT_KIND_ARTICLE}
	}
}

// carrying is the vault the window is showing and the file at a path, where
// that file carries the artifact asked about. A file carrying none of that kind
// holds nothing to read.
func (a *API) carrying(
	ctx context.Context,
	path string,
	of v1.ArtifactKind,
) (domain.Vault, domain.Fingerprint, error) {
	showing, ref, err := a.held(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(reaching(err), err)
	}
	at := a.points(ctx, showing, ref)
	if !slices.Contains(carried(ref.Kind, a.producing(at)), of) {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(
			connect.CodeInvalidArgument, errNotCarried)
	}
	return showing, ref, nil
}

// producing is what fetching an address would keep its text under, and nothing
// where this build reaches no address at all. What is at an address is the
// fetcher's to say, so nothing here reads the address itself.
func (a *API) producing(at domain.URL) string {
	if a.Imports == nil || a.Imports.By == nil || at == "" {
		return ""
	}
	return a.Imports.By.Downloading(at).Producer
}

// standing is the name one artifact stands under in the store, and whether the
// schema names that artifact at all.
func standing(of v1.ArtifactKind) (string, bool) {
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_OCR:
		return readingID, true
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT:
		return transcriptID, true
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED:
		return transcriptCorrectedID, true
	case v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED:
		return readingCorrectedID, true
	case v1.ArtifactKind_ARTIFACT_KIND_ARTICLE:
		return articleID, true
	case v1.ArtifactKind_ARTIFACT_KIND_COPY:
		return copiedID, true
	default:
		return "", false
	}
}

// points is the address the file at a path holds, and nothing for every other
// file. What is made from it follows from that.
func (a *API) points(ctx context.Context, v domain.Vault, ref domain.Fingerprint) domain.URL {
	if ref.Kind != domain.KindURL {
		return domain.URL("")
	}
	return a.pointing(ctx, v, ref.Path)
}

// linked is the text of what a note points at, as it now stands. What kind of
// text that is follows from the address: a video is a transcript, and every
// other page is an article.
func (a *API) linked(
	ctx context.Context, v domain.Vault, path string, at domain.URL,
) (*v1.Artifact, error) {
	out := &v1.Artifact{
		Kind:  textOf(a.producing(at)),
		State: v1.State_STATE_NONE,
	}
	_, stores, held := a.transcribing()
	if !held || string(at) == "" {
		return out, nil
	}
	store, err := stores.Open(v)
	if err != nil {
		return nil, err
	}
	hash := derived.Fingerprint([]byte(string(at)))
	for _, from := range derived.Producers() {
		got, err := farUnder(ctx, store, from, hash)
		if err != nil {
			return nil, err
		}
		if got.stands != untouched {
			return stood(textOf(a.producing(at)), got), nil
		}
	}
	return out, nil
}

// drops takes the copy of a video off this disk. The note stands as it did,
// pointing at the address, and the tab frames it again.
func (a *API) drops(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.URL,
) (*connect.Response[v1.DeleteArtifactResponse], error) {
	_, stores, held := a.transcribing()
	if !held {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	// A copy kept in the vault is the person's own file, and taking it away is
	// taking a file out of their folder.
	if beside, _, stands := a.standing(ctx, v, ref.Path, at); stands && beside != "" &&
		a.Files.Writers != nil {
		writer, err := a.Files.Writers.Open(v)
		if err != nil {
			return nil, connect.NewError(reaching(err), err)
		}
		if err := writer.Remove(ctx, beside); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, connect.NewError(reaching(err), err)
		}
	}
	store, err := stores.Open(v)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	name := derived.Copy(derived.Fingerprint([]byte(string(at))))
	if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, connect.NewError(reaching(err), err)
	}
	return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
}

// copyOf is whether a copy of the video at an address stands on this disk, and
// how large it is.
func (a *API) copyOf(
	ctx context.Context, v domain.Vault, path string, at domain.URL,
) *v1.Artifact {
	out := &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_COPY,
		State: v1.State_STATE_NONE,
	}
	_, size, held := a.standing(ctx, v, path, at)
	if !held {
		return out
	}
	out.State, out.Bytes = v1.State_STATE_DONE, size
	return out
}

// copies fetches a copy of what is at an address and answers with what stands
// once it has. A copy over the size the settings name is not fetched, and the
// size it was refused at is said.
func (a *API) copies(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.URL,
) (*v1.Artifact, error) {
	if a.Imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// A copy is minutes of fetching, so how far it has got is reported as it
	// arrives.
	asked := *a.Imports
	asked.Progress = func(done, total int64) {
		a.say(task.Task{
			ID: copying + ref.Path, Doing: "Downloading a copy", About: ref.Path,
			Count: done, Total: total, Unit: task.Bytes, Asked: true,
		})
	}
	got, err := asked.Copy(ctx, v, ref.Path)
	a.finished(copying + ref.Path)
	out := a.copyOf(ctx, v, ref.Path, at)
	if errors.Is(err, source.ErrBeingDownloaded) {
		out.State = v1.State_STATE_RUNNING
		return out, nil
	}
	if err != nil {
		return nil, connect.NewError(fetched(err), err)
	}
	if got.TooLarge() {
		out.State = v1.State_STATE_FAILED
		out.Error = fmt.Sprintf(
			"This is %d MB, and a copy may be up to %d MB. "+
				"Raise importing.copy_max_size_mb to keep it.",
			got.Bytes>>20, got.Limit>>20)
	}
	return out, nil
}

// fetching opens the name a run over an address is reported under, so a tab
// drawing that url reads what was fetched as soon as the run is done. copying
// is the same for the copy fetched from it.
const (
	fetching = "fetching:"
	copying  = "copying:"
)

// Changed says an artifact of the file at a path was written from outside the
// window, so a tab drawing it reads what now stands. It is the same channel a
// run reports itself through, and a tab follows both the same way.
func (a *API) Changed(path string) {
	a.say(task.Task{ID: fetching + path, Doing: "Correcting a transcript", About: path})
	a.finished(fetching + path)
}

// ListArtifacts is every artifact the file at a path can carry and what has
// become of each.
//
// A file carrying nothing yet is answered with rows all the same: whether a
// book has been read is a question about the book, and a client that reads it
// off an absence cannot tell a book nobody has read from one this build knows
// nothing about.
func (a *API) ListArtifacts(
	ctx context.Context,
	r *connect.Request[v1.ListArtifactsRequest],
) (*connect.Response[v1.ListArtifactsResponse], error) {
	showing, ref, err := a.held(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	at := a.points(ctx, showing, ref)
	out := &v1.ListArtifactsResponse{}
	for _, of := range carried(ref.Kind, a.producing(at)) {
		one, err := a.artifact(ctx, showing, ref, at, of)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Artifacts = append(out.Artifacts, one)
	}
	return connect.NewResponse(out), nil
}

// CreateArtifact asks for one artifact of one file to be made, and answers with
// what became of the ask.
//
// A file already carrying it is left alone and answered with what stands, so
// asking twice is the same as asking once. A run that stopped part way is begun
// afresh.
func (a *API) CreateArtifact(
	ctx context.Context,
	r *connect.Request[v1.CreateArtifactRequest],
) (*connect.Response[v1.CreateArtifactResponse], error) {
	of := r.Msg.GetKind()
	if _, named := standing(of); !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	showing, ref, err := a.held(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	// What a url carries is what fetches it, so a build that reaches no address
	// says it cannot rather than that the file carries nothing.
	at := a.points(ctx, showing, ref)
	if ref.Kind == domain.KindURL && a.producing(at) == "" {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// The kind of the file decides what is made from it, so an artifact the file
	// does not carry is a client asking for a run over the wrong thing.
	if !slices.Contains(carried(ref.Kind, a.producing(at)), of) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	var made *v1.Artifact
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED:
		made, err = a.proofreadTranscript(ctx, showing, ref)
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT, v1.ArtifactKind_ARTIFACT_KIND_ARTICLE:
		// A recording's transcript is heard by a model here; a note's is
		// fetched from the address it points at.
		if ref.Kind == domain.KindURL {
			made, err = a.fetch(ctx, showing, ref, at)
			break
		}
		made, err = a.run(ctx, showing, ref, of)
	case v1.ArtifactKind_ARTIFACT_KIND_COPY:
		made, err = a.copies(ctx, showing, ref, at)
	default:
		made, err = a.run(ctx, showing, ref, of)
	}
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateArtifactResponse{Artifact: made}), nil
}

// DeleteArtifact takes away everything listening to a recording produced: the
// words a model heard, the words a person put right, and the chunks cut from
// them. The recording stands as it did before anything listened to it.
func (a *API) DeleteArtifact(
	ctx context.Context,
	r *connect.Request[v1.DeleteArtifactRequest],
) (*connect.Response[v1.DeleteArtifactResponse], error) {
	showing, ref, err := a.held(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	of := r.Msg.GetKind()
	if _, named := standing(of); !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	// A copy is bytes and no words: taking it away leaves the url as it was,
	// pointing at the address it points at.
	if of == v1.ArtifactKind_ARTIFACT_KIND_COPY {
		return a.drops(ctx, showing, ref, a.points(ctx, showing, ref))
	}
	if ref.Kind == domain.KindURL {
		if a.Imports == nil {
			return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
		}
		if err := a.Imports.DeleteText(ctx, showing, ref.Path); err != nil {
			return nil, connect.NewError(reaching(err), err)
		}
		return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
	}
	if ref.Kind != domain.KindRecording {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	// The queue told about it is the one behind the vault the recording was
	// found in.
	drops := *a.Drops
	drops.Forgets = a.forgets()

	res, err := drops.Execute(ctx, showing, ref.Path)
	switch {
	case err != nil:
		return nil, connect.NewError(reaching(err), err)
	case res.Busy:
		return nil, connect.NewError(connect.CodeFailedPrecondition, errBeingHeard)
	case res.None:
		return nil, connect.NewError(connect.CodeNotFound, errNotHeard)
	}
	return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
}

// run begins reading a scan or hearing a recording, and answers with what the
// artifact now is.
func (a *API) run(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	of v1.ArtifactKind,
) (*v1.Artifact, error) {
	by := a.runner(of)
	if by == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	got, err := a.far(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	// What stands is what the ask comes to. A source a run already answered
	// about is answered the same until that record is taken away.
	if got.stands == done || got.stands == under || got.stands == silent || got.stands == unopened {
		return stood(of, got), nil
	}

	// What this machine has fetched is not asked about. A run comes up in its
	// turn and fetches what it needs then.
	//
	// The list of what is being done draws the run from the moment it begins,
	// under the work and the file it is over.
	return &v1.Artifact{
		Kind:  of,
		State: beginning(by.Start(v, ref.Path)),
	}, nil
}

// beginning is what a run just set going is, as the schema carries it.
func beginning(started port.StartOutcome) v1.State {
	if started == port.Queued {
		return v1.State_STATE_QUEUED
	}
	return v1.State_STATE_RUNNING
}

// runner is what reads a scan or hears a recording. Nothing while the passes
// behind the vault the window is showing are still coming up.
func (a *API) runner(of v1.ArtifactKind) Runner {
	if of == v1.ArtifactKind_ARTIFACT_KIND_OCR {
		return a.recognises()
	}
	return a.transcribes()
}

// errComingUp is a run asked for over a vault whose passes are not up yet. The
// vault is in the window and what runs behind it arrives after, so the caller
// asks again.
var errComingUp = errors.New("the vault is still coming up")

// errNoDownloader is a build on a machine holding neither of the tools an
// address is reached with. The settings name where each of them is.
var errNoDownloader = errors.New("this build cannot download what an address holds")

// fetch reaches the address a link note points at and answers with what stands
// once it has.
//
// It is waited for: a video's words are one request and a page is one page, and
// both are over in the time a person waits for a window to answer.
func (a *API) fetch(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.URL,
) (*v1.Artifact, error) {
	if a.Imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// A person who asks for this asks for the address afresh: what was fetched
	// before goes, and the site is read again.
	asked := *a.Imports
	asked.Again = true
	a.say(task.Task{ID: fetching + ref.Path, Doing: "Fetching an address", About: ref.Path})
	_, err := asked.Execute(ctx, v, ref.Path)
	a.finished(fetching + ref.Path)
	if err != nil {
		return nil, connect.NewError(fetched(err), err)
	}
	return a.linked(ctx, v, ref.Path, at)
}

// proofreadTranscript begins putting the transcript of a recording right, and
// answers with what the corrections now are.
func (a *API) proofreadTranscript(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	// Taken once, so the whole answer is the work of the vault the window was
	// showing when it was asked.
	puts := a.proofreads()
	if puts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	// Which model puts a transcript right is the settings', so an installation
	// naming none has nothing to ask, and the call was answered.
	if !puts.ProofreaderReady() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoProofreading)
	}
	_, _, listened, err := a.made(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	// Words a model heard are what a proofreader is given, so a recording
	// nothing listened to has nothing to put right. A client that listed what
	// the recording carries never asks.
	if !listened {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNotHeard)
	}

	res, err := puts.Proofread(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	return &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED,
		State: came(res),
	}, nil
}

// came is what asking for a transcript to be put right came to.
//
// Words already standing are corrections, whether a model or the person wrote
// them: nothing runs over them either way. One run to a recording, by the name
// it writes under: a run listening to this recording holds that name, and so
// does a proofreading of it.
func came(res source.ProofreadTranscriptResult) v1.State {
	switch {
	case res.Busy:
		return v1.State_STATE_RUNNING
	case res.None:
		return v1.State_STATE_NONE
	case res.Edited, res.Already:
		return v1.State_STATE_DONE
	}
	return v1.State_STATE_RUNNING
}

// artifact is one artifact of one file, as it now stands.
func (a *API) artifact(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.URL,
	of v1.ArtifactKind,
) (*v1.Artifact, error) {
	if of == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED {
		return a.corrections(ctx, v, ref)
	}
	// A url's transcript is the text at the address it holds, and a
	// recording's is what a model heard: the same kind, made two ways.
	if ref.Kind == domain.KindURL &&
		(of == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT ||
			of == v1.ArtifactKind_ARTIFACT_KIND_ARTICLE) {
		return a.linked(ctx, v, ref.Path, at)
	}
	if of == v1.ArtifactKind_ARTIFACT_KIND_COPY {
		return a.copyOf(ctx, v, ref.Path, at), nil
	}
	got, err := a.far(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, err
	}
	return stood(of, got), nil
}

// corrections is what putting a recording's transcript right has come to.
//
// A recording nothing has listened to carries none. What stands is corrections
// whether a model or the person wrote them, and a run holding the recording is
// a run over these words: one run to a recording, by the name it writes under.
func (a *API) corrections(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	out := &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED,
		State: v1.State_STATE_NONE,
	}
	said, store, listened, err := a.made(ctx, v, ref.Path)
	if err != nil || !listened {
		return out, err
	}
	switch _, err := store.Read(ctx, derived.Corrections(said.Producer, said.Hash)); {
	case err == nil:
		out.State = v1.State_STATE_DONE
		return out, nil
	case !errors.Is(err, fs.ErrNotExist):
		return nil, err
	}
	if !free(ctx, store, derived.Partial(said.Producer, said.Hash)) {
		out.State = v1.State_STATE_RUNNING
	}
	return out, nil
}

// stood is how far a run got, as the artifact a client reads.
func stood(of v1.ArtifactKind, got reached) *v1.Artifact {
	out := &v1.Artifact{Kind: of, Bytes: int64(got.size)}
	switch got.stands {
	case done:
		out.State = v1.State_STATE_DONE
	case silent:
		out.State = v1.State_STATE_EMPTY
	case unopened:
		out.State, out.Error = v1.State_STATE_FAILED, got.why
	case under:
		out.State = v1.State_STATE_RUNNING
	case stopped:
		out.State = v1.State_STATE_STOPPED
	default:
		out.State = v1.State_STATE_NONE
	}
	return out
}

// reaching is the code a file that could not be reached is answered with.
// fetched is what a run over an address answers with. What a tool said about an
// address is what the person is owed, and it reaches them only under a code
// that carries its own words.
func fetched(err error) connect.Code {
	if code := reaching(err); code != connect.CodeInternal {
		return code
	}
	return connect.CodeFailedPrecondition
}

func reaching(err error) connect.Code {
	switch {
	case errors.Is(err, port.ErrOutside):
		return connect.CodeInvalidArgument
	case errors.Is(err, fs.ErrNotExist), port.NoNote(err):
		return connect.CodeNotFound
	case errors.Is(err, errNoVault):
		return connect.CodeFailedPrecondition
	default:
		return connect.CodeInternal
	}
}

// standing is where the copy of a video is: beside the note as a file of the
// vault, or in the application's own folder. Nothing where no copy stands.
//
// A copy lands wherever `importing.copies_to_vault` said when it was fetched,
// and a setting turned afterwards does not move what is already here. Both
// places are looked in, and the vault's own file is the one a person can see.
func (a *API) standing(
	ctx context.Context, v domain.Vault, path string, at domain.URL,
) (beside string, size int64, held bool) {
	if at == "" {
		return "", 0, false
	}
	if reader, err := a.Readers.Open(v); err == nil {
		if ref, err := reader.Stat(ctx, source.CopyBeside(path)); err == nil {
			return ref.Path, ref.Size, true
		}
	}
	_, stores, ready := a.transcribing()
	if !ready {
		return "", 0, false
	}
	store, err := stores.Open(v)
	if err != nil {
		return "", 0, false
	}
	file, size, err := store.Open(ctx, derived.Copy(derived.Fingerprint([]byte(string(at)))))
	if err != nil {
		return "", 0, false
	}
	_ = file.Close()
	return "", size, true
}

// textOf is what the text at an address is: a video is words with the times
// they were said at, and every other page is the prose it is written around.
func textOf(produces string) v1.ArtifactKind {
	if produces == derived.Captions {
		return v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT
	}
	return v1.ArtifactKind_ARTIFACT_KIND_ARTICLE
}
