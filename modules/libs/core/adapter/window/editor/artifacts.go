package editor

import (
	"context"
	"errors"
	"io/fs"
	"slices"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
func carried(kind domain.SourceKind, at domain.WebAddress) []v1.ArtifactKind {
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
	case kind == domain.KindNote && at.IsVideo():
		return []v1.ArtifactKind{
			v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT,
			v1.ArtifactKind_ARTIFACT_KIND_COPY,
		}
	case kind == domain.KindNote && at.URL != "":
		return []v1.ArtifactKind{v1.ArtifactKind_ARTIFACT_KIND_ARTICLE}
	default:
		return nil
	}
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

// points is where the note at a path points, and nothing for every other file.
// What is made from a note follows from that: a note pointing nowhere has
// nothing at an address to fetch.
func (a *API) points(ctx context.Context, v domain.Vault, ref domain.Fingerprint) domain.WebAddress {
	if ref.Kind != domain.KindNote || a.Notes.Read == nil {
		return domain.WebAddress{}
	}
	found, err := a.Notes.Read.Execute(ctx, v, ref.Path)
	if err != nil {
		return domain.WebAddress{}
	}
	return found.Address
}

// linked is the text of what a note points at, as it now stands. What kind of
// text that is follows from the address: a video is a transcript, and every
// other page is an article.
func (a *API) linked(
	ctx context.Context, v domain.Vault, path string, at domain.WebAddress,
) (*v1.Artifact, error) {
	out := &v1.Artifact{
		Name:  named(v, path, articleID),
		Kind:  textOf(at),
		State: v1.State_STATE_NONE,
	}
	_, stores, held := a.hearing()
	if !held || at.URL == "" {
		return out, nil
	}
	store, err := stores.Open(v)
	if err != nil {
		return nil, err
	}
	hash := derived.Fingerprint([]byte(at.URL))
	for _, from := range derived.Producers() {
		got, err := farUnder(ctx, store, from, hash)
		if err != nil {
			return nil, err
		}
		if got.stands != untouched {
			return stood(v, path, textOf(at), got), nil
		}
	}
	return out, nil
}

// drops takes the copy of a video off this disk. The note stands as it did,
// pointing at the address, and the tab frames it again.
func (a *API) drops(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.WebAddress,
) (*connect.Response[v1.DeleteArtifactResponse], error) {
	_, stores, held := a.hearing()
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
	name := derived.Copy(derived.Fingerprint([]byte(at.URL)))
	if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, connect.NewError(reaching(err), err)
	}
	return connect.NewResponse(&v1.DeleteArtifactResponse{
		Artifact: &v1.Artifact{
			Name:  named(v, ref.Path, copiedID),
			Kind:  v1.ArtifactKind_ARTIFACT_KIND_COPY,
			State: v1.State_STATE_NONE,
		},
	}), nil
}

// copyOf is whether a copy of the video at an address stands on this disk, and
// how large it is.
func (a *API) copyOf(
	ctx context.Context, v domain.Vault, path string, at domain.WebAddress,
) *v1.Artifact {
	out := &v1.Artifact{
		Name:  named(v, path, copiedID),
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_COPY,
		State: v1.State_STATE_NONE,
	}
	_, size, held := a.standing(ctx, v, path, at)
	if !held {
		return out
	}
	out.State, out.Size = v1.State_STATE_DONE, size
	return out
}

// copies fetches a copy of the video at an address and answers with what stands
// once it has. A copy over the size the settings name is not fetched, and the
// size it was refused at is said.
func (a *API) copies(
	ctx context.Context, v domain.Vault, ref domain.Fingerprint, at domain.WebAddress,
) (*v1.Artifact, error) {
	if a.Imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoFetcher)
	}
	got, err := a.Imports.Copy(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	out := a.copyOf(ctx, v, ref.Path, at)
	switch {
	case got.Busy:
		out.State = v1.State_STATE_RUNNING
	case got.TooLarge:
		out.State, out.Size = v1.State_STATE_FAILED, got.Bytes
		out.Error = errTooLarge.Error()
	}
	return out, nil
}

// errTooLarge is a copy over the size the settings name. Nothing was fetched,
// and the size it was refused at is on the answer.
var errTooLarge = errors.New("this video is larger than importing.copy_under_mb")

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
	for _, of := range carried(ref.Kind, at) {
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
	// The kind of the file decides what is made from it, so an artifact the file
	// does not carry is a client asking for a run over the wrong thing.
	at := a.points(ctx, showing, ref)
	if !slices.Contains(carried(ref.Kind, at), of) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	var made *v1.Artifact
	switch of {
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED:
		made, err = a.proofreadTranscript(ctx, showing, ref)
	case v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT, v1.ArtifactKind_ARTIFACT_KIND_ARTICLE:
		// A recording's transcript is heard by a model here; a note's is
		// fetched from the address it points at.
		if ref.Kind == domain.KindNote {
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
	// A copy of a video is bytes and no words: taking it away leaves the note
	// as it was, pointing at the address it points at.
	if at := a.points(ctx, showing, ref); at.IsVideo() {
		return a.drops(ctx, showing, ref, at)
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
	return connect.NewResponse(&v1.DeleteArtifactResponse{
		Artifact: &v1.Artifact{
			Name:  named(showing, ref.Path, transcriptID),
			Kind:  v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT,
			State: v1.State_STATE_NONE,
		},
	}), nil
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
		return stood(v, ref.Path, of, got), nil
	}

	// What this machine has fetched is not asked about. A run comes up in its
	// turn and fetches what it needs then.
	//
	// The list of what is being done draws the run from the moment it begins,
	// under the work and the file it is over.
	id, _ := standing(of)
	return &v1.Artifact{
		Name:  named(v, ref.Path, id),
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

// errNoFetcher is a build on a machine holding neither of the tools an address
// is reached with. The settings name where each of them is.
var errNoFetcher = errors.New("this build cannot fetch what an address holds")

// fetch reaches the address a link note points at and answers with what stands
// once it has.
//
// It is waited for: a video's words are one request and a page is one page, and
// both are over in the time a person waits for a window to answer.
func (a *API) fetch(
	ctx context.Context,
	v domain.Vault,
	ref domain.Fingerprint,
	at domain.WebAddress,
) (*v1.Artifact, error) {
	if a.Imports == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoFetcher)
	}
	if _, err := a.Imports.Execute(ctx, v, ref.Path); err != nil {
		return nil, connect.NewError(reaching(err), err)
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
	return &v1.Artifact{Name: named(v, ref.Path, transcriptCorrectedID), State: came(res)}, nil
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
	at domain.WebAddress,
	of v1.ArtifactKind,
) (*v1.Artifact, error) {
	if of == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED {
		return a.corrections(ctx, v, ref)
	}
	// A note's transcript is the text at the address it points at, and a
	// recording's is what a model heard: the same kind, made two ways.
	if ref.Kind == domain.KindNote &&
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
	return stood(v, ref.Path, of, got), nil
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
		Name:  named(v, ref.Path, transcriptCorrectedID),
		Kind:  v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED,
		State: v1.State_STATE_NONE,
	}
	said, store, listened, err := a.made(ctx, v, ref.Path)
	if err != nil || !listened {
		return out, err
	}
	switch put, err := store.Read(ctx, derived.Corrections(said.Producer, said.Hash)); {
	case err == nil:
		out.State, out.Size = v1.State_STATE_DONE, int64(len(put))
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
func stood(v domain.Vault, path string, of v1.ArtifactKind, got reached) *v1.Artifact {
	id, _ := standing(of)
	out := &v1.Artifact{Name: named(v, path, id), Kind: of, Size: int64(got.size)}
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

// named is where an artifact stands: the vault, the file it was made from, and
// the name the store keeps it under.
func named(v domain.Vault, path, id string) string {
	return "vaults/" + string(v.ID) + "/files/" + path + "/artifacts/" + id
}

// reaching is the code a file that could not be reached is answered with.
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
	ctx context.Context, v domain.Vault, path string, at domain.WebAddress,
) (beside string, size int64, held bool) {
	if !at.IsVideo() {
		return "", 0, false
	}
	if reader, err := a.Readers.Open(v); err == nil {
		if ref, err := reader.Stat(ctx, source.CopyBeside(path)); err == nil {
			return ref.Path, ref.Size, true
		}
	}
	_, stores, ready := a.hearing()
	if !ready {
		return "", 0, false
	}
	store, err := stores.Open(v)
	if err != nil {
		return "", 0, false
	}
	file, size, err := store.Open(ctx, derived.Copy(derived.Fingerprint([]byte(at.URL))))
	if err != nil {
		return "", 0, false
	}
	_ = file.Close()
	return "", size, true
}

// textOf is what the text at an address is: a video is words with the times
// they were said at, and every other page is the prose it is written around.
func textOf(at domain.WebAddress) v1.ArtifactKind {
	if at.IsVideo() {
		return v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT
	}
	return v1.ArtifactKind_ARTIFACT_KIND_ARTICLE
}
