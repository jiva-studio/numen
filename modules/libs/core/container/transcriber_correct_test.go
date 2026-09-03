package container

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// said is what the list of what is being done says about putting a transcript
// right.
func said(held *Transcribing, path string) (doing, failed string) {
	for _, at := range held.tasks.List() {
		if at.ID == correcting(path) {
			return at.Doing, at.Failed
		}
	}
	return "", ""
}

// An installation that did not ask for its transcripts to be put right asks
// nothing and says nothing.
func TestATranscriptIsPutRightOnlyWhereItWasAskedFor(t *testing.T) {
	held, v := listens(t, &deaf{}, "talks/one.mp3")
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "nowhere"}

	held.correct(t.Context(), v, "talks/one.mp3", false)

	if doing, _ := said(held, "talks/one.mp3"); doing != "" {
		t.Errorf("a transcript nobody asked about is %q", doing)
	}
}

// A profile no settings name is a person's mistake, and they are shown it.
func TestAProfileNoSettingsNameIsShown(t *testing.T) {
	held, v := listens(t, &deaf{}, "talks/one.mp3")
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "nowhere", Automatically: true}

	held.correct(t.Context(), v, "talks/one.mp3", false)

	doing, failed := said(held, "talks/one.mp3")
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
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "nowhere", Automatically: true}

	held.Start(v, "talks/one.mp3")
	held.Wait()

	if doing, failed := said(held, "talks/one.mp3"); doing != "" {
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
			p.asked = append(p.asked, line.At)
		}
		if said, held := p.says[batch.At]; held {
			out[batch.At] = said
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

// spoken is the words as the cues a model wrote them down as, one second apart.
func spoken(words []string) []transcript.Cue {
	out := make([]transcript.Cue, 0, len(words))
	for at, said := range words {
		out = append(out, transcript.Cue{Text: said, From: at * 1000, To: at*1000 + 800})
	}
	return out
}

// putting is where a run putting a transcript right stood when it ended.
type putting struct {
	By string `json:"by"`
	At int    `json:"at"`
}

// stopped is a Transcribing over a vault holding one recording, with its
// transcript on the shelf and a proofreading of it standing at through lines.
//
// One line to a batch and one batch to a request, so the number a reply is
// about is the number of the line it puts right.
func stopped(
	t *testing.T,
	by port.Proofreader,
	through int,
	words ...string,
) (*Transcribing, domain.Vault, port.DerivedStore, string) {
	t.Helper()
	held, v := listens(t, &deaf{}, recording)
	held.cfg.Proofreading = proofreading.Defaults()
	held.cfg.Proofreading.Profiles = map[string]proofreading.Profile{
		"by hand": {Use: proofreading.UseAgent, Model: "a-model", BatchSize: 1, InFlight: 1},
	}
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "by hand", Automatically: true}
	held.cfg.AgentProofreader = func(AgentProofreading) (port.Proofreader, error) { return by, nil }

	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(recording)))
	if err != nil {
		t.Fatal(err)
	}
	hash := text.Fingerprint(raw)
	store, err := held.cfg.DerivedStores().Open(v)
	if err != nil {
		t.Fatal(err)
	}
	cues := spoken(words)
	if err := store.Write(t.Context(), text.Artifact(text.ASR, hash), transcript.Marshal(cues)); err != nil {
		t.Fatal(err)
	}
	if through == 0 {
		return held, v, store, hash
	}

	// The shelf as a run that ended among the batches left it: the lines it put
	// right, and the count standing after them.
	for at := range through {
		cues[at].Text = strings.ToUpper(cues[at].Text)
	}
	if err := store.Write(t.Context(), text.Corrected(text.ASR, hash), transcript.Marshal(cues)); err != nil {
		t.Fatal(err)
	}
	stood, err := json.Marshal(putting{By: by.Name(), At: cues[through-1].To})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Write(t.Context(), text.Proofread(text.ASR, hash), stood); err != nil {
		t.Fatal(err)
	}
	return held, v, store, hash
}

const recording = "talks/one.mp3"

// said is what the transcript on the shelf says, line by line.
func says(t *testing.T, store port.DerivedStore, name string) []string {
	t.Helper()
	raw, err := store.Read(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	_, cues := transcript.Parse(raw)
	out := make([]string, 0, len(cues))
	for _, cue := range cues {
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

	held.TakingUp(t.Context(), heard{recording}, v)
	held.Wait()

	if got := by.lines(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	want := []string{"FIRST THING", "SECOND THING", "THIRD THING"}
	if got := says(t, store, text.Corrected(text.ASR, hash)); !slices.Equal(got, want) {
		t.Errorf("the transcript says %q", got)
	}
	if doing, failed := said(held, recording); doing != "" {
		t.Errorf("the transcript is left in the list as %q, failing with %q", doing, failed)
	}
}

// A transcript put right to its last line is asked about nothing and shown
// nowhere, however often the application opens.
func TestATranscriptAlreadyPutRightIsAskedAboutNothing(t *testing.T) {
	by := &puts{}
	held, v, _, _ := stopped(t, by, 3, "first thing", "second thing", "third thing")

	held.TakingUp(t.Context(), heard{recording}, v)
	held.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if doing, failed := said(held, recording); doing != "" {
		t.Errorf("the transcript is in the list as %q, failing with %q", doing, failed)
	}
}

// An installation that did not ask for its transcripts to be put right takes up
// nothing.
func TestNoTranscriptIsTakenUpWhereItWasNotAskedFor(t *testing.T) {
	by := &puts{says: map[int]string{1: "1|SECOND THING"}}
	held, v, _, _ := stopped(t, by, 1, "first thing", "second thing")
	held.cfg.SpeechProofreading.Automatically = false

	held.TakingUp(t.Context(), heard{recording}, v)
	held.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// A transcript a person asked for is put right with the profile the settings
// name for speech. The profile named for a scanned reading is another setting
// and is not read here.
func TestATranscriptAskedForIsPutRightWithTheProfileNamedForSpeech(t *testing.T) {
	speech := &puts{says: map[int]string{0: "0|FIRST THING", 1: "1|SECOND THING"}}
	scans := &puts{says: map[int]string{0: "0|off a page", 1: "1|off a page"}}
	held, v, store, hash := stopped(t, speech, 0, "first thing", "second thing")
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "by hand"}
	held.cfg.ScanProofreading = proofreading.Proofread{With: "off a page", Automatically: true}
	held.cfg.Proofreading.Profiles["off a page"] = proofreading.Profile{
		Use: proofreading.UseAgent, Model: "another model", BatchSize: 1, InFlight: 1,
	}
	held.cfg.AgentProofreader = func(said AgentProofreading) (port.Proofreader, error) {
		if said.Model == "another model" {
			return scans, nil
		}
		return speech, nil
	}

	_, _ = held.Proofread(t.Context(), v, recording)
	held.Wait()

	if got := scans.lines(); len(got) != 0 {
		t.Errorf("the reading's proofreader was asked about lines %v", got)
	}
	if got := speech.lines(); len(got) == 0 {
		t.Error("the speech proofreader was asked about nothing")
	}
	want := []string{"FIRST THING", "SECOND THING"}
	if got := says(t, store, text.Corrected(text.ASR, hash)); !slices.Equal(got, want) {
		t.Errorf("the transcript says %q", got)
	}
}

// An installation that names something to put a transcript right with offers
// the run, and one that names nothing does not.
func TestAProofreadingIsOfferedWhereAProfileIsNamed(t *testing.T) {
	held, _ := listens(t, &deaf{}, recording)

	if held.ProofreaderReady() {
		t.Error("an installation naming no profile offers the run")
	}
	held.cfg.SpeechProofreading = proofreading.Proofread{With: "by hand"}
	if !held.ProofreaderReady() {
		t.Error("an installation naming a profile does not offer the run")
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
	held.tasks.Set(task.Task{
		ID: correcting(recording), Doing: "Proofreading a transcript", About: recording,
		Done: 1, Total: 2,
	})

	held.TakingUp(t.Context(), heard{recording}, v)
	held.Wait()

	if doing, _ := said(held, recording); doing != "Proofreading a transcript" {
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
	held.cfg.AgentProofreader = func(AgentProofreading) (port.Proofreader, error) {
		close(reached)
		<-stand
		return by, nil
	}

	go func() { _, _ = held.Proofread(t.Context(), v, recording) }()
	<-reached
	doing, _ := said(held, recording)
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

	if doing, failed := said(held, recording); doing != "" {
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

	doing, failed := said(held, recording)
	if doing != "Proofreading a transcript" {
		t.Fatalf("a run that failed is in the list as %q", doing)
	}
	if !strings.Contains(failed, "not on this machine") {
		t.Errorf("the failure says %q", failed)
	}
}
