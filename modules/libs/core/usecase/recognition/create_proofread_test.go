package recognition

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// replying is a proofreader that is asked nothing: what a run is built with,
// and the instruction it was opened under.
type replying struct{ told string }

func (r *replying) GetName() string { return "replying" }

func (r *replying) Proofread(context.Context, []domain.ProofreadPrompt) (map[int]string, error) {
	return nil, nil
}

// queued is a proofreader with a queue, which is asked nothing either.
type queued struct{ replying }

func (q *queued) Submit(context.Context, []domain.ProofreadPrompt) (string, error) { return "", nil }

func (q *queued) GetReplies(context.Context, string) (map[int]string, bool, error) {
	return nil, false, nil
}

// somewhereToProofread is a vault to read out of and a store to keep the
// corrections in. A proofreading is refused without them, so the tests here
// hand over both and watch the proofreader.
func somewhereToProofread() (port.VaultReaders, port.DerivedStores) {
	return vaults{}, shelves{newShelf()}
}

// newProofreadingConfig is an installation with a proofreader, at the sizes it
// proofreads at.
func newProofreadingConfig(by *replying, queue *queued) run.ProofreadingConfig {
	return run.ProofreadingConfig{
		IsNamed:         true,
		By:              func(told string) (port.Proofreader, error) { by.told = told; return by, nil },
		Queue:           func(told string) (port.ProofreadQueue, error) { queue.told = told; return queue, nil },
		Batch:           12,
		Overlap:         3,
		InFlight:        5,
		MaxEditDistance: 0.25,
	}
}

// A recognition is put right at the sizes the installation proofreads at, and the
// pages are left with the queue where there is one.
func TestARecognitionIsBuiltAtTheSizesTheInstallationProofreadsAt(t *testing.T) {
	t.Parallel()
	by, queue := &replying{}, &queued{}
	readers, derived := somewhereToProofread()

	right, held, err := CreateProofread(newProofreadingConfig(by, queue), readers, derived)
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
	if right.Queue != port.ProofreadQueue(queue) {
		t.Error("the pages are left nowhere, and the installation named a queue")
	}
}

// A document's recognition is proofread under the instruction written for a
// scan.
func TestARecognitionIsProofreadAsAScan(t *testing.T) {
	t.Parallel()
	by, queue := &replying{}, &queued{}
	readers, derived := somewhereToProofread()
	if _, _, err := CreateProofread(newProofreadingConfig(by, queue), readers, derived); err != nil {
		t.Fatal(err)
	}
	if by.told != proofread.ScanInstruction || queue.told != proofread.ScanInstruction {
		t.Error("a document's recognition is not proofread as a scan")
	}
}

// An installation that placed no proofreader holds none, and that is no
// failure: the recognition is used exactly as it was read.
func TestNamingNoProofreaderHoldsNoneAndIsNoFailure(t *testing.T) {
	t.Parallel()
	readers, derived := somewhereToProofread()
	for _, said := range []run.ProofreadingConfig{
		{},
		{By: func(string) (port.Proofreader, error) { return nil, nil }},
	} {
		right, held, err := CreateProofread(said, readers, derived)
		if err != nil || held || right.by != nil {
			t.Errorf("a recognition came back: held %v, %v", held, err)
		}
	}
}

// A proofreader that cannot be opened — no key, no address — says so, and says
// what could not be done with it.
func TestAProofreaderThatCannotBeOpenedSaysWhatCouldNotBeDone(t *testing.T) {
	t.Parallel()
	keyless := errors.New("no key is configured")
	said := run.ProofreadingConfig{
		IsNamed: true,
		By:      func(string) (port.Proofreader, error) { return nil, keyless },
	}
	readers, derived := somewhereToProofread()

	_, held, err := CreateProofread(said, readers, derived)
	if held || !errors.Is(err, keyless) {
		t.Fatalf("held %v, %v", held, err)
	}
	if !strings.Contains(err.Error(), "nothing to proofread with") {
		t.Errorf("%q does not say what could not be done", err)
	}
}

// A queue that cannot be opened is its own answer: there is a proofreader, and
// nowhere to leave the pages for it.
func TestAQueueThatCannotBeOpenedSaysThePagesCannotBeLeft(t *testing.T) {
	t.Parallel()
	shut := errors.New("the batch address is not reachable")
	by := &replying{}
	said := run.ProofreadingConfig{
		IsNamed: true,
		By:      func(string) (port.Proofreader, error) { return by, nil },
		Queue:   func(string) (port.ProofreadQueue, error) { return nil, shut },
	}
	readers, derived := somewhereToProofread()

	_, held, err := CreateProofread(said, readers, derived)
	if held || !errors.Is(err, shut) {
		t.Fatalf("held %v, %v", held, err)
	}
	if !strings.Contains(err.Error(), "nothing to leave the pages with") {
		t.Errorf("%q does not say what could not be done", err)
	}
}
