package editor

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/triggerpasses"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
)

func TestAFileEventAsksForAPassOverTheChangedPathsOnly(t *testing.T) {
	passes := triggerpasses.NewPasses()
	passes.Take()
	report := newChangeReporter(&State{ChangeEvents: newChangeBroadcaster()}, indexing.NewWorker(indexing.Config{}), passes, func([]string) {})

	report([]string{"links/talk.md"}, []string{"scans/page.png"}, false)
	report([]string{"links/talk.md"}, nil, false)

	isAll, paths := passes.Take()
	if isAll || !slices.Equal(paths, []string{"links/talk.md", "scans/page.png"}) {
		t.Errorf("asked for all %v over %v, want the two paths alone", isAll, paths)
	}
	if isAll, paths := passes.Take(); isAll || len(paths) != 0 {
		t.Errorf("a pass taken is asked for again: all %v over %v", isAll, paths)
	}
}

func TestAReloadAsksForAPassOverTheWholeVault(t *testing.T) {
	passes := triggerpasses.NewPasses()
	passes.Take()
	report := newChangeReporter(&State{ChangeEvents: newChangeBroadcaster()}, indexing.NewWorker(indexing.Config{}), passes, func([]string) {})

	report(nil, nil, true)

	if isAll, _ := passes.Take(); !isAll {
		t.Error("a reload asked for no pass over the whole vault")
	}
}
