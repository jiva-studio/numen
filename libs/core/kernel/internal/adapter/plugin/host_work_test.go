package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"os"
	"sync/atomic"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

const workingPlugin = "org.example.sample"

// workingHost is a running host whose plugin has a task listed and holds the
// lock.
type workingHost struct {
	host    *Host
	process *os.Process
	tasks   *run.Tasks
	lease   *run.Lease
}

func newWorkingHost(t *testing.T) workingHost {
	t.Helper()
	host := newDiscoveredHost(t)
	var started atomic.Int32
	var procs []*os.Process
	host.Processes.SetLauncher(startSleepLauncher(t, &started, &procs))
	tasks := run.NewTasks()
	work := run.NewPluginWork(&run.Lock{}, tasks, nil)
	services, _ := lendEndpoint(host, hostendpoint.Address)
	services.SetPluginWork(work)
	if _, err := host.Activation.EnsureActive(t.Context(), workingPlugin); err != nil {
		t.Fatal(err)
	}
	host.Activation.Release(workingPlugin)

	work.ReportTask(workingPlugin, plugindomain.Task{ID: "scan.sample", Title: "Scanning"})
	turn, err := work.EnqueueWork(t.Context(), workingPlugin, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(tasks.List()); got != 1 {
		t.Fatalf("got %d tasks, want the plugin's one", got)
	}
	return workingHost{host: host, process: procs[0], tasks: tasks, lease: turn}
}

func (w workingHost) checkCleared(t *testing.T) {
	t.Helper()
	select {
	case <-w.lease.WatchEnd():
	case <-time.After(5 * time.Second):
		t.Fatal("the stopped plugin still holds the lock")
	}
	wiretest.WaitUntil(t, "the stopped plugin's tasks are gone", func() bool { return len(w.tasks.List()) == 0 })
}

func TestDisabledPluginTasksAreCleared(t *testing.T) {
	w := newWorkingHost(t)

	if err := setEnabled(t.Context(), w.host, workingPlugin, false); err != nil {
		t.Fatal(err)
	}
	w.checkCleared(t)
}

func TestCrashedPluginTasksAreCleared(t *testing.T) {
	w := newWorkingHost(t)

	wiretest.KillProcess(t, w.process)
	waitForProcessExit(t, supervisorOf(w.host, workingPlugin).GetExitSignal())
	w.checkCleared(t)
}
