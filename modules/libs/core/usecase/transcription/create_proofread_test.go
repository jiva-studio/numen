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
	context.Context, []domain.ProofreadLine, domain.SpeechWindow,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, nil
}

// somewhereToProofread is a vault to read out of and a store to keep the
// corrections in. A proofreading is refused without them, so the tests here
// hand over both and watch the proofreader.
func somewhereToProofread() (port.VaultReaders, port.DerivedStores) {
	return vaults{}, shelves{newShelf()}
}

// A transcript is put right in requests that hold lines over from the one
// before, several of them answered in one call.
func TestATranscriptIsBuiltAtTheSizesTheInstallationProofreadsAt(t *testing.T) {
	t.Parallel()
	by := &replying{}
	config := run.ProofreadingConfig{
		IsNamed:  true,
		By:       func() (port.Proofreader, error) { return by, nil },
		Batch:    12,
		Overlap:  3,
		InFlight: 5,
	}
	readers, derived := somewhereToProofread()

	right, held, err := CreateProofread(config, readers, derived)
	if err != nil || !held {
		t.Fatalf("held %v, %v", held, err)
	}
	if right.BatchSize != 12 || right.Overlap != 3 || right.InFlight != 5 {
		t.Errorf("requests of %d holding %d over, %d to a call",
			right.BatchSize, right.Overlap, right.InFlight)
	}
	if right.by != port.Proofreader(by) {
		t.Error("a transcript is put right by something the installation did not place")
	}
}

// An installation that placed no proofreader holds none, and that is no
// failure: the transcript is used exactly as it was heard.
func TestNamingNoProofreaderHoldsNoneAndIsNoFailure(t *testing.T) {
	t.Parallel()
	readers, derived := somewhereToProofread()
	for _, said := range []run.ProofreadingConfig{
		{},
		{By: func() (port.Proofreader, error) { return nil, nil }},
	} {
		right, held, err := CreateProofread(said, readers, derived)
		if err != nil || held || right.by != nil {
			t.Errorf("a transcript came back: held %v, %v", held, err)
		}
	}
}

// A proofreader that cannot be opened — no key, no address — says so.
func TestAProofreaderThatCannotBeOpenedIsAnError(t *testing.T) {
	t.Parallel()
	keyless := errors.New("no key is configured")
	said := run.ProofreadingConfig{
		IsNamed: true,
		By:      func() (port.Proofreader, error) { return nil, keyless },
	}
	readers, derived := somewhereToProofread()

	if _, _, err := CreateProofread(said, readers, derived); !errors.Is(err, keyless) {
		t.Errorf("got %v", err)
	}
}
