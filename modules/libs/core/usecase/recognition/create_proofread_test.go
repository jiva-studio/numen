package recognition

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// replying is a proofreader that is asked nothing: what a run is built with.
type replying struct{}

func (r *replying) GetName() string { return "replying" }

func (r *replying) ProofreadScan(context.Context, []domain.ProofreadPage) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, nil
}

func (r *replying) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, errSpeechNotProofread
}

// somewhereToProofread is a vault to read out of and a store to keep the
// corrections in. A proofreading is refused without them, so the tests here
// hand over both and watch the proofreader.
func somewhereToProofread() (port.VaultReaders, port.DerivedStores) {
	return vaults{}, fakeStores{newShelf()}
}

// newProofreadingConfig is an installation with a proofreader.
func newProofreadingConfig(by *replying) run.ProofreadingConfig {
	return run.ProofreadingConfig{
		IsNamed: true,
		By:      func() (port.Proofreader, error) { return by, nil },
	}
}

// A recognition is put right by the proofreader the installation names.
func TestARecognitionIsBuiltWithTheConfiguredProofreader(t *testing.T) {
	t.Parallel()
	by := &replying{}
	readers, derived := somewhereToProofread()

	right, target, err := CreateProofread(newProofreadingConfig(by), readers, derived)
	if err != nil || !target {
		t.Fatalf("held %v, %v", target, err)
	}
	if right.by != port.Proofreader(by) {
		t.Error("a recognition is put right by something the installation did not place")
	}
}

// An installation that placed no proofreader holds none, and that is no
// failure: the recognition is used exactly as it was read.
func TestNamingNoProofreaderContainsNoneAndIsNoFailure(t *testing.T) {
	t.Parallel()
	readers, derived := somewhereToProofread()
	for _, response := range []run.ProofreadingConfig{
		{},
		{By: func() (port.Proofreader, error) { return nil, nil }},
	} {
		right, target, err := CreateProofread(response, readers, derived)
		if err != nil || target || right.by != nil {
			t.Errorf("a recognition came back: held %v, %v", target, err)
		}
	}
}

// A proofreader that cannot be opened says so, and says what could not be done
// with it.
func TestAProofreaderThatCannotBeOpenedReturnsWhatCouldNotBeCompleted(t *testing.T) {
	t.Parallel()
	absent := errors.New("no proofreading plugin is installed")
	response := run.ProofreadingConfig{
		IsNamed: true,
		By:      func() (port.Proofreader, error) { return nil, absent },
	}
	readers, derived := somewhereToProofread()

	_, target, err := CreateProofread(response, readers, derived)
	if target || !errors.Is(err, absent) {
		t.Fatalf("held %v, %v", target, err)
	}
	if !strings.Contains(err.Error(), "nothing to proofread with") {
		t.Errorf("%q does not say what could not be done", err)
	}
}
