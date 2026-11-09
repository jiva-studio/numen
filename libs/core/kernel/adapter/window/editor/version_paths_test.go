package editor

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

func TestAChangeOfTheVersionsOfAFileReachesTheWindow(t *testing.T) {
	state := &State{ChangeEvents: newChangeBroadcaster()}
	line, stop := state.ChangeEvents.subscribe()
	defer stop()
	index := indexing.NewWorker(indexing.Config{}).NewSession()

	newVersionChangeReporter(state, index).ReportVersionsChanged(domain.Vault{ID: "v1"}, "scans/page.png")

	got := newChangeResponse(<-line)
	if !slices.Equal(got.GetVersionPaths(), []string{"scans/page.png"}) || len(got.GetPaths()) != 0 {
		t.Errorf("the window received %v", got)
	}
}

func TestAChangedFileIsCheckedForAChangeOfItsLiveVersion(t *testing.T) {
	state := &State{ChangeEvents: newChangeBroadcaster()}
	var checked []string
	report := newChangeReporter(state, indexing.NewWorker(indexing.Config{}), newTriggerPasses(), func(paths []string) { checked = append(checked, paths...) })

	report([]string{"links/talk.md"}, []string{"scans/page.png"}, false)

	if want := []string{"links/talk.md", "scans/page.png"}; !slices.Equal(checked, want) {
		t.Errorf("checked %v, want %v", checked, want)
	}
}
