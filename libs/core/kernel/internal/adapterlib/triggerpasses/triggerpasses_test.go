package triggerpasses_test

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	pluginmodel "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/triggerpasses"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
)

func TestANewVaultStartsWithAPassOverTheWholeVault(t *testing.T) {
	if isAll, _ := triggerpasses.NewPasses().Take(); !isAll {
		t.Error("the passes of a vault opened start with none over the whole vault")
	}
}

func TestPathsAskedForAreTakenOnceInOrder(t *testing.T) {
	passes := triggerpasses.NewPasses()
	passes.Take()

	passes.RequestPaths([]string{"scans/b.png", "links/a.md"})
	passes.RequestPaths([]string{"links/a.md"})
	passes.RequestPaths(nil)

	isAll, paths := passes.Take()
	if isAll || !slices.Equal(paths, []string{"links/a.md", "scans/b.png"}) {
		t.Errorf("asked for all %v over %v, want the two paths alone", isAll, paths)
	}
	if isAll, paths := passes.Take(); isAll || len(paths) != 0 {
		t.Errorf("a pass taken is asked for again: all %v over %v", isAll, paths)
	}
}

// listing declares no trigger and counts how often the triggers are read, which is once for each
// pass that is run.
type listing struct{ reads atomic.Int32 }

func (l *listing) ListTriggers() []pluginmodel.TriggerEntry {
	l.reads.Add(1)
	return nil
}

func (l *listing) ListHandlers() []pluginmodel.CommandEntry { return nil }

func runPasses(t *testing.T) (passes *triggerpasses.Passes, passesRun func() int32, changes chan<- struct{}) {
	t.Helper()
	plugins := &listing{}
	line := make(chan struct{})
	passes = triggerpasses.NewPasses()
	triggers := trigger.NewTriggers(trigger.TriggersDeps{Plugins: plugins})
	go triggerpasses.Run(t.Context(), triggers, domain.Vault{ID: "v1"}, passes, line, func(error) {})
	return passes, plugins.reads.Load, line
}

func TestTheFirstPassRunsOverTheWholeVaultAndAnotherWaitsForARequest(t *testing.T) {
	passes, passesRun, _ := runPasses(t)

	testsupport.WaitFor(t, func() bool { return passesRun() == 1 })
	time.Sleep(20 * time.Millisecond)
	if got := passesRun(); got != 1 {
		t.Fatalf("%d passes ran with nothing asked for, want 1", got)
	}

	passes.RequestPaths([]string{"links/a.md"})
	testsupport.WaitFor(t, func() bool { return passesRun() == 2 })
}

func TestAChangeOfThePluginsAsksForAPassOverTheWholeVault(t *testing.T) {
	_, passesRun, changes := runPasses(t)
	testsupport.WaitFor(t, func() bool { return passesRun() == 1 })

	changes <- struct{}{}

	testsupport.WaitFor(t, func() bool { return passesRun() == 2 })
}
