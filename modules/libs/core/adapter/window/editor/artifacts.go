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
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/task"
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
	recognitionID          = derived.OCR
	recognitionCorrectedID = derived.OCR + ".corrected"
	transcriptID           = derived.Transcript
	transcriptCorrectedID  = derived.Transcript + ".corrected"
	articleID              = derived.Article
	copiedID               = derived.Copies
)

// errNoArtifact is a kind no file of the vault carries, and errNotCarried one
// this file's kind does not carry.
var (
	errNoArtifact = errors.New("nothing of that name is made from a file")
	errNotCarried = errors.New("this file carries no artifact of that name")
)

// getArtifactKinds is every artifact a file can carry, in the order they are made. A
// file carrying none is one nothing is made from.
//
// A note carries what is at the address it points at, and what that is follows
// from the address: a video is words with the times they were said at, and
// every other page is the prose it is written around.
func getArtifactKinds(kind domain.SourceKind, produces string) []v1.ArtifactKind {
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

// getArtifactFile is the vault the window is showing and the file at a path, where
// that file carries the artifact asked about. A file carrying none of that kind
// holds nothing to read.
func (a *API) getArtifactFile(
	ctx context.Context,
	path string,
	of v1.ArtifactKind,
) (domain.Vault, domain.Fingerprint, error) {
	showing, ref, err := a.getShownFile(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(getReachCode(err), err)
	}
	at := a.getArtifactAddress(ctx, showing, ref)
	if !slices.Contains(getArtifactKinds(ref.Kind, a.getProducer(at)), of) {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(
			connect.CodeInvalidArgument, errNotCarried)
	}
	return showing, ref, nil
}

// getProducer is what fetching an address would keep its text under, and nothing
// where this build reaches no address at all. What is at an address is the
// fetcher's to say, so nothing here reads the address itself.
func (a *API) getProducer(at domain.URL) string {
	if a.Imports == nil || a.Imports.By == nil || at == "" {
		return ""
	}
	return a.Imports.By.GetDownloadModel(at).Producer
}

// getArtifactID is the name one artifact stands under in the store, and whether the
// schema names that artifact at all.
func getArtifactID(of v1.ArtifactKind) (string, bool) {
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_OCR:
		return recognitionID, true
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT:
		return transcriptID, true
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED:
		return transcriptCorrectedID, true
	case v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED:
		return recognitionCorrectedID, true
	case v1.ArtifactKind_ARTIFACT_KIND_ARTICLE:
		return articleID, true
	case v1.ArtifactKind_ARTIFACT_KIND_COPY:
		return copiedID, true
	default:
		return "", false
	}
}

// getArtifactAddress is the address the file at a path holds, and nothing for every other
// file. What is made from it follows from that.
func (a *API) getArtifactAddress(ctx context.Context, v domain.Vault, ref domain.Fingerprint) domain.URL {
	if ref.Kind != domain.KindURL {
		return domain.URL("")
	}
	return a.Texts.GetNoteAddress.Execute(ctx, v, ref.Path)
}

// getLinkedArtifact is the text of what a note points at, as it now stands. What kind of
// text that is follows from the address: a video is a transcript, and every
// other page is an article.
func (a *API) getLinkedArtifact(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.URL,
) (*v1.Artifact, error) {
	of := getTextKind(a.getProducer(at))
	if at == "" {
		return newArtifact(of, source.RunNone, 0), nil
	}
	progress, err := a.Texts.GetRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, err
	}
	return newRunArtifact(of, progress), nil
}

// removeCopy takes the copy of a video off this disk. The note stands as it did,
// pointing at the address, and the tab frames it again.
func (a *API) removeCopy(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.URL,
) (*connect.Response[v1.DeleteArtifactResponse], error) {
	if err := a.Texts.RemoveCopy.Execute(ctx, v, ref.Path, at); err != nil {
		if errors.Is(err, source.ErrNotReady) {
			return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
		}
		return nil, connect.NewError(getReachCode(err), err)
	}
	return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
}

// getVideoCopy is whether a copy of the video at an address stands on this disk, and
// how large it is.
func (a *API) getVideoCopy(
	ctx context.Context, v domain.Vault, path string, at domain.URL,
) *v1.Artifact {
	out := &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_COPY,
		State: v1.State_STATE_NONE,
	}
	location := a.Texts.GetCopyLocation.Execute(ctx, v, path, at)
	if !location.Exists {
		return out
	}
	out.State, out.Bytes = v1.State_STATE_DONE, location.Size
	return out
}

// fetchCopy fetches a copy of what is at an address and answers with what stands
// once it has. A copy over the size the settings name is not fetched, and the
// size it was refused at is said.
func (a *API) fetchCopy(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.URL,
) (*v1.Artifact, error) {
	if a.Imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// A copy is minutes of fetching, so how far it has got is reported as it
	// arrives.
	asked := *a.Imports
	asked.Progress = func(done, total int64) {
		a.reportTask(task.Task{
			ID: copying + ref.Path, Title: "Downloading a copy", About: ref.Path,
			Count: done, Total: total, Unit: task.Bytes, IsUserStarted: true,
		})
	}
	got, err := asked.Copy(ctx, v, ref.Path)
	a.finishTask(copying + ref.Path)
	out := a.getVideoCopy(ctx, v, ref.Path, at)
	if errors.Is(err, source.ErrBeingDownloaded) {
		out.State = v1.State_STATE_RUNNING
		return out, nil
	}
	if err != nil {
		return nil, connect.NewError(getDownloadCode(err), err)
	}
	if got.IsTooLarge() {
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

// ReportArtifactChange says an artifact of the file at a path was written from
// outside the window, so a tab drawing it reads what now stands. It is the same
// channel a run reports itself through, and a tab follows both the same way.
func (a *API) ReportArtifactChange(path string) {
	a.reportTask(task.Task{ID: fetching + path, Title: "Correcting a transcript", About: path})
	a.finishTask(fetching + path)
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
	showing, ref, err := a.getShownFile(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	at := a.getArtifactAddress(ctx, showing, ref)
	out := &v1.ListArtifactsResponse{}
	for _, of := range getArtifactKinds(ref.Kind, a.getProducer(at)) {
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
	if _, named := getArtifactID(of); !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	showing, ref, err := a.getShownFile(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	// What a url carries is what fetches it, so a build that reaches no address
	// says it cannot rather than that the file carries nothing.
	at := a.getArtifactAddress(ctx, showing, ref)
	if ref.Kind == domain.KindURL && a.getProducer(at) == "" {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// The kind of the file decides what is made from it, so an artifact the file
	// does not carry is a client asking for a run over the wrong thing.
	if !slices.Contains(getArtifactKinds(ref.Kind, a.getProducer(at)), of) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	var made *v1.Artifact
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED:
		made, err = a.proofreadTranscript(ctx, showing, ref)
	case v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED:
		made, err = a.proofreadRecognition(ctx, showing, ref)
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT, v1.ArtifactKind_ARTIFACT_KIND_ARTICLE:
		// A recording's transcript is heard by a model here; a note's is
		// fetched from the address it points at.
		if ref.Kind == domain.KindURL {
			made, err = a.fetchArtifact(ctx, showing, ref, at)
			break
		}
		made, err = a.run(ctx, showing, ref, of)
	case v1.ArtifactKind_ARTIFACT_KIND_COPY:
		made, err = a.fetchCopy(ctx, showing, ref, at)
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
	showing, ref, err := a.getShownFile(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	of := r.Msg.GetKind()
	if _, named := getArtifactID(of); !named {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	// A copy is bytes and no words: taking it away leaves the url as it was,
	// pointing at the address it points at.
	if of == v1.ArtifactKind_ARTIFACT_KIND_COPY {
		return a.removeCopy(ctx, showing, ref, a.getArtifactAddress(ctx, showing, ref))
	}
	if ref.Kind == domain.KindURL {
		if a.Imports == nil {
			return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
		}
		if err := a.Imports.DeleteText(ctx, showing, ref.Path); err != nil {
			return nil, connect.NewError(getReachCode(err), err)
		}
		return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
	}
	if ref.Kind != domain.KindRecording {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	// The queue told about it is the one behind the vault the recording was
	// found in.
	drops := *a.Drops
	drops.Forgets = a.getRecordingForgetter()

	res, err := drops.Execute(ctx, showing, ref.Path)
	switch {
	case err != nil:
		return nil, connect.NewError(getReachCode(err), err)
	case res.IsBusy:
		return nil, connect.NewError(connect.CodeFailedPrecondition, errBeingHeard)
	case res.IsNone:
		return nil, connect.NewError(connect.CodeNotFound, source.ErrNoTranscript)
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
	progress, err := a.Texts.GetRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	// What stands is what the ask comes to. A source a run already answered
	// about is answered the same until that record is taken away.
	if progress.State == source.RunComplete || progress.State == source.RunActive ||
		progress.State == source.RunNoSpeech || progress.State == source.RunUnopened {
		return newRunArtifact(of, progress), nil
	}

	// What this machine has fetched is not asked about. A run comes up in its
	// turn and fetches what it needs then.
	//
	// The list of what is being done draws the run from the moment it begins,
	// under the work and the file it is over.
	return &v1.Artifact{
		Kind:  of,
		State: getStartState(by.Start(v, ref.Path)),
	}, nil
}

// getStartState is what a run just set going is, as the schema carries it.
func getStartState(started source.StartOutcome) v1.State {
	if started == source.Queued {
		return v1.State_STATE_QUEUED
	}
	return v1.State_STATE_RUNNING
}

// runner is what reads a scan or hears a recording. Nothing while the passes
// behind the vault the window is showing are still coming up.
func (a *API) runner(of v1.ArtifactKind) Runner {
	if of == v1.ArtifactKind_ARTIFACT_KIND_OCR {
		return a.getRecogniser()
	}
	return a.getTranscriber()
}

// errComingUp is a run asked for over a vault whose passes are not up yet. The
// vault is in the window and what runs behind it arrives after, so the caller
// asks again.
var errComingUp = errors.New("the vault is still coming up")

// errNoDownloader is a build on a machine holding neither of the tools an
// address is reached with. The settings name where each of them is.
var errNoDownloader = errors.New("this build cannot download what an address holds")

// fetchArtifact reaches the address a link note points at and answers with what
// stands once it has.
//
// It is waited for: a video's words are one request and a page is one page, and
// both are over in the time a person waits for a window to answer.
func (a *API) fetchArtifact(
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
	asked.IsRepeat = true
	a.reportTask(task.Task{ID: fetching + ref.Path, Title: "Downloading an address", About: ref.Path})
	_, err := asked.Execute(ctx, v, ref.Path)
	a.finishTask(fetching + ref.Path)
	if err != nil {
		return nil, connect.NewError(getDownloadCode(err), err)
	}
	return a.getLinkedArtifact(ctx, v, ref, at)
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
	puts := a.getProofreader()
	if puts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	// Which model puts a transcript right is the settings', so an installation
	// naming none has nothing to ask, and the call was answered.
	if !puts.ProofreaderReady() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoProofreading)
	}
	standing, err := a.Texts.GetTranscript.Execute(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	// Words a model heard are what a proofreader is given, so a recording
	// nothing listened to has nothing to put right. A client that listed what
	// the recording carries never asks.
	if !standing.Exists {
		return nil, connect.NewError(connect.CodeFailedPrecondition, source.ErrNoTranscript)
	}

	res, err := puts.Proofread(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	return &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED,
		State: getProofreadState(res),
	}, nil
}

// getProofreadState is what asking for a transcript to be put right came to.
//
// Words already standing are corrections, whether a model or the person wrote
// them: nothing runs over them either way. One run to a recording, by the name
// it writes under: a run listening to this recording holds that name, and so
// does a proofreading of it.
func getProofreadState(res source.ProofreadTranscriptResult) v1.State {
	switch {
	case res.IsBusy:
		return v1.State_STATE_RUNNING
	case res.IsNone:
		return v1.State_STATE_NONE
	case res.IsEdited, res.IsAlready:
		return v1.State_STATE_DONE
	}
	return v1.State_STATE_RUNNING
}

// proofreadRecognition begins putting the recognition of a scanned document right, and
// answers with what the corrections now are.
func (a *API) proofreadRecognition(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	puts := a.getRecognitionProofreader()
	if puts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	if !puts.ProofreaderReady() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoProofreading)
	}
	progress, err := a.Texts.GetRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	if progress.State != source.RunComplete {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNotRead)
	}

	res, err := puts.Proofread(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	return &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED,
		State: getRecognitionProofreadState(res),
	}, nil
}

func getRecognitionProofreadState(res source.ProofreadRecognitionResult) v1.State {
	switch {
	case res.IsBusy, res.IsWaiting:
		return v1.State_STATE_RUNNING
	case res.IsNone:
		return v1.State_STATE_NONE
	case res.Read >= res.Pages && res.Pages > 0:
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
	if of == v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED {
		return a.recognitionCorrections(ctx, v, ref)
	}
	// A url's transcript is the text at the address it holds, and a
	// recording's is what a model heard: the same kind, made two ways.
	if ref.Kind == domain.KindURL &&
		(of == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT ||
			of == v1.ArtifactKind_ARTIFACT_KIND_ARTICLE) {
		return a.getLinkedArtifact(ctx, v, ref, at)
	}
	if of == v1.ArtifactKind_ARTIFACT_KIND_COPY {
		return a.getVideoCopy(ctx, v, ref.Path, at), nil
	}
	progress, err := a.Texts.GetRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, err
	}
	return newRunArtifact(of, progress), nil
}

// recognitionCorrections is what putting a document's recognition right has come to.
func (a *API) recognitionCorrections(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	state, err := a.Texts.GetRecognitionCorrectionState.Execute(ctx, v, ref.Path)
	if err != nil {
		return nil, err
	}
	return newArtifact(v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED, state, 0), nil
}

// corrections is what putting a recording's transcript right has come to.
func (a *API) corrections(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	state, err := a.Texts.GetTranscriptCorrectionState.Execute(ctx, v, ref.Path)
	if err != nil {
		return nil, err
	}
	return newArtifact(v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED, state, 0), nil
}

// newRunArtifact is how far a run got, as the artifact a client reads.
func newRunArtifact(of v1.ArtifactKind, progress source.RunProgress) *v1.Artifact {
	out := newArtifact(of, progress.State, progress.Size)
	if progress.State == source.RunUnopened {
		out.Error = progress.Why
	}
	return out
}

// newArtifact is a state a run stands at and the bytes written so far, as the artifact a client reads.
func newArtifact(of v1.ArtifactKind, state source.RunState, size int) *v1.Artifact {
	out := &v1.Artifact{Kind: of, Bytes: int64(size)}
	switch state {
	case source.RunComplete:
		out.State = v1.State_STATE_DONE
	case source.RunNoSpeech:
		out.State = v1.State_STATE_EMPTY
	case source.RunUnopened:
		out.State = v1.State_STATE_FAILED
	case source.RunActive:
		out.State = v1.State_STATE_RUNNING
	case source.RunStopped:
		out.State = v1.State_STATE_STOPPED
	default:
		out.State = v1.State_STATE_NONE
	}
	return out
}

// getDownloadCode is what a run over an address answers with. What a tool said about
// an address is what the person is owed, and it reaches them only under a code
// that carries its own words.
func getDownloadCode(err error) connect.Code {
	if code := getReachCode(err); code != connect.CodeInternal {
		return code
	}
	return connect.CodeFailedPrecondition
}

// getReachCode is the code a file that could not be reached is answered with.
func getReachCode(err error) connect.Code {
	switch {
	case errors.Is(err, source.ErrOutside):
		return connect.CodeInvalidArgument
	case errors.Is(err, fs.ErrNotExist), source.IsNoNote(err):
		return connect.CodeNotFound
	case errors.Is(err, errNoVault):
		return connect.CodeFailedPrecondition
	default:
		return connect.CodeInternal
	}
}

// getTextKind is what the text at an address is: a video is words with the times
// they were said at, and every other page is the prose it is written around.
func getTextKind(produces string) v1.ArtifactKind {
	if produces == derived.Captions {
		return v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT
	}
	return v1.ArtifactKind_ARTIFACT_KIND_ARTICLE
}
