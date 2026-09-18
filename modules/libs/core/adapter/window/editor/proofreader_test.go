package editor

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// proofreads is the proofreading a test hands the window: whether this
// installation names anything to put a transcript right with, what it was asked
// to put right, and what it says came of the ask.
type proofreads struct {
	isReady bool
	came    source.ProofreadTranscriptResult
	why     error

	vault string
	path  string
	times int
}

func (p *proofreads) ProofreaderReady() bool { return p.isReady }

func (p *proofreads) Proofread(
	_ context.Context,
	v domain.Vault,
	path string,
) (source.ProofreadTranscriptResult, error) {
	p.times++
	p.vault, p.path = string(v.ID), path
	return p.came, p.why
}

// newProofreadWindow is a window over the same vault the runs are asked for over,
// with a proofreading a test watches.
func newProofreadWindow(t *testing.T, held port.DerivedStore, read indexed) (*API, http.Handler, *proofreads) {
	t.Helper()
	api, handler := openRunWindow(t, held, read, willRun(), willRun())
	by := &proofreads{isReady: true}
	setPasses(api, func(on *passes) { on.proofreads = by })
	return api, handler, by
}

// onTheShelf is a store holding the transcript a model wrote of the recording.
func onTheShelf() stored {
	return stored{derived.Artifact(asr, hashed): []byte("what the model heard")}
}

// A transcript the window asks for is put right, the run is told which file of
// which vault, and the corrections are said to be under way.
func TestATranscriptIsProofreadWhenTheWindowAsksForIt(t *testing.T) {
	api, _, by := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())

	made := createArtifact(t, api, talk, correctedOf)
	if made.GetState() != v1.State_STATE_RUNNING {
		t.Fatalf("the transcript was answered %s", made.GetState())
	}
	if made.GetKind() != v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED {
		t.Errorf("the answer is about a %s", made.GetKind())
	}
	if by.times != 1 || by.path != talk || by.vault != string(api.GetShownVault().ID) {
		t.Errorf("the run was asked for %q of %q, %d times", by.path, by.vault, by.times)
	}
}

// Every way a run can come to nothing is a state the window draws, so that
// pressing the item is never silence.
func TestWhatAProofreadingCameToIsSaid(t *testing.T) {
	for _, one := range []struct {
		name  string
		came  source.ProofreadTranscriptResult
		state v1.State
	}{
		{"a recording another run holds", source.ProofreadTranscriptResult{IsBusy: true}, v1.State_STATE_RUNNING},
		{"a transcript already put right", source.ProofreadTranscriptResult{IsAlready: true}, v1.State_STATE_DONE},
		{"words a person wrote themselves", source.ProofreadTranscriptResult{IsEdited: true}, v1.State_STATE_DONE},
		{"a transcript holding no words", source.ProofreadTranscriptResult{IsNone: true}, v1.State_STATE_NONE},
	} {
		t.Run(one.name, func(t *testing.T) {
			api, _, by := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())
			by.came = one.came

			if made := createArtifact(t, api, talk, correctedOf); made.GetState() != one.state {
				t.Fatalf("it was answered %s", made.GetState())
			}
		})
	}
}

// A recording the index says nothing has listened to holds no words to put
// right, and the run is never asked for.
func TestARecordingNothingHasListenedToHasNothingToProofread(t *testing.T) {
	api, _, by := newProofreadWindow(t, stored{}, newEmptyIndex())

	if code := getRefusedCode(t, api, talk, correctedOf); code != connect.CodeFailedPrecondition {
		t.Fatalf("a recording nothing has heard was refused %s", code)
	}
	if by.times != 0 {
		t.Error("a run was given a recording holding no words")
	}
	// Nothing stands, and the window that asked what the recording carries is
	// told so.
	if state := getArtifactStates(t, api, talk)[correctedOf]; state != v1.State_STATE_NONE {
		t.Errorf("the corrections of a recording nothing heard are %s", state)
	}
}

// A file that is not a recording carries no transcript, and nothing is given to
// the run.
func TestOnlyARecordingsTranscriptIsProofread(t *testing.T) {
	for _, path := range []string{book, idea} {
		t.Run(path, func(t *testing.T) {
			api, _, by := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())

			if code := getRefusedCode(t, api, path, correctedOf); code != connect.CodeInvalidArgument {
				t.Fatalf("a file of the wrong kind was refused %s", code)
			}
			if by.times != 0 {
				t.Error("a run was given a file carrying no transcript")
			}
		})
	}
}

// An installation naming nothing to put a transcript right with says so, and
// the window offers the run nowhere after that. Which model does it is the
// settings', so the call was answered and the answer is that there is nothing
// to ask.
func TestAnInstallationNamingNoProofreaderSaysSo(t *testing.T) {
	api, _, _ := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())
	setPasses(api, func(on *passes) { on.proofreads = &proofreads{} })

	if code := getRefusedCode(t, api, talk, correctedOf); code != connect.CodeFailedPrecondition {
		t.Fatalf("it was refused %s", code)
	}
}

// A vault whose passes are not up yet holds nothing to put a transcript right
// with, and the caller asks again.
func TestAProofreadingAskedForBeforeThePassesAreUpIsAskedAgain(t *testing.T) {
	api, _, _ := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())
	setPasses(api, func(on *passes) { on.proofreads = nil })

	if code := getRefusedCode(t, api, talk, correctedOf); code != connect.CodeUnavailable {
		t.Fatalf("it was refused %s", code)
	}
}

// The corrections a recording carries are what stands beside the transcript,
// and asking what it carries begins no run.
func TestTheCorrectionsARecordingCarriesAreWhatStands(t *testing.T) {
	const put = "the name and the named are not two"
	held := onTheShelf()
	held[derived.Corrections(asr, hashed)] = []byte(put)
	api, _, by := newProofreadWindow(t, held, newTranscriptIndex())

	out, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: talk}))
	if err != nil {
		t.Fatalf("asked what the recording carries and was refused: %v", err)
	}
	var corrections *v1.Artifact
	for _, one := range out.Msg.GetArtifacts() {
		if one.GetKind() == v1.ArtifactKind_ARTIFACT_KIND_TRANSCRIPT_CORRECTED {
			corrections = one
		}
	}
	if corrections == nil {
		t.Fatal("the recording carries no corrections at all")
	}
	if corrections.GetState() != v1.State_STATE_DONE {
		t.Errorf("corrections that stand are %s", corrections.GetState())
	}
	if by.times != 0 {
		t.Error("a run was begun by a question that only asked")
	}
}
