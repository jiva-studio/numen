package source

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// shown is what the list of what is being done says about putting a transcript
// right.
func shown(held *TranscriptionWorker, path string) (doing, failed string) {
	for _, at := range held.with.Tasks.List() {
		if at.ID == proofreadingID(path) {
			return at.Doing, at.Failed
		}
	}
	return "", ""
}

// nowhere is a profile that cannot be opened, as a person naming one no
// settings name is told.
func nowhere(string) (port.Proofreader, error) {
	return nil, errors.New(`no proofreading profile named "nowhere"`)
}

// An installation that did not ask for its transcripts to be put right asks
// nothing and says nothing.
func TestATranscriptIsPutRightOnlyWhereItWasAskedFor(t *testing.T) {
	held, v := listens(t, &deaf{}, "talks/one.mp3")
	held.with.Proofreading = ProofreadingConfig{Named: true, By: nowhere}

	held.proofread(t.Context(), v, "talks/one.mp3", false)

	if doing, _ := shown(held, "talks/one.mp3"); doing != "" {
		t.Errorf("a transcript nobody asked about is %q", doing)
	}
}

// A profile no settings name is a person's mistake, and they are shown it.
func TestAProfileNoSettingsNameIsShown(t *testing.T) {
	held, v := listens(t, &deaf{}, "talks/one.mp3")
	held.with.Proofreading = ProofreadingConfig{Named: true, Automatically: true, By: nowhere}

	held.proofread(t.Context(), v, "talks/one.mp3", false)

	doing, failed := shown(held, "talks/one.mp3")
	if doing != "Proofreading a transcript" {
		t.Fatalf("the list says %q", doing)
	}
	if !strings.Contains(failed, "nowhere") {
		t.Errorf("the failure does not name the profile: %q", failed)
	}
}

// Silence carries no words, so nothing is asked about it.
func TestSilenceIsNotPutRight(t *testing.T) {
	by := &deaf{}
	held, v := listens(t, by, "talks/one.mp3")
	held.with.Proofreading = ProofreadingConfig{Named: true, Automatically: true, By: nowhere}

	held.Start(v, "talks/one.mp3")
	held.Wait()

	if doing, failed := shown(held, "talks/one.mp3"); doing != "" {
		t.Errorf("silence is %q, failing with %q", doing, failed)
	}
}

// puts is a proofreader answering with what it was told to say about a batch,
// and keeping the lines it was asked about.
type puts struct {
	says map[int]string

	mu    sync.Mutex
	asked []int
}

func (p *puts) Name() string { return "a proofreader" }

func (p *puts) Proofread(_ context.Context, batches []proofread.Batch) (map[int]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := map[int]string{}
	for _, batch := range batches {
		for _, line := range batch.Lines {
			p.asked = append(p.asked, line.Number)
		}
		if said, held := p.says[batch.Number]; held {
			out[batch.Number] = said
		}
	}
	return out, nil
}

// lines is what this proofreader was asked about.
func (p *puts) lines() []int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.asked)
}

// cues is the words as the cues a model wrote them down as, one second apart.
func cues(words []string) []transcript.Cue {
	out := make([]transcript.Cue, 0, len(words))
	for at, said := range words {
		out = append(out, transcript.Cue{Text: said, From: at * 1000, To: at*1000 + 800})
	}
	return out
}

// stopped is a TranscriptionWorker over a vault holding one recording, with its
// transcript on the shelf and a proofreading of it standing at through lines.
//
// One line to a batch and one batch to a request, so the number a reply is
// about is the number of the line it puts right.
func stopped(
	t *testing.T,
	by port.Proofreader,
	through int,
	words ...string,
) (*TranscriptionWorker, domain.Vault, port.DerivedStore, string) {
	t.Helper()
	held, v := listens(t, &deaf{}, recording)
	held.with.Proofreading = ProofreadingConfig{
		Named: true, Automatically: true, Batch: 1, InFlight: 1,
		By: func(string) (port.Proofreader, error) { return by, nil },
	}

	reader, err := held.with.Readers.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := reader.Read(t.Context(), recording)
	if err != nil {
		t.Fatal(err)
	}
	hash := text.Fingerprint(raw)
	store, err := held.with.Derived.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	spoken := cues(words)
	if err := store.Write(t.Context(), text.Artifact(text.ASR, hash), transcript.Marshal(spoken)); err != nil {
		t.Fatal(err)
	}
	if through == 0 {
		return held, v, store, hash
	}

	// The shelf as a run that ended among the batches left it: the lines it put
	// right, and the count standing after them.
	for at := range through {
		spoken[at].Text = strings.ToUpper(spoken[at].Text)
	}
	if err := store.Write(t.Context(), text.Corrections(text.ASR, hash), transcript.Marshal(spoken)); err != nil {
		t.Fatal(err)
	}
	stood, err := json.Marshal(putting{By: by.Name(), At: spoken[through-1].To})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Write(t.Context(), text.Proofread(text.ASR, hash), stood); err != nil {
		t.Fatal(err)
	}
	return held, v, store, hash
}

const recording = "talks/one.mp3"

// written is what the transcript on the shelf says, line by line.
func written(t *testing.T, store port.DerivedStore, name string) []string {
	t.Helper()
	raw, err := store.Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	_, said := transcript.Parse(raw)
	out := make([]string, 0, len(said))
	for _, cue := range said {
		out = append(out, cue.Text)
	}
	return out
}

// A proofreading that ended among the batches stands at the line it reached,
// and the lines after it are asked about when the application opens.
func TestATranscriptLeftPartWayThroughIsTakenUpWhenTheApplicationOpens(t *testing.T) {
	by := &puts{says: map[int]string{
		1: "1|SECOND THING",
		2: "2|THIRD THING",
	}}
	held, v, store, hash := stopped(t, by, 1, "first thing", "second thing", "third thing")

	held.TakingUp(t.Context(), recognised{recording}, v)
	held.Wait()

	if got := by.lines(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	want := []string{"FIRST THING", "SECOND THING", "THIRD THING"}
	if got := written(t, store, text.Corrections(text.ASR, hash)); !slices.Equal(got, want) {
		t.Errorf("the transcript says %q", got)
	}
	if doing, failed := shown(held, recording); doing != "" {
		t.Errorf("the transcript is left in the list as %q, failing with %q", doing, failed)
	}
}

// A transcript put right to its last line is asked about nothing and shown
// nowhere, however often the application opens.
func TestATranscriptAlreadyPutRightIsAskedAboutNothing(t *testing.T) {
	by := &puts{}
	held, v, _, _ := stopped(t, by, 3, "first thing", "second thing", "third thing")

	held.TakingUp(t.Context(), recognised{recording}, v)
	held.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if doing, failed := shown(held, recording); doing != "" {
		t.Errorf("the transcript is in the list as %q, failing with %q", doing, failed)
	}
}

// An installation that did not ask for its transcripts to be put right takes up
// nothing.
func TestNoTranscriptIsTakenUpWhereItWasNotAskedFor(t *testing.T) {
	by := &puts{says: map[int]string{1: "1|SECOND THING"}}
	held, v, _, _ := stopped(t, by, 1, "first thing", "second thing")
	held.with.Proofreading.Automatically = false

	held.TakingUp(t.Context(), recognised{recording}, v)
	held.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// One run to a transcript. A run that finds it held leaves the list to the run
// that holds it.
func TestATranscriptAnotherRunHoldsKeepsItsPlaceInTheList(t *testing.T) {
	by := &puts{}
	held, v, store, hash := stopped(t, by, 1, "first thing", "second thing")
	release, err := store.Claim(t.Context(), text.Partial(text.ASR, hash))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	held.with.Tasks.Set(task.Task{
		ID: proofreadingID(recording), Doing: "Proofreading a transcript", About: recording,
		Count: 1, Total: 2,
	})

	held.TakingUp(t.Context(), recognised{recording}, v)
	held.Wait()

	if doing, _ := shown(held, recording); doing != "Proofreading a transcript" {
		t.Errorf("the run holding the transcript is in the list as %q", doing)
	}
	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// refuses is a proofreader that will not answer about anything.
type refuses struct{ why error }

func (r *refuses) Name() string { return "a proofreader that will not answer" }

func (r *refuses) Proofread(context.Context, []proofread.Batch) (map[int]string, error) {
	return nil, r.why
}

// A transcript this proofreader has been over every line of is asked about
// nothing, and whoever asked is told that rather than nothing at all.
func TestATranscriptAlreadyPutRightSaysThatNothingWasLeft(t *testing.T) {
	by := &puts{}
	held, v, _, _ := stopped(t, by, 2, "first thing", "second thing")

	res, err := held.Proofread(t.Context(), v, recording)
	held.Wait()

	if err != nil {
		t.Fatalf("putting a transcript right again failed: %v", err)
	}
	if !res.Already {
		t.Errorf("a transcript nothing was left of answered %+v", res)
	}
	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// A run with work to do answers as soon as it has a question to put, and does
// not keep whoever asked waiting for the whole of it.
func TestAProofreadingWithWorkToDoAnswersBeforeItIsOver(t *testing.T) {
	stand := make(chan struct{})
	by := &waits{on: stand, asked: make(chan struct{})}
	held, v, _, _ := stopped(t, by, 0, "first thing", "second thing")

	res, err := held.Proofread(t.Context(), v, recording)

	if err != nil {
		t.Fatalf("a run that began answered %v", err)
	}
	if res.Already || res.None || res.Edited || res.Busy {
		t.Errorf("a run with work to do answered %+v", res)
	}
	<-by.asked
	close(stand)
	held.Wait()
}

// waits is a proofreader that says it was asked and then waits to be let go.
type waits struct {
	on    chan struct{}
	asked chan struct{}
	once  sync.Once
}

func (w *waits) Name() string { return "a proofreader that waits" }

func (w *waits) Proofread(context.Context, []proofread.Batch) (map[int]string, error) {
	w.once.Do(func() { close(w.asked) })
	<-w.on
	return map[int]string{}, nil
}

// A run a person asked for stands in the list from the moment they asked, and
// not from the moment it has something to count. Opening the proofreader is the
// first thing it does, and the list already says so there.
func TestAProofreadingAskedForStandsInTheListBeforeItOpensTheProofreader(t *testing.T) {
	by := &puts{}
	held, v, _, _ := stopped(t, by, 0, "first thing", "second thing")
	reached, stand := make(chan struct{}), make(chan struct{})
	held.with.Proofreading.By = func(string) (port.Proofreader, error) {
		close(reached)
		<-stand
		return by, nil
	}

	go func() { _, _ = held.Proofread(t.Context(), v, recording) }()
	<-reached
	doing, _ := shown(held, recording)
	close(stand)
	held.Wait()

	if doing != "Proofreading a transcript" {
		t.Errorf("a run a person asked for is in the list as %q", doing)
	}
}

// A run that ends leaves nothing behind in the list.
func TestAProofreadingThatEndsLeavesTheListEmpty(t *testing.T) {
	by := &puts{says: map[int]string{0: "0|FIRST THING"}}
	held, v, _, _ := stopped(t, by, 0, "first thing")

	_, _ = held.Proofread(t.Context(), v, recording)
	held.Wait()

	if doing, failed := shown(held, recording); doing != "" {
		t.Errorf("a run that ended is in the list as %q, failing with %q", doing, failed)
	}
}

// A proofreader that will not answer is a failure a person is shown, and the
// failure stays in the list.
func TestAProofreaderThatWillNotAnswerIsShownAsAFailure(t *testing.T) {
	by := &refuses{why: errors.New("the command line is not on this machine")}
	held, v, _, _ := stopped(t, by, 0, "first thing", "second thing")

	_, _ = held.Proofread(t.Context(), v, recording)
	held.Wait()

	doing, failed := shown(held, recording)
	if doing != "Proofreading a transcript" {
		t.Fatalf("a run that failed is in the list as %q", doing)
	}
	if !strings.Contains(failed, "not on this machine") {
		t.Errorf("the failure says %q", failed)
	}
}
