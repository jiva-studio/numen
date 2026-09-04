package webui

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
// Which model does the work follows from the file. A scan is read and a
// recording is heard, so the caller names the artifact and never the producer.

// The artifacts a file of the vault can carry, by the id each is asked for
// under. An id is the last part of an artifact's name.
const (
	// readingID is the text a model read out of a scan.
	readingID = "ocr"
	// heardID is the words a model heard in a recording, and correctedID those
	// words put right.
	heardID     = derived.ASR
	correctedID = derived.ASR + ".corrected"
)

// errNoArtifact is an id no file of the vault carries, and errNotCarried one
// this file's kind does not carry.
var (
	errNoArtifact = errors.New("nothing of that name is made from a file")
	errNotCarried = errors.New("this file carries no artifact of that name")
)

// errNothingHeard is corrections asked for over a recording nothing has
// listened to, which holds no words to put right.
var errNothingHeard = errors.New("nothing has listened to this recording")

// errNotDroppable is an artifact taken away that is not taken away here.
var errNotDroppable = errors.New("only what a recording was heard as is taken away")

// carried is every artifact a file of this kind can carry, in the order they
// are made. A kind carrying none is a file nothing is made from.
func carried(kind domain.SourceKind) []string {
	switch kind {
	case domain.KindBook:
		return []string{readingID}
	case domain.KindRecording:
		return []string{heardID, correctedID}
	default:
		return nil
	}
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
	out := &v1.ListArtifactsResponse{}
	for _, id := range carried(ref.Kind) {
		one, err := a.artifact(ctx, showing, ref, id)
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
	id := r.Msg.GetArtifactId()
	if !known(id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	showing, ref, err := a.held(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	// The kind decides what is made from a file, so an id the file does not
	// carry is a client asking for a run over the wrong thing.
	if !slices.Contains(carried(ref.Kind), id) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotCarried)
	}

	var made *v1.Artifact
	if id == correctedID {
		made, err = a.putRight(ctx, showing, ref)
	} else {
		made, err = a.run(ctx, showing, ref, id)
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
	if !known(r.Msg.GetArtifactId()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNoArtifact)
	}
	if r.Msg.GetArtifactId() != heardID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNotDroppable)
	}
	showing, ref, err := a.held(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
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
			Name:  named(showing, ref.Path, heardID),
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
	id string,
) (*v1.Artifact, error) {
	by := a.runner(id)
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
		return stood(v, ref.Path, id, got), nil
	}

	// What this machine has fetched is not asked about. A run comes up in its
	// turn and fetches what it needs then.
	state := v1.State_STATE_RUNNING
	if by.Start(v, ref.Path) == port.Queued {
		state = v1.State_STATE_QUEUED
	}
	// The list of what is being done draws the run from the moment it begins,
	// under the work and the file it is over.
	return &v1.Artifact{Name: named(v, ref.Path, id), State: state}, nil
}

// runner is what reads a scan or hears a recording. Nothing while the passes
// behind the vault the window is showing are still coming up.
func (a *API) runner(id string) Runner {
	if id == readingID {
		return a.recognises()
	}
	return a.transcribes()
}

// errComingUp is a run asked for over a vault whose passes are not up yet. The
// vault is in the window and what runs behind it arrives after, so the caller
// asks again.
var errComingUp = errors.New("the vault is still coming up")

// putRight begins putting the transcript of a recording right, and answers with
// what the corrections now are.
func (a *API) putRight(
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
	_, _, listened, err := a.heard(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	// Words a model heard are what a proofreader is given, so a recording
	// nothing listened to has nothing to put right. A client that listed what
	// the recording carries never asks.
	if !listened {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNothingHeard)
	}

	res, err := puts.Proofread(ctx, v, ref.Path)
	if err != nil {
		return nil, connect.NewError(reaching(err), err)
	}
	return &v1.Artifact{Name: named(v, ref.Path, correctedID), State: came(res)}, nil
}

// came is what asking for a transcript to be put right came to.
//
// Words already standing are corrections, whether a model or the person wrote
// them: nothing runs over them either way. One run to a recording, by the name
// it writes under: a run listening to this recording holds that name, and so
// does a proofreading of it.
func came(res source.PutRightResult) v1.State {
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
	id string,
) (*v1.Artifact, error) {
	if id == correctedID {
		return a.corrections(ctx, v, ref)
	}
	got, err := a.far(ctx, v, ref.Path, ref.Kind)
	if err != nil {
		return nil, err
	}
	return stood(v, ref.Path, id, got), nil
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
	out := &v1.Artifact{Name: named(v, ref.Path, correctedID), State: v1.State_STATE_NONE}
	said, store, listened, err := a.heard(ctx, v, ref.Path)
	if err != nil || !listened {
		return out, err
	}
	switch put, err := store.Read(ctx, derived.Corrected(said.Producer, said.Hash)); {
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
func stood(v domain.Vault, path, id string, got reached) *v1.Artifact {
	out := &v1.Artifact{Name: named(v, path, id), Size: int64(got.size)}
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

// known says whether an id names an artifact at all, whatever file it is asked
// of.
func known(id string) bool {
	return id == readingID || id == heardID || id == correctedID
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
