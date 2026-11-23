package refresh_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/refresh"
)

const taskID = "saved-search-refresh"

func addGalesNote(w *world) {
	w.vault.WriteFile(galesPath, formatSearchNote(galesID, galesQuery))
}

func TestRunDueDeletesTheCacheOfARemovedSearch(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)
	w.runDueSearches(t)
	if !w.hasRecord(t, galesID+".json") {
		t.Fatal("no cache for the gales")
	}

	w.vault.Remove(galesPath)
	w.runDueSearches(t)

	if w.hasRecord(t, galesID+".json") || !w.hasRecord(t, stormsID+".json") {
		t.Error("want the cache of the removed search deleted and the other kept")
	}
}

func TestRunDueDeletesTheCacheOfANoteThatIsNoLongerASearch(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)
	w.runDueSearches(t)
	if !w.hasRecord(t, galesID+".json") {
		t.Fatal("no cache for the gales")
	}

	w.vault.WriteFile(galesPath, "---\nid: "+galesID+"\n---\nThe gales of 1911.\n")
	w.runDueSearches(t)

	if w.hasRecord(t, galesID+".json") {
		t.Error("want the cache of a note that is no longer a saved search deleted")
	}
}

func TestRunDueReportsATaskAndFinishesIt(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)

	w.runDueSearches(t)

	reports := w.tasks.getReports()
	if len(reports) == 0 {
		t.Fatal("no task reported")
	}
	first, last := reports[0], reports[len(reports)-1]
	if first.ID != taskID || first.Title != "Refreshing saved searches" || first.TotalCount != 2 || first.CompletedCount != 0 {
		t.Errorf("first report = %+v, want the refresh of two searches", first)
	}
	if last.CompletedCount != 2 || last.FailureReason != "" {
		t.Errorf("last report = %+v, want both completed", last)
	}
	if got := w.tasks.getFinishedIDs(); len(got) != 1 || got[0] != taskID {
		t.Errorf("finished = %v, want the refresh task once", got)
	}
}

func TestRunDueReportsNoTaskWhenNothingIsDue(t *testing.T) {
	w := newWorld(t, 0)
	w.runDueSearches(t)
	reports := len(w.tasks.getReports())
	if reports == 0 {
		t.Fatal("no task reported for the first run")
	}

	w.runDueSearches(t)

	if got := len(w.tasks.getReports()); got != reports {
		t.Errorf("%d reports with nothing due, want none", got-reports)
	}
}

func TestOneFailingSearchDoesNotStopTheOthers(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)
	w.vault.Refuse(galesPath, errors.New("the disk is busy"))

	if err := w.runner.RunDue(t.Context(), testVault); err != nil {
		t.Logf("RunDue = %v", err)
	}

	if record := w.readRecord(t, stormsID); len(record.Results) != 2 {
		t.Errorf("record = %+v, want the storms computed after the gales failed", record)
	}
	reports := w.tasks.getReports()
	if len(reports) == 0 || reports[len(reports)-1].FailureReason == "" {
		t.Errorf("reports = %+v, want the failure in the task", reports)
	}
	if got := w.tasks.getFinishedIDs(); len(got) != 1 {
		t.Errorf("finished = %v, want the task finished", got)
	}
}

func TestRunsOfOneVaultDoNotOverlap(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)
	both := make(chan struct{})
	var once sync.Once
	w.host.setHook(func() {
		if w.host.getActiveCount() > 1 {
			once.Do(func() { close(both) })
		}
		select {
		case <-both:
		case <-time.After(200 * time.Millisecond):
		}
	})

	var group sync.WaitGroup
	for _, notePath := range []string{stormsPath, galesPath} {
		group.Go(func() {
			if _, err := w.runner.RunNow(t.Context(), testVault, notePath); err != nil {
				t.Errorf("RunNow(%s) = %v", notePath, err)
			}
		})
	}
	group.Go(func() {
		if err := w.runner.RunDue(t.Context(), testVault); err != nil {
			t.Errorf("RunDue = %v", err)
		}
	})
	group.Wait()

	if got := w.host.getMaxActive(); got != 1 {
		t.Errorf("%d searches of one vault ran at once, want 1", got)
	}
}

func TestWatchReportsTheStartAndTheWriteOfARun(t *testing.T) {
	w := newWorld(t, 0)
	changes, stop := w.runner.Watch(testVault, stormsPath)
	defer stop()

	done := make(chan error, 1)
	go func() {
		_, err := w.runner.RunNow(t.Context(), testVault, stormsPath)
		done <- err
	}()

	start := receiveChange(t, changes)
	write := receiveChange(t, changes)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !start.IsRunning {
		t.Errorf("first change = %+v, want the start of the run", start)
	}
	if write.IsRunning || !write.ComputedAt.Equal(getStartTime()) {
		t.Errorf("second change = %+v, want the write computed now", write)
	}
}

func TestWatchOfAnotherNoteReceivesNothing(t *testing.T) {
	w := newWorld(t, 0)
	addGalesNote(w)
	changes, stop := w.runner.Watch(testVault, galesPath)
	defer stop()

	w.runNow(t, stormsPath)

	select {
	case change := <-changes:
		t.Errorf("change = %+v for a run of another note, want none", change)
	default:
	}
}

func TestWatchReceivesNothingAfterItStops(t *testing.T) {
	w := newWorld(t, 0)
	changes, stop := w.runner.Watch(testVault, stormsPath)

	stop()
	w.runNow(t, stormsPath)

	select {
	case change, isOpen := <-changes:
		if isOpen {
			t.Errorf("change = %+v after the watch stopped, want none", change)
		}
	default:
	}
}

func receiveChange(t *testing.T, changes <-chan refresh.Change) refresh.Change {
	t.Helper()
	select {
	case change := <-changes:
		return change
	case <-time.After(5 * time.Second):
		t.Fatal("no change within 5 seconds")
		return refresh.Change{}
	}
}
