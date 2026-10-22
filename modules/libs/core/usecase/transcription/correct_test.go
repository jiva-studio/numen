package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// getTask is what the list of what is being done says about putting a
// transcript right.
func getTask(held *Worker, path string) (doing, failed string) {
	for _, at := range held.with.Tasks.List() {
		if at.ID == run.GetProofreadID(path) {
			return at.Title, at.Error
		}
	}
	return "", ""
}

// openNoProofreader is a profile that cannot be opened, as a person naming one no
// settings name is told.
func openNoProofreader() (port.Proofreader, error) {
	return nil, errors.New(`no proofreading profile named "nowhere"`)
}

// An installation that did not ask for its transcripts to be put right asks
// nothing and says nothing.
func TestATranscriptIsInsertRightOnlyWhereItWasAskedFor(t *testing.T) {
	target, v := newTranscriptionWorker(t, &deaf{}, "talks/one.mp3")
	target.with.Proofreading = run.ProofreadingConfig{IsNamed: true, By: openNoProofreader}

	target.proofread(t.Context(), v, "talks/one.mp3", "", false)

	if executing, _ := getTask(target, "talks/one.mp3"); executing != "" {
		t.Errorf("a transcript unrequested about is %q", executing)
	}
}

// A profile no settings name is a person's mistake, and they are shown it.
func TestAProfileNoSettingsNameIsShown(t *testing.T) {
	target, v := newTranscriptionWorker(t, &deaf{}, "talks/one.mp3")
	target.with.Proofreading = run.ProofreadingConfig{IsNamed: true, IsAutomatic: true, By: openNoProofreader}

	target.proofread(t.Context(), v, "talks/one.mp3", "", false)

	executing, failed := getTask(target, "talks/one.mp3")
	if executing != "Proofreading a transcript" {
		t.Fatalf("the list says %q", executing)
	}
	if !strings.Contains(failed, "nowhere") {
		t.Errorf("the failure does not name the profile: %q", failed)
	}
}

// Silence carries no words, so nothing is asked about it.
func TestSilenceIsNotInsertRight(t *testing.T) {
	by := &deaf{}
	target, v := newTranscriptionWorker(t, by, "talks/one.mp3")
	target.with.Proofreading = run.ProofreadingConfig{IsNamed: true, IsAutomatic: true, By: openNoProofreader}

	target.Start(v, "talks/one.mp3")
	target.Wait()

	if executing, failed := getTask(target, "talks/one.mp3"); executing != "" {
		t.Errorf("silence is %q, failing with %q", executing, failed)
	}
}

// cues is the words as the cues a model wrote them down as, one second apart.
func cues(words []string) []domain.TranscriptCue {
	out := make([]domain.TranscriptCue, 0, len(words))
	for at, response := range words {
		out = append(out, domain.TranscriptCue{Text: response, From: at * 1000, To: at*1000 + 800})
	}
	return out
}

// newStoppedWorker is a Worker over a vault holding one recording,
// with its transcript on the shelf and a proofreading of it standing at through
// lines.
//
// One line to a request and one request to a call, so a call answers about one
// line.
func newStoppedWorker(
	t *testing.T,
	by port.Proofreader,
	through int,
	words ...string,
) (*Worker, domain.Vault, port.DerivedStore, string) {
	t.Helper()
	target, v := newTranscriptionWorker(t, &deaf{}, recording)
	target.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, IsAutomatic: true,
		By: func() (port.Proofreader, error) { return by, nil },
	}

	reader, err := target.with.Readers.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := reader.Read(t.Context(), recording)
	if err != nil {
		t.Fatal(err)
	}
	hash := text.Fingerprint(raw)
	store, err := target.with.Derived.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	cuesList := cues(words)
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, hash), transcript.Marshal(cuesList)); err != nil {
		t.Fatal(err)
	}
	if through == 0 {
		return target, v, store, hash
	}

	// The shelf as a run that ended among the batches left it: the lines it put
	// right, and the count standing after them.
	for at := range through {
		cuesList[at].Text = strings.ToUpper(cuesList[at].Text)
	}
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash), transcript.Marshal(cuesList)); err != nil {
		t.Fatal(err)
	}
	status, err := json.Marshal(putting{By: by.GetName(), At: cuesList[through-1].To})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, hash), status); err != nil {
		t.Fatal(err)
	}
	return target, v, store, hash
}

const recording = "talks/one.mp3"

// readLines is what the transcript on the shelf says, line by line.
func readLines(t *testing.T, store port.DerivedStore, name domain.ArtifactName) []string {
	t.Helper()
	raw, err := store.ReadArtifact(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	_, response := transcript.Parse(raw)
	out := make([]string, 0, len(response))
	for _, cue := range response {
		out = append(out, cue.Text)
	}
	return out
}

// and the lines after it are asked about when the application opens.
func TestATranscriptLeftPartWayThroughIsTakenUpWhenTheApplicationOpens(t *testing.T) {
	by := &corrector{says: says(fixes(1, "SECOND THING"), fixes(2, "THIRD THING"))}
	target, v, store, hash := newStoppedWorker(t, by, 1, "first thing", "second thing", "third thing")

	target.TakeUp(t.Context(), recognised{recording}, v)
	target.Wait()

	if got := by.lines(); !slices.Equal(got, []int{1, 2}) {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	want := []string{"FIRST THING", "SECOND THING", "THIRD THING"}
	if got := readLines(t, store, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, hash)); !slices.Equal(got, want) {
		t.Errorf("the transcript says %q", got)
	}
	if executing, failed := getTask(target, recording); executing != "" {
		t.Errorf("the transcript is left in the list as %q, failing with %q", executing, failed)
	}
}

// A transcript put right to its last line is asked about nothing and shown
// nowhere, however often the application opens.
func TestATranscriptAlreadyInsertRightIsAskedAboutNothing(t *testing.T) {
	by := &corrector{}
	target, v, _, _ := newStoppedWorker(t, by, 3, "first thing", "second thing", "third thing")

	target.TakeUp(t.Context(), recognised{recording}, v)
	target.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
	if executing, failed := getTask(target, recording); executing != "" {
		t.Errorf("the transcript is in the list as %q, failing with %q", executing, failed)
	}
}

// An installation that did not ask for its transcripts to be put right takes up
// nothing.
func TestNoTranscriptIsTakenUpWhereItWasNotAskedFor(t *testing.T) {
	by := &corrector{says: says(fixes(1, "SECOND THING"))}
	target, v, _, _ := newStoppedWorker(t, by, 1, "first thing", "second thing")
	target.with.Proofreading.IsAutomatic = false

	target.TakeUp(t.Context(), recognised{recording}, v)
	target.Wait()

	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// One run to a transcript. A run that finds it held leaves the list to the run
// that holds it.
func TestATranscriptAnotherRunContainsKeepsItsPlaceInTheList(t *testing.T) {
	by := &corrector{}
	target, v, store, hash := newStoppedWorker(t, by, 1, "first thing", "second thing")
	release, err := store.ClaimArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, hash))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	target.with.Tasks.Set(run.Task{
		ID: run.GetProofreadID(recording), Title: "Proofreading a transcript", About: recording,
		Count: 1, Total: 2,
	})

	target.TakeUp(t.Context(), recognised{recording}, v)
	target.Wait()

	if executing, _ := getTask(target, recording); executing != "Proofreading a transcript" {
		t.Errorf("the run holding the transcript is in the list as %q", executing)
	}
	if got := by.lines(); len(got) != 0 {
		t.Errorf("the proofreader was asked about lines %v", got)
	}
}

// refuses is a proofreader that will not answer about anything.
type refuses struct{ why error }

func (r *refuses) GetName() string { return "a proofreader that will not answer" }

func (r *refuses) ProofreadScan(context.Context, []domain.ProofreadPage) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, r.why
}

func (r *refuses) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	return domain.SpeechProofreading{}, r.why
}

// A transcript this proofreader has been over every line of is asked about
// nothing, and whoever asked is told that and not nothing at all.
func TestATranscriptAlreadyInsertRightReturnsThatNothingWasLeft(t *testing.T) {
	by := &corrector{}
	target, v, _, _ := newStoppedWorker(t, by, 2, "first thing", "second thing")

	res, err := target.Proofread(t.Context(), v, recording)
	target.Wait()

	if err != nil {
		t.Fatalf("putting a transcript right again failed: %v", err)
	}
	if !res.IsAlready {
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
	target, v, _, _ := newStoppedWorker(t, by, 0, "first thing", "second thing")

	res, err := target.Proofread(t.Context(), v, recording)

	if err != nil {
		t.Fatalf("a run that began answered %v", err)
	}
	if res.IsAlready || res.IsNone || res.IsEdited || res.IsBusy {
		t.Errorf("a run with work to do answered %+v", res)
	}
	<-by.asked
	close(stand)
	target.Wait()
}

// waits is a proofreader that says it was asked and then waits to be let go.
type waits struct {
	on    chan struct{}
	asked chan struct{}
	once  sync.Once
}

func (w *waits) GetName() string { return "a proofreader that waits" }

func (w *waits) ProofreadScan(context.Context, []domain.ProofreadPage) (domain.ScanProofreading, error) {
	return domain.ScanProofreading{}, nil
}

func (w *waits) ProofreadSpeech(
	context.Context, []domain.ProofreadLine, int,
) (domain.SpeechProofreading, error) {
	w.once.Do(func() { close(w.asked) })
	<-w.on
	return domain.SpeechProofreading{IsDone: true}, nil
}

// not from the moment it has something to count. Opening the proofreader is the
// first thing it does, and the list already says so there.
func TestAProofreadingAskedForExistsInTheListBeforeItOpensTheProofreader(t *testing.T) {
	by := &corrector{}
	target, v, _, _ := newStoppedWorker(t, by, 0, "first thing", "second thing")
	reached, stand := make(chan struct{}), make(chan struct{})
	target.with.Proofreading.By = func() (port.Proofreader, error) {
		close(reached)
		<-stand
		return by, nil
	}

	go func() { _, _ = target.Proofread(t.Context(), v, recording) }()
	<-reached
	executing, _ := getTask(target, recording)
	close(stand)
	target.Wait()

	if executing != "Proofreading a transcript" {
		t.Errorf("a run a person asked for is in the list as %q", executing)
	}
}

// A run that ends leaves nothing behind in the list.
func TestAProofreadingThatEndsPreservesTheListEmpty(t *testing.T) {
	by := &corrector{says: says(fixes(0, "FIRST THING"))}
	target, v, _, _ := newStoppedWorker(t, by, 0, "first thing")

	_, _ = target.Proofread(t.Context(), v, recording)
	target.Wait()

	if executing, failed := getTask(target, recording); executing != "" {
		t.Errorf("a run that ended is in the list as %q, failing with %q", executing, failed)
	}
}

// A proofreader that will not answer is a failure a person is shown, and the
// failure stays in the list.
func TestAProofreaderThatWillNotAnswerIsShownAsAFailure(t *testing.T) {
	by := &refuses{why: errors.New("the command line is not on this machine")}
	target, v, _, _ := newStoppedWorker(t, by, 0, "first thing", "second thing")

	_, _ = target.Proofread(t.Context(), v, recording)
	target.Wait()

	executing, failed := getTask(target, recording)
	if executing != "Proofreading a transcript" {
		t.Fatalf("a run that failed is in the list as %q", executing)
	}
	if !strings.Contains(failed, "not on this machine") {
		t.Errorf("the failure says %q", failed)
	}
}
