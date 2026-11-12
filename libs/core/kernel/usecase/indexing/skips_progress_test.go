package indexing

import (
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

// progressLog is a recorder that also keeps every report of the vector pass.
type progressLog struct {
	*recorder

	mu      sync.Mutex
	reports []work.Task
}

func (l *progressLog) ReportTask(pos work.Task) {
	if pos.ID == work.MakingVectors {
		l.mu.Lock()
		l.reports = append(l.reports, pos)
		l.mu.Unlock()
	}
	l.recorder.ReportTask(pos)
}

func (l *progressLog) getReports() []work.Task {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]work.Task(nil), l.reports...)
}

func (l *progressLog) getCountedTasks() []work.Task {
	l.mu.Lock()
	defer l.mu.Unlock()
	var counted []work.Task
	for _, pos := range l.reports {
		if pos.Total > 0 {
			counted = append(counted, pos)
		}
	}
	return counted
}

func canceledPass(t *testing.T, openReading openFunc, events *progressLog, log *readyLog) (*testPass, *holdingModel, int64) {
	t.Helper()
	model := &holdingModel{fakeEmbedder: &fakeEmbedder{dims: 64}, started: make(chan struct{}, 1), isBlocking: true}
	made, db := openReading(t)
	fit(t, db, model)
	recipe := model.GetModel().Recipe()

	v := made.newVault(t, map[string]string{"Held.md": noteWith(getHeldWords(), 200)})
	cut(t, made, db, v)
	heldChunks, _, err := db.Progress.GetProgress(t.Context(), v.ID, recipe)
	if err != nil {
		t.Fatal(err)
	}

	p := newPass(made, db, events, model, recipe, v)
	p.worker.searchSourceReady = log.report
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
	if !events.tasks.Cancel(work.MakingVectors) {
		t.Fatal("the vector pass is not cancellable")
	}
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass did not return after its task was canceled")
	}
	model.setBlocking(false)

	writeNote(t, v.Path, "Other.md", noteWith(getBeforeNames(), 200))
	cut(t, made, db, v)
	return p, model, heldChunks
}

func testProgressOfAPassAfterACancelReachesItsTotal(t *testing.T, openReading openFunc) {
	t.Helper()
	events := &progressLog{recorder: newRecorder()}
	p, _, _ := canceledPass(t, openReading, events, &readyLog{})
	earlier := len(events.getCountedTasks())

	p.embed(t.Context())

	counted := events.getCountedTasks()[earlier:]
	if len(counted) == 0 {
		t.Fatal("the pass after the cancel reported no counted progress")
	}
	if last := counted[len(counted)-1]; last.Count != last.Total {
		t.Errorf("the pass ended at %d of %d", last.Count, last.Total)
	}
	if pos := findTask(t, events.recorder, work.MakingVectors); pos != nil {
		t.Errorf("the pass is still listed: %+v", *pos)
	}

	reports := len(events.getReports())
	p.embed(t.Context())
	if later := events.getReports(); len(later) != reports {
		t.Errorf("a pass with only the canceled note left reported %+v", later[reports:])
	}
	if pos := findTask(t, events.recorder, work.MakingVectors); pos != nil {
		t.Errorf("a pass with only the canceled note left is listed: %+v", *pos)
	}
}

func testSearchSourceReadyExcludesTheCanceledNote(t *testing.T, openReading openFunc) {
	t.Helper()
	events := &progressLog{recorder: newRecorder()}
	log := &readyLog{}
	p, model, heldChunks := canceledPass(t, openReading, events, log)
	if len(log.vaults) != 0 {
		t.Fatalf("told %v after the canceled pass", log.vaults)
	}

	p.embed(t.Context())
	p.embed(t.Context())

	if len(log.vaults) != 2 {
		t.Errorf("told %d times, want once for each pass after the cancel", len(log.vaults))
	}
	if calls := model.getHeldCalls(); calls != 0 {
		t.Errorf("the canceled note was sent to the model %d times", calls)
	}
	target, embedded, err := p.worker.vectorCounts(t.Context(), p.v)
	if err != nil {
		t.Fatal(err)
	}
	if target-embedded != heldChunks {
		t.Errorf("%d of %d chunks lack a vector, want the %d of the canceled note", target-embedded, target, heldChunks)
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
