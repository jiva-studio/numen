package editor

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// proofreads is the proofreading a test hands the window: whether this
// installation names anything to put a transcript right with, what it was asked
// to put right, and what it says came of the ask.
type proofreads struct {
	isReady bool
	came    transcription.ProofreadResult
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
) (transcription.ProofreadResult, error) {
	p.times++
	p.vault, p.path = string(v.ID), path
	return p.came, p.why
}

// newProofreadWindow is a window over the same vault the runs are asked for over,
// with a proofreading a test watches.
func newProofreadWindow(t *testing.T, held port.DerivedStore, read indexed) (*Services, http.Handler, *proofreads) {
	t.Helper()
	api, handler := openRunWindow(t, held, read, willRun(), willRun())
	by := &proofreads{isReady: true}
	modifySession(api, func(on *vaultSession) { on.transcriptProofreader = by })
	return api, handler, by
}

// onTheShelf is a store holding the transcript a model wrote of the recording.
func onTheShelf() stored {
	return stored{domain.NewArtifactName(domain.ArtifactText, asr, hashed): []byte("what the model heard")}
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
	if by.times != 1 || by.path != talk || by.vault != string(api.state.GetCurrentVault().ID) {
		t.Errorf("the run was asked for %q of %q, %d times", by.path, by.vault, by.times)
	}
}

// Every way a run can come to nothing is a state the window draws, so that
// pressing the item is never silence.
func TestWhatAProofreadingCameToIsSaid(t *testing.T) {
	for _, one := range []struct {
		name  string
		came  transcription.ProofreadResult
		state v1.State
	}{
		{"a recording another run holds", transcription.ProofreadResult{IsBusy: true}, v1.State_STATE_RUNNING},
		{"a transcript already put right", transcription.ProofreadResult{IsAlready: true}, v1.State_STATE_DONE},
		{"words a person wrote themselves", transcription.ProofreadResult{IsEdited: true}, v1.State_STATE_DONE},
		{"a transcript holding no words", transcription.ProofreadResult{IsNone: true}, v1.State_STATE_NONE},
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
		t.Fatalf("a recording nothing has heard failed %s", code)
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
				t.Fatalf("a file of the wrong kind failed %s", code)
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
func TestAnInstallationNamingNoProofreaderReturnsSo(t *testing.T) {
	api, _, _ := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())
	modifySession(api, func(on *vaultSession) { on.transcriptProofreader = &proofreads{} })

	if code := getRefusedCode(t, api, talk, correctedOf); code != connect.CodeFailedPrecondition {
		t.Fatalf("it failed %s", code)
	}
}

// A vault whose passes are not up yet holds nothing to put a transcript right
// with, and the caller asks again.
func TestAProofreadingAskedForBeforeThePassesAreUpIsAskedAgain(t *testing.T) {
	api, _, _ := newProofreadWindow(t, onTheShelf(), newTranscriptIndex())
	modifySession(api, func(on *vaultSession) { on.transcriptProofreader = nil })

	if code := getRefusedCode(t, api, talk, correctedOf); code != connect.CodeUnavailable {
		t.Fatalf("it failed %s", code)
	}
}

// The corrections a recording carries are what stands beside the transcript,
// and asking what it carries begins no run.
func TestTheCorrectionsARecordingCarriesAreWhatExists(t *testing.T) {
	const placed = "the name and the named are not two"
	target := onTheShelf()
	target[domain.NewArtifactName(domain.ArtifactCorrections, asr, hashed)] = []byte(placed)
	api, _, by := newProofreadWindow(t, target, newTranscriptIndex())

	out, err := api.Artifact.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: talk}))
	if err != nil {
		t.Fatalf("asked what the recording carries and failed: %v", err)
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
