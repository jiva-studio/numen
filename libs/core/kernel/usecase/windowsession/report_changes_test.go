package windowsession

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestAFileEventAsksForAPassOverTheChangedPathsOnly(t *testing.T) {
	passes := newTriggerPasses()
	passes.take()
	w := newTestWindow(t, nil)
	report := (&PassRunner{changes: w.changes}).newChangeReporter(passes, func([]string) {})

	report([]string{"links/talk.md"}, []string{"scans/page.png"}, false)
	report([]string{"links/talk.md"}, nil, false)

	isAll, paths := passes.take()
	if isAll || !slices.Equal(paths, []string{"links/talk.md", "scans/page.png"}) {
		t.Errorf("asked for all %v over %v, want the two paths alone", isAll, paths)
	}
	if isAll, paths := passes.take(); isAll || len(paths) != 0 {
		t.Errorf("a pass taken is asked for again: all %v over %v", isAll, paths)
	}
}

func TestAReloadAsksForAPassOverTheWholeVaultAndWakesTheSources(t *testing.T) {
	w := newTestWindow(t, nil)
	passes := newTriggerPasses()
	passes.take()
	report := (&PassRunner{changes: w.changes, wake: w.wake}).newChangeReporter(passes, func([]string) {})

	report(nil, nil, true)

	if isAll, _ := passes.take(); !isAll {
		t.Error("a reload asked for no pass over the whole vault")
	}
	if _, sources := w.wake.counts(); sources != 1 {
		t.Errorf("the sources were woken %d times, want once", sources)
	}
}

func TestAChangedNoteWakesTheVectorsAndAnAssetWakesTheSources(t *testing.T) {
	w := newTestWindow(t, nil)
	report := (&PassRunner{changes: w.changes, wake: w.wake}).newChangeReporter(newTriggerPasses(), func([]string) {})

	report([]string{"links/talk.md"}, nil, false)
	report(nil, []string{"books/tide.epub"}, false)

	if notes, sources := w.wake.counts(); notes != 1 || sources != 1 {
		t.Errorf("woke the notes %d times and the sources %d times, want once each", notes, sources)
	}
	if len(w.wake.skipped) != 2 {
		t.Errorf("the canceled sources were cleared %d times, want for each change", len(w.wake.skipped))
	}
}

func TestAChangeIsShownToTheViewsAsEveryPathThatChanged(t *testing.T) {
	w := newTestWindow(t, nil)
	report := (&PassRunner{changes: w.changes}).newChangeReporter(newTriggerPasses(), func([]string) {})

	report([]string{"links/talk.md"}, []string{"scans/page.png"}, false)

	got := w.changes.all()
	if len(got) != 1 || !slices.Equal(got[0].Paths, []string{"links/talk.md", "scans/page.png"}) {
		t.Errorf("the views were told %v", got)
	}
}

func TestAChangedFileIsCheckedForAChangeOfItsLiveVersion(t *testing.T) {
	w := newTestWindow(t, nil)
	var checked []string
	report := (&PassRunner{changes: w.changes}).newChangeReporter(newTriggerPasses(), func(paths []string) { checked = append(checked, paths...) })

	report([]string{"links/talk.md"}, []string{"scans/page.png"}, false)

	if want := []string{"links/talk.md", "scans/page.png"}; !slices.Equal(checked, want) {
		t.Errorf("checked %v, want %v", checked, want)
	}
}

func TestAChangeOfTheVersionsOfAFileReachesTheWindow(t *testing.T) {
	w := newTestWindow(t, nil)
	var cut []string
	index := IndexSession{ReportVersionsChanged: func(_ domain.Vault, path string) { cut = append(cut, path) }}

	versionChangeReporter{changes: w.changes, index: index}.ReportVersionsChanged(domain.Vault{ID: "v1"}, "scans/page.png")

	got := w.changes.all()
	if len(got) != 1 || !slices.Equal(got[0].VersionPaths, []string{"scans/page.png"}) || len(got[0].Paths) != 0 {
		t.Errorf("the window received %v", got)
	}
	if !slices.Equal(cut, []string{"scans/page.png"}) {
		t.Errorf("the index cut %v, want the file whose versions changed", cut)
	}
}

// listing is a plugin host that declares no trigger and counts how often the triggers are read.
type listing struct {
	mu    sync.Mutex
	reads int
}

func (l *listing) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reads
}

func (l *listing) ExecuteAll(context.Context, domain.Vault) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reads++
	return nil
}

func (l *listing) Execute(_ context.Context, _ domain.Vault, _ []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reads++
	return nil
}

func waitFor(t *testing.T, is func() bool) {
	t.Helper()
	for range 500 {
		if is() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the condition never held")
}

func runPasses(t *testing.T) (passes *triggerPasses, passesRun func() int, changes chan<- struct{}) {
	t.Helper()
	runner := &listing{}
	line := make(chan struct{})
	passes = newTriggerPasses()
	go runTriggers(t.Context(), runner, domain.Vault{ID: "v1"}, passes, line, func(error) {})
	return passes, runner.count, line
}

func TestTheFirstPassRunsOverTheWholeVaultAndAnotherWaitsForARequest(t *testing.T) {
	passes, passesRun, _ := runPasses(t)

	waitFor(t, func() bool { return passesRun() == 1 })
	time.Sleep(20 * time.Millisecond)
	if got := passesRun(); got != 1 {
		t.Fatalf("%d passes ran with nothing asked for, want 1", got)
	}

	passes.requestPaths([]string{"links/a.md"})
	waitFor(t, func() bool { return passesRun() == 2 })
}

func TestAChangeOfThePluginsAsksForAPassOverTheWholeVault(t *testing.T) {
	_, passesRun, changes := runPasses(t)
	waitFor(t, func() bool { return passesRun() == 1 })

	changes <- struct{}{}

	waitFor(t, func() bool { return passesRun() == 2 })
}

func TestANewVaultStartsWithAPassOverTheWholeVault(t *testing.T) {
	if isAll, _ := newTriggerPasses().take(); !isAll {
		t.Error("the passes of a vault opened start with none over the whole vault")
	}
}
