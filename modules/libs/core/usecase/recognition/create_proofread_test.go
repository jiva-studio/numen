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

func (r *replying) ProofreadScan(context.Context, []domain.ProofreadPage, float64) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, nil
}

func (r *replying) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, domain.SpeechWindow,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, errSpeechNotProofread
}

// somewhereToProofread is a vault to read out of and a store to keep the
// corrections in. A proofreading is refused without them, so the tests here
// hand over both and watch the proofreader.
func somewhereToProofread() (port.VaultReaders, port.DerivedStores) {
	return vaults{}, shelves{newShelf()}
}

// newProofreadingConfig is an installation with a proofreader, at the sizes it
// proofreads at.
func newProofreadingConfig(by *replying) run.ProofreadingConfig {
	return run.ProofreadingConfig{
		IsNamed:         true,
		By:              func() (port.Proofreader, error) { return by, nil },
		Batch:           12,
		Overlap:         3,
		InFlight:        5,
		MaxEditDistance: 0.25,
	}
}

// A recognition is put right at the sizes the installation proofreads at.
func TestARecognitionIsBuiltAtTheSizesTheInstallationProofreadsAt(t *testing.T) {
	t.Parallel()
	by := &replying{}
	readers, derived := somewhereToProofread()

	right, held, err := CreateProofread(newProofreadingConfig(by), readers, derived)
	if err != nil || !held {
		t.Fatalf("held %v, %v", held, err)
	}
	if right.Pages != 12 {
		t.Errorf("%d pages a request, and the installation asks about 12", right.Pages)
	}
	if right.MaxEditDistance != 0.25 {
		t.Errorf("a correction may stand %v from the line, and the installation says 0.25",
			right.MaxEditDistance)
	}
	if right.by != port.Proofreader(by) {
		t.Error("a recognition is put right by something the installation did not place")
	}
}

// An installation that placed no proofreader holds none, and that is no
// failure: the recognition is used exactly as it was read.
func TestNamingNoProofreaderHoldsNoneAndIsNoFailure(t *testing.T) {
	t.Parallel()
	readers, derived := somewhereToProofread()
	for _, said := range []run.ProofreadingConfig{
		{},
		{By: func() (port.Proofreader, error) { return nil, nil }},
	} {
		right, held, err := CreateProofread(said, readers, derived)
		if err != nil || held || right.by != nil {
			t.Errorf("a recognition came back: held %v, %v", held, err)
		}
	}
}

// A proofreader that cannot be opened says so, and says what could not be done
// with it.
func TestAProofreaderThatCannotBeOpenedSaysWhatCouldNotBeDone(t *testing.T) {
	t.Parallel()
	absent := errors.New("no proofreading plugin is installed")
	said := run.ProofreadingConfig{
		IsNamed: true,
		By:      func() (port.Proofreader, error) { return nil, absent },
	}
	readers, derived := somewhereToProofread()

	_, held, err := CreateProofread(said, readers, derived)
	if held || !errors.Is(err, absent) {
		t.Fatalf("held %v, %v", held, err)
	}
	if !strings.Contains(err.Error(), "nothing to proofread with") {
		t.Errorf("%q does not say what could not be done", err)
	}
}
