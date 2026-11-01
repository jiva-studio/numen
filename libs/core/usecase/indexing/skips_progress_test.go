package indexing

import (
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// progressLog is a recorder that also keeps every report of the vector pass.
type progressLog struct {
	*recorder

	mu      sync.Mutex
	reports []run.Task
}

func (l *progressLog) ReportTask(pos run.Task) {
	if pos.ID == MakingVectors {
		l.mu.Lock()
		l.reports = append(l.reports, pos)
		l.mu.Unlock()
	}
	l.recorder.ReportTask(pos)
}

// getReports returns every report of the vector pass so far.
func (l *progressLog) getReports() []run.Task {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]run.Task(nil), l.reports...)
}

// getCounted returns the reports of the vector pass that carry a total.
func (l *progressLog) getCounted() []run.Task {
	l.mu.Lock()
	defer l.mu.Unlock()
	var counted []run.Task
	for _, pos := range l.reports {
		if pos.Total > 0 {
			counted = append(counted, pos)
		}
	}
	return counted
}

// cancelledPass is a pass over a vault whose note Held.md had its vector pass cancelled by a person,
// and the number of chunks of that note.
func cancelledPass(t *testing.T, events *progressLog, log *readyLog) (*pass, *holdingModel, int64) {
	t.Helper()
	model := &holdingModel{fakeEmbedder: &fakeEmbedder{dims: 64}, started: make(chan struct{}, 1), isBlocking: true}
	made, db := reading(t)
	fit(t, db, model)
	recipe := model.GetModel().Recipe()

	v := testsupport.NewVault(t, map[string]string{"Held.md": noteWith(getHeldWords(), 200)})
	cut(t, made, db, v)
	heldChunks, _, err := db.Progress().GetProgress(t.Context(), v.ID, recipe)
	if err != nil {
		t.Fatal(err)
	}

	p := newPass(made, db, events, model, recipe, v)
	p.worker.vectorsReady = log.report
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		p.embed(t.Context())
	}()
	select {
	case <-model.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass never asked for the held note's vectors")
	}
	if !events.tasks.Cancel(MakingVectors) {
		t.Fatal("the vector pass is not cancellable")
	}
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass did not return after its task was cancelled")
	}
	model.setBlocking(false)

	writeNote(t, v.Path, "Other.md", noteWith(getBefore(), 200))
	cut(t, made, db, v)
	return p, model, heldChunks
}

// The pass after a cancel counts its progress against the chunks it embeds, and the passes after it
// find nothing pending while only the cancelled note lacks vectors.
func TestProgressOfAPassAfterACancelReachesItsTotal(t *testing.T) {
	events := &progressLog{recorder: newRecorder()}
	p, _, _ := cancelledPass(t, events, &readyLog{})
	earlier := len(events.getCounted())

	p.embed(t.Context())

	counted := events.getCounted()[earlier:]
	if len(counted) == 0 {
		t.Fatal("the pass after the cancel reported no counted progress")
	}
	if last := counted[len(counted)-1]; last.Count != last.Total {
		t.Errorf("the pass ended at %d of %d", last.Count, last.Total)
	}
	if pos := findTask(t, events.recorder, MakingVectors); pos != nil {
		t.Errorf("the pass is still listed: %+v", *pos)
	}

	reports := len(events.getReports())
	p.embed(t.Context())
	if later := events.getReports(); len(later) != reports {
		t.Errorf("a pass with only the cancelled note left reported %+v", later[reports:])
	}
	if pos := findTask(t, events.recorder, MakingVectors); pos != nil {
		t.Errorf("a pass with only the cancelled note left is listed: %+v", *pos)
	}
}

// The vectors-ready listener is told once the notes nobody cancelled carry vectors, and the
// cancelled note still has none: the event covers the vault without it.
func TestVectorsReadyExcludesTheCancelledNote(t *testing.T) {
	events := &progressLog{recorder: newRecorder()}
	log := &readyLog{}
	p, model, heldChunks := cancelledPass(t, events, log)
	if len(log.vaults) != 0 {
		t.Fatalf("told %v after the cancelled pass", log.vaults)
	}

	p.embed(t.Context())
	p.embed(t.Context())

	if len(log.vaults) != 2 {
		t.Errorf("told %d times, want once for each pass after the cancel", len(log.vaults))
	}
	if calls := model.getHeldCalls(); calls != 0 {
		t.Errorf("the cancelled note was sent to the model %d times", calls)
	}
	target, embedded, err := p.worker.vectorCounts(t.Context(), p.v)
	if err != nil {
		t.Fatal(err)
	}
	if target-embedded != heldChunks {
		t.Errorf("%d of %d chunks lack a vector, want the %d of the cancelled note", target-embedded, target, heldChunks)
	}

	p.worker.ClearSkips([]string{"Held.md"})
	p.embed(t.Context())
	if target, embedded, err = p.worker.vectorCounts(t.Context(), p.v); err != nil {
		t.Fatal(err)
	}
	if embedded != target {
		t.Errorf("after the cancel was cleared %d of %d chunks carry a vector", embedded, target)
	}
}

// The count of skipped chunks is the one of the list it was counted for: a change of the list forgets
// it, and a pass that started before the change records nothing.
func TestSkippedChunksAreForgottenWhenTheListChanges(t *testing.T) {
	skips := newSkipSet()
	skips.addVectors("Book.pdf")
	generation := skips.getGeneration()
	skips.setSkippedChunks(12, generation)
	if got := skips.getSkippedChunks(); got != 12 {
		t.Fatalf("skipped chunks = %d, want 12", got)
	}

	skips.clear([]string{"Book.pdf"})
	if got := skips.getSkippedChunks(); got != 0 {
		t.Errorf("skipped chunks after a clear = %d, want 0", got)
	}
	skips.setSkippedChunks(12, generation)
	if got := skips.getSkippedChunks(); got != 0 {
		t.Errorf("a pass that started before the clear recorded %d skipped chunks", got)
	}
}
