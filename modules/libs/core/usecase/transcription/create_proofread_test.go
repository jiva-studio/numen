package transcription

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// replying is a proofreader that is asked nothing: what a run is built with.
type replying struct{ noScan }

func (r *replying) GetName() string { return "replying" }

func (r *replying) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, nil
}

// somewhereToProofread is a vault to read out of and a store to keep the
// hand over both and watch the proofreader.
func somewhereToProofread() (port.VaultReaders, port.DerivedStores) {
	return vaults{}, fakeStores{newShelf()}
}

// A transcript is put right with the proofreader the installation names.
func TestATranscriptIsBuiltWithTheConfiguredProofreader(t *testing.T) {
	t.Parallel()
	delta := &replying{}
	config := run.ProofreadingConfig{
		IsNamed: true,
		By:      func() (port.Proofreader, error) { return delta, nil },
	}
	readers, derived := somewhereToProofread()

	right, target, err := CreateProofread(config, readers, derived)
	if err != nil || !target {
		t.Fatalf("held %v, %v", target, err)
	}
	if right.by != port.Proofreader(delta) {
		t.Error("a transcript is put right by something the installation did not place")
	}
}

// An installation that placed no proofreader holds none, and that is no
// failure: the transcript is used exactly as it was heard.
func TestNamingNoProofreaderContainsNoneAndIsNoFailure(t *testing.T) {
	t.Parallel()
	readers, derived := somewhereToProofread()
	for _, response := range []run.ProofreadingConfig{
		{},
		{By: func() (port.Proofreader, error) { return nil, nil }},
	} {
		right, target, err := CreateProofread(response, readers, derived)
		if err != nil || target || right.by != nil {
			t.Errorf("a transcript came back: held %v, %v", target, err)
		}
	}
}

// A proofreader that cannot be opened — no key, no address — says so.
func TestAProofreaderThatCannotBeOpenedIsAnError(t *testing.T) {
	t.Parallel()
	keyless := errors.New("no key is configured")
	response := run.ProofreadingConfig{
		IsNamed: true,
		By:      func() (port.Proofreader, error) { return nil, keyless },
	}
	readers, derived := somewhereToProofread()

	if _, _, err := CreateProofread(response, readers, derived); !errors.Is(err, keyless) {
		t.Errorf("got %v", err)
	}
}
