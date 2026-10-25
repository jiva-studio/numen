package plugin

import (
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

const workingPlugin = "md.numen.engines.ocr"

// workingHost is a running host whose plugin has a task listed and holds the
// lock.
type workingHost struct {
	host    *Host
	process *os.Process
	tasks   *run.Tasks
	turn    pluginport.PluginTurn
}

func newWorkingHost(t *testing.T) workingHost {
	t.Helper()
	host := newDiscoveredHost(t)
	var started atomic.Int32
	var procs []*os.Process
	host.SetLauncher(startSleepLauncher(t, &started, &procs))
	tasks := run.NewTasks()
	work := run.NewPluginWork(&run.Lock{}, tasks)
	host.SetPluginWork(work)
	if _, err := host.EnsureActive(t.Context(), workingPlugin); err != nil {
		t.Fatal(err)
	}
	host.Release(workingPlugin)

	work.ReportTask(workingPlugin, pluginport.PluginTask{ID: "scan.pdf", Title: "Recognising"})
	turn, err := work.EnqueueWork(t.Context(), workingPlugin, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(tasks.List()); got != 1 {
		t.Fatalf("got %d tasks, want the plugin's one", got)
	}
	return workingHost{host: host, process: procs[0], tasks: tasks, turn: turn}
}

func (w workingHost) checkCleared(t *testing.T) {
	t.Helper()
	select {
	case <-w.turn.WatchEnd():
	case <-time.After(5 * time.Second):
		t.Fatal("the stopped plugin still holds the lock")
	}
	plugintest.WaitUntil(t, "the stopped plugin's tasks are gone", func() bool { return len(w.tasks.List()) == 0 })
}

func TestADisabledPluginsTasksAreCleared(t *testing.T) {
	w := newWorkingHost(t)

	if err := w.host.SetPluginEnabled(t.Context(), workingPlugin, false); err != nil {
		t.Fatal(err)
	}
	w.checkCleared(t)
}

func TestACrashedPluginsTasksAreCleared(t *testing.T) {
	w := newWorkingHost(t)

	plugintest.KillProcess(t, w.process)
	waitForProcessExit(t, w.host.supervisors[workingPlugin].GetExitSignal())
	w.checkCleared(t)
}
