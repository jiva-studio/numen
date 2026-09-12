package source

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// replying is a proofreader that is asked nothing: what a run is built with,
// and the instruction it was opened under.
type replying struct{ told string }

func (r *replying) Name() string { return "replying" }

func (r *replying) Proofread(context.Context, []proofread.Batch) (map[int]string, error) {
	return nil, nil
}

// queued is a proofreader with a queue, which is asked nothing either.
type queued struct{ replying }

func (q *queued) Leave(context.Context, []proofread.Batch) (string, error) { return "", nil }

func (q *queued) Collect(context.Context, string) (map[int]string, bool, error) {
	return nil, false, nil
}

// newProofreadingConfig is an installation with a proofreader for both kinds of
// text, at the sizes it proofreads at.
func newProofreadingConfig(by *replying, queue *queued) ProofreadingConfig {
	return ProofreadingConfig{
		Named:           true,
		By:              func(told string) (port.Proofreader, error) { by.told = told; return by, nil },
		Queue:           func(told string) (port.ProofreadQueue, error) { queue.told = told; return queue, nil },
		Batch:           12,
		Overlap:         3,
		InFlight:        5,
		MaxEditDistance: 0.25,
	}
}

// A reading is put right at the sizes the installation proofreads at, and the
// pages are left with the queue where there is one.
func TestAReadingIsBuiltAtTheSizesTheInstallationProofreadsAt(t *testing.T) {
	t.Parallel()
	by, queue := &replying{}, &queued{}

	right, held, err := newProofreadingConfig(by, queue).Reading(nil, nil)
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
	if right.By != port.Proofreader(by) {
		t.Error("a reading is put right by something the installation did not place")
	}
	if right.Queue != port.ProofreadQueue(queue) {
		t.Error("the pages are left nowhere, and the installation named a queue")
	}
}

// A transcript is put right in batches that hold lines over from the one
// before, several of them standing out at once.
func TestATranscriptIsBuiltAtTheSizesTheInstallationProofreadsAt(t *testing.T) {
	t.Parallel()
	by, queue := &replying{}, &queued{}

	right, held, err := newProofreadingConfig(by, queue).Transcript(nil, nil)
	if err != nil || !held {
		t.Fatalf("held %v, %v", held, err)
	}
	if right.BatchSize != 12 || right.Overlap != 3 || right.InFlight != 5 {
		t.Errorf("batches of %d holding %d over, %d at once",
			right.BatchSize, right.Overlap, right.InFlight)
	}
	if right.By != port.Proofreader(by) {
		t.Error("a transcript is put right by something the installation did not place")
	}
}

// Each kind of text is proofread under the instruction written for it: what a
// machine read off a page and what it heard are two different jobs.
func TestEachKindOfTextIsProofreadUnderItsOwnInstruction(t *testing.T) {
	t.Parallel()
	by, queue := &replying{}, &queued{}
	if _, _, err := newProofreadingConfig(by, queue).Reading(nil, nil); err != nil {
		t.Fatal(err)
	}
	if by.told != proofread.ScanInstruction || queue.told != proofread.ScanInstruction {
		t.Error("a document's reading is not proofread as a scan")
	}

	by, queue = &replying{}, &queued{}
	if _, _, err := newProofreadingConfig(by, queue).Transcript(nil, nil); err != nil {
		t.Fatal(err)
	}
	if by.told != proofread.SpeechInstruction {
		t.Error("a transcript is not proofread as speech")
	}
}

// An installation that placed no proofreader holds none, and that is no
// failure: the text is used exactly as it was made.
func TestNamingNoProofreaderHoldsNoneAndIsNoFailure(t *testing.T) {
	t.Parallel()
	for _, said := range []ProofreadingConfig{
		{},
		{By: func(string) (port.Proofreader, error) { return nil, nil }},
	} {
		right, held, err := said.Reading(nil, nil)
		if err != nil || held || right.By != nil {
			t.Errorf("a reading came back: held %v, %v", held, err)
		}
		transcript, held, err := said.Transcript(nil, nil)
		if err != nil || held || transcript.By != nil {
			t.Errorf("a transcript came back: held %v, %v", held, err)
		}
	}
}

// A proofreader that cannot be opened — no key, no address — says so, and says
// what could not be done with it.
func TestAProofreaderThatCannotBeOpenedSaysWhatCouldNotBeDone(t *testing.T) {
	t.Parallel()
	keyless := errors.New("no key is configured")
	said := ProofreadingConfig{
		Named: true,
		By:    func(string) (port.Proofreader, error) { return nil, keyless },
	}

	_, held, err := said.Reading(nil, nil)
	if held || !errors.Is(err, keyless) {
		t.Fatalf("held %v, %v", held, err)
	}
	if !strings.Contains(err.Error(), "nothing to proofread with") {
		t.Errorf("%q does not say what could not be done", err)
	}

	if _, _, err := said.Transcript(nil, nil); !errors.Is(err, keyless) {
		t.Errorf("got %v", err)
	}
}

// A queue that cannot be opened is its own answer: there is a proofreader, and
// nowhere to leave the pages for it.
func TestAQueueThatCannotBeOpenedSaysThePagesCannotBeLeft(t *testing.T) {
	t.Parallel()
	shut := errors.New("the batch address is not reachable")
	by := &replying{}
	said := ProofreadingConfig{
		Named: true,
		By:    func(string) (port.Proofreader, error) { return by, nil },
		Queue: func(string) (port.ProofreadQueue, error) { return nil, shut },
	}

	_, held, err := said.Reading(nil, nil)
	if held || !errors.Is(err, shut) {
		t.Fatalf("held %v, %v", held, err)
	}
	if !strings.Contains(err.Error(), "nothing to leave the pages with") {
		t.Errorf("%q does not say what could not be done", err)
	}
}
