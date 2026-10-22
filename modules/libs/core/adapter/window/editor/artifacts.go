package editor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// artifactHandler lists, makes and takes away what a model wrote about a file
// of the vault.
type artifactHandler struct {
	state *State
	files *artifactFiles

	imports                       *source.ImportURL
	drops                         *transcription.Drop
	getRunProgress                source.GetRunProgress
	getTranscript                 source.GetTranscript
	getCopyLocation               source.GetCopyLocation
	deleteCopy                    source.RemoveCopy
	getTranscriptCorrectionState  source.GetTranscriptCorrectionState
	getRecognitionCorrectionState source.GetRecognitionCorrectionState
}

// What a model wrote about a file of the vault is asked for as a resource: it
// is listed, made and taken away, and what has become of it is a state.
//
// Which model does the work follows from the file, so the caller names the
// artifact and never the producer.

// errNoArtifact is a kind no file of the vault carries, and errNotCarried one
// this file's kind does not carry.
var (
	errNoArtifact = errors.New("artifact not found for file")
	errNotCarried = errors.New("this file carries no artifact of that name")
)

// getArtifactKinds is every artifact a file can carry, in the order they are made, as the schema names them.
func getArtifactKinds(kind domain.SourceKind, produces string) []v1.ArtifactKind {
	var out []v1.ArtifactKind
	for _, one := range domain.GetArtifactKinds(kind, getAddressText(produces)) {
		out = append(out, getWireArtifactKind(one))
	}
	return out
}

// getAddressText is what the producer that fetched an address found there.
func getAddressText(produces string) domain.AddressText {
	switch produces {
	case "":
		return domain.AddressTextNone
	case domain.Captions:
		return domain.AddressTextCaptions
	default:
		return domain.AddressTextArticle
	}
}

// getWireArtifactKind is the name the schema gives an artifact.
func getWireArtifactKind(of domain.ArtifactKind) v1.ArtifactKind {
	switch of {
	case domain.ArtifactKindRecognition:
		return v1.ArtifactKind_ARTIFACT_KIND_OCR
	case domain.ArtifactKindRecognitionCorrected:
		return v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED
	case domain.ArtifactKindTranscript:
		return v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT
	case domain.ArtifactKindTranscriptCorrected:
		return v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED
	case domain.ArtifactKindArticle:
		return v1.ArtifactKind_ARTIFACT_KIND_ARTICLE
	case domain.ArtifactKindCopy:
		return v1.ArtifactKind_ARTIFACT_KIND_COPY
	default:
		return v1.ArtifactKind_ARTIFACT_KIND_UNSPECIFIED
	}
}

// that file carries the artifact asked about. A file carrying none of that kind
// holds nothing to read.
func (f *artifactFiles) getArtifactFile(
	ctx context.Context,
	path string,
	of v1.ArtifactKind,
) (domain.Vault, domain.Fingerprint, error) {
	showing, ref, err := f.getShownFile(ctx, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(getReachCode(err), err)
	}
	pos := f.getArtifactAddress(ctx, showing, ref)
	producer, err := f.getProducer(ctx, pos)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(connect.CodeUnavailable, err)
	}
	if !slices.Contains(getArtifactKinds(ref.Kind, producer), of) {
		return domain.Vault{}, domain.Fingerprint{}, connect.NewError(
			connect.CodeInvalidArgument, errNotCarried)
	}
	return showing, ref, nil
}

// getProducer is what fetching an address would keep its text under, and nothing
// where this build reaches no address at all. What is at an address is the
// fetcher's to say, so nothing here reads the address itself. A fetcher that
// does not answer is an error.
func (f *artifactFiles) getProducer(ctx context.Context, pos domain.URL) (string, error) {
	if f.imports == nil || f.imports.By == nil || pos == "" {
		return "", nil
	}
	delta, err := f.imports.By.GetDownloadModel(ctx, pos)
	if err != nil {
		return "", err
	}
	return delta.Producer, nil
}

// hasArtifact is whether the schema names that artifact at all.
func hasArtifact(of v1.ArtifactKind) bool {
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_OCR,
		v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT,
		v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED,
		v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED,
		v1.ArtifactKind_ARTIFACT_KIND_ARTICLE,
		v1.ArtifactKind_ARTIFACT_KIND_COPY:
		return true
	default:
		return false
	}
}

// getArtifactAddress is the address the file at a path holds, and nothing for every other
// file. What is made from it follows from that.
func (f *artifactFiles) getArtifactAddress(ctx context.Context, v domain.Vault, ref domain.Fingerprint) domain.URL {
	if ref.Kind != domain.KindURL {
		return domain.URL("")
	}
	return f.getNoteAddress.Execute(ctx, v, ref.Path)
}

// getLinkedArtifact is the text of what a note points at, as it now stands. What kind of
// text that is follows from the address: a video is a transcript, and every
// other page is an article.
func (h *artifactHandler) getLinkedArtifact(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, pos domain.URL,
) (*v1.Artifact, error) {
	producer, err := h.files.getProducer(ctx, pos)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	of := getTextKind(producer)
	if pos == "" {
		return newArtifact(of, source.RunNone, 0), nil
	}
	progress, err := h.getRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, err
	}
	return newRunArtifact(of, progress), nil
}

// removeCopy takes the copy of a video off this disk. The note stands as it did,
// pointing at the address, and the tab frames it again.
func (h *artifactHandler) removeCopy(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, pos domain.URL,
) (*connect.Response[v1.DeleteArtifactResponse], error) {
	if err := h.deleteCopy.Execute(ctx, v, ref.Path, pos); err != nil {
		if errors.Is(err, source.ErrNotReady) {
			return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
		}
		return nil, connect.NewError(getReachCode(err), err)
	}
	return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
}

// getVideoCopy is whether a copy of the video at an address relies on this disk, and
// how large it is.
func (h *artifactHandler) getVideoCopy(
	ctx context.Context, v domain.Vault, path string, pos domain.URL,
) *v1.Artifact {
	out := &v1.Artifact{
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_COPY,
		State: v1.State_STATE_NONE,
	}
	location := h.getCopyLocation.Execute(ctx, v, path, pos)
	if !location.Exists {
		return out
	}
	out.State, out.Bytes = v1.State_STATE_DONE, location.Size
	return out
}

// fetchCopy fetches a copy of what is at an address and answers with what stands
// once it has. A copy over the size the settings name is not fetched, and the
func (h *artifactHandler) fetchCopy(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, pos domain.URL,
) (*v1.Artifact, error) {
	if h.imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// A copy is minutes of fetching, so how far it has got is reported as it
	// arrives.
	report := func(finished, total int64) {
		h.state.reportTask(run.Task{
			ID: copying + ref.Path, Title: "Downloading a copy", About: ref.Path,
			Count: finished, Total: total, Unit: run.Bytes, IsUserStarted: true,
		})
	}
	got, err := h.imports.Copy(ctx, v, ref.Path, report)
	h.state.finishTask(copying + ref.Path)
	out := h.getVideoCopy(ctx, v, ref.Path, pos)
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
func (h *artifactHandler) ReportArtifactChange(path string) {
	h.state.reportTask(run.Task{ID: fetching + path, Title: "Correcting a transcript", About: path})
	h.state.finishTask(fetching + path)
}

// ListArtifacts is every artifact the file at a path can carry and what has
// become of each.
//
// A file carrying nothing yet is answered with rows all the same: whether a
// book has been read is a question about the book, and a client that reads it
// off an absence cannot tell a book nobody has read from one this build knows
// nothing about.
func (h *artifactHandler) ListArtifacts(
	ctx context.Context,
	r *connect.Request[v1.ListArtifactsRequest],
) (*connect.Response[v1.ListArtifactsResponse], error) {
	showing, ref, err := h.files.getShownFile(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	pos := h.files.getArtifactAddress(ctx, showing, ref)
	producer, err := h.files.getProducer(ctx, pos)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	out := &v1.ListArtifactsResponse{}
	for _, of := range getArtifactKinds(ref.Kind, producer) {
		one, err := h.artifact(ctx, showing, ref, pos, of)
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
func (h *artifactHandler) CreateArtifact(
	ctx context.Context,
	r *connect.Request[v1.CreateArtifactRequest],
) (*connect.Response[v1.CreateArtifactResponse], error) {
	of := r.Msg.GetKind()
	if !hasArtifact(of) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	showing, ref, err := h.files.getShownFile(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	// What a url carries is what fetches it, so a build that reaches no address
	// says it cannot and not that the file carries nothing.
	pos := h.files.getArtifactAddress(ctx, showing, ref)
	producer, err := h.files.getProducer(ctx, pos)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if ref.Kind == domain.KindURL && producer == "" {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// The kind of the file decides what is made from it, so an artifact the file
	// does not carry is a client asking for a run over the wrong thing.
	if !slices.Contains(getArtifactKinds(ref.Kind, producer), of) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	var made *v1.Artifact
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED:
		made, err = h.proofreadTranscript(ctx, showing, ref)
	case v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED:
		made, err = h.proofreadRecognition(ctx, showing, ref)
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT, v1.ArtifactKind_ARTIFACT_KIND_ARTICLE:
		// A recording's transcript is heard by a model here; a note's is
		// fetched from the address it points at.
		if ref.Kind == domain.KindURL {
			made, err = h.fetchArtifact(ctx, showing, ref, pos)
			break
		}
		made, err = h.run(ctx, showing, ref, of)
	case v1.ArtifactKind_ARTIFACT_KIND_COPY:
		made, err = h.fetchCopy(ctx, showing, ref, pos)
	default:
		made, err = h.run(ctx, showing, ref, of)
	}
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CreateArtifactResponse{Artifact: made}), nil
}

// DeleteArtifact takes away everything listening to a recording produced: the
// words a model heard, the words a person put right, and the chunks cut from
// them. The recording stands as it did before anything listened to it.
func (h *artifactHandler) DeleteArtifact(
	ctx context.Context,
	r *connect.Request[v1.DeleteArtifactRequest],
) (*connect.Response[v1.DeleteArtifactResponse], error) {
	showing, ref, err := h.files.getShownFile(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	of := r.Msg.GetKind()
	if !hasArtifact(of) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	// A copy is bytes and no words: taking it away leaves the url as it was,
	// pointing at the address it points at.
	if of == v1.ArtifactKind_ARTIFACT_KIND_COPY {
		return h.removeCopy(ctx, showing, ref, h.files.getArtifactAddress(ctx, showing, ref))
	}
	if ref.Kind == domain.KindURL {
		if h.imports == nil {
			return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
		}
		if err := h.imports.DeleteText(ctx, showing, ref.Path); err != nil {
			return nil, connect.NewError(getReachCode(err), err)
		}
		return connect.NewResponse(&v1.DeleteArtifactResponse{}), nil
	}
	if ref.Kind != domain.KindRecording {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	// The queue told about it is the one behind the vault the recording was
	// found in.
	res, err := h.drops.Execute(ctx, showing, ref.Path, h.state.getRecordingForgetter())
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
func (h *artifactHandler) run(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	of v1.ArtifactKind,
) (*v1.Artifact, error) {
	delta := h.runner(of)
	if delta == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	progress, err := h.getRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, connect.NewError(getReachCode(err), err)
	}
	// What stands is what the ask evaluates to. A source a run already answered
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
		State: getStartState(delta.Start(v, ref.Path)),
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
func (h *artifactHandler) runner(of v1.ArtifactKind) Runner {
	if of == v1.ArtifactKind_ARTIFACT_KIND_OCR {
		return h.state.getRecogniser()
	}
	return h.state.getTranscriber()
}

// errComingUp is a run asked for over a vault whose passes are not up yet. The
// vault is in the window and what runs behind it arrives after, so the caller
// asks again.
var errComingUp = errors.New("the vault is still coming up")

// errNoDownloader is a build on a machine holding neither of the tools an
// address is reached with. The settings name where each of them is.
var errNoDownloader = errors.New("this build cannot download what an address holds")

// fetchArtifact reaches the address a link note points at and answers with what
// appears once it has.
//
// It is waited for: a video's words are one request and a page is one page, and
// both are over in the time a person waits for a window to answer.
func (h *artifactHandler) fetchArtifact(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	pos domain.URL,
) (*v1.Artifact, error) {
	if h.imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDownloader)
	}
	// A person who asks for this asks for the address afresh: what was fetched
	// before goes, and the site is read again.
	h.state.reportTask(run.Task{ID: fetching + ref.Path, Title: "Downloading an address", About: ref.Path})
	_, err := h.imports.Execute(ctx, v, ref.Path, true)
	h.state.finishTask(fetching + ref.Path)
	if err != nil {
		return nil, connect.NewError(getDownloadCode(err), err)
	}
	return h.getLinkedArtifact(ctx, v, ref, pos)
}

// proofreadTranscript begins putting the transcript of a recording right, and
// answers with what the corrections now are.
func (h *artifactHandler) proofreadTranscript(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	// Taken once, so the whole answer is the work of the vault the window was
	// showing when it was asked.
	puts := h.state.getProofreader()
	if puts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	// Which model puts a transcript right is the settings', so an installation
	// naming none has nothing to ask, and the call was answered.
	if !puts.ProofreaderReady() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoProofreading)
	}
	standing, err := h.getTranscript.Execute(ctx, v, ref.Path)
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
func getProofreadState(res transcription.ProofreadResult) v1.State {
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
func (h *artifactHandler) proofreadRecognition(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	puts := h.state.getRecognitionProofreader()
	if puts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	if !puts.ProofreaderReady() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoProofreading)
	}
	progress, err := h.getRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
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

func getRecognitionProofreadState(res recognition.ProofreadResult) v1.State {
	switch {
	case res.IsBusy:
		return v1.State_STATE_RUNNING
	case res.IsNone:
		return v1.State_STATE_NONE
	case res.Read >= res.Pages && res.Pages > 0:
		return v1.State_STATE_DONE
	}
	return v1.State_STATE_RUNNING
}

// artifact is one artifact of one file, as it now stands.
func (h *artifactHandler) artifact(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	pos domain.URL,
	of v1.ArtifactKind,
) (*v1.Artifact, error) {
	if of == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED {
		return h.corrections(ctx, v, ref)
	}
	if of == v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED {
		return h.recognitionCorrections(ctx, v, ref)
	}
	// A url's transcript is the text at the address it holds, and a
	// recording's is what a model heard: the same kind, made two ways.
	if ref.Kind == domain.KindURL &&
		(of == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT ||
			of == v1.ArtifactKind_ARTIFACT_KIND_ARTICLE) {
		return h.getLinkedArtifact(ctx, v, ref, pos)
	}
	if of == v1.ArtifactKind_ARTIFACT_KIND_COPY {
		return h.getVideoCopy(ctx, v, ref.Path, pos), nil
	}
	progress, err := h.getRunProgress.Execute(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, err
	}
	return newRunArtifact(of, progress), nil
}

// recognitionCorrections is what putting a document's recognition right has come to.
func (h *artifactHandler) recognitionCorrections(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	state, err := h.getRecognitionCorrectionState.Execute(ctx, v, ref.Path)
	if err != nil {
		return nil, err
	}
	return newArtifact(v1.ArtifactKind_ARTIFACT_KIND_OCR_CORRECTED, state, 0), nil
}

// corrections is what putting a recording's transcript right has come to.
func (h *artifactHandler) corrections(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
) (*v1.Artifact, error) {
	state, err := h.getTranscriptCorrectionState.Execute(ctx, v, ref.Path)
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

// getReachCode is the code a file that could not be reached is answered with. A
// plugin that is not there to answer is a build that cannot do it.
func getReachCode(err error) connect.Code {
	switch {
	case errors.Is(err, source.ErrOutside):
		return connect.CodeInvalidArgument
	case errors.Is(err, source.ErrUnavailable), errors.Is(err, errNoDocuments):
		return connect.CodeUnimplemented
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
	if produces == domain.Captions {
		return v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT
	}
	return v1.ArtifactKind_ARTIFACT_KIND_ARTICLE
}

// path. A file that is not one carries no transcript.
func (f *artifactFiles) recording(
	ctx context.Context,
	path string,
) (domain.Vault, domain.Fingerprint, error) {
	showing, ref, err := f.getShownFile(ctx, path)
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

// the one read of the vault a request gets, and everything the request goes
// on to do is done to that vault.
//
// Everything from outside reaches the vault through a reader, so a path leaving
func (f *artifactFiles) getShownFile(ctx context.Context, path string) (domain.Vault, domain.Fingerprint, error) {
	showing := f.state.GetCurrentVault()
	if showing.ID == "" || f.getFingerprint == nil {
		return domain.Vault{}, domain.Fingerprint{}, errNoVault
	}
	ref, err := f.getFingerprint.Execute(ctx, showing, path)
	if err != nil {
		return domain.Vault{}, domain.Fingerprint{}, err
	}
	return showing, ref, nil
}
