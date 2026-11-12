package plugin_test

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
)

// taskHost is a running, idle host whose plugin reports tasks through the host endpoint.
type taskHost struct {
	host    *container.PluginRoles
	process *os.Process
	client  numenv1connect.HostQueueServiceClient
}

func newTaskHost(t *testing.T) taskHost {
	t.Helper()
	host := newDiscoveredHost(t)
	var started atomic.Int32
	var procs []*os.Process
	host.Processes.SetLauncher(startSleepLauncher(t, &started, &procs))
	services, _ := lendEndpoint(host, hostendpoint.Address)
	services.SetPluginWork(run.NewPluginWork(&run.Lock{}, run.NewTasks(), nil))
	if _, err := host.Activation.EnsureActive(t.Context(), workingPlugin); err != nil {
		t.Fatal(err)
	}
	host.Activation.Release(workingPlugin)
	return taskHost{host: host, process: procs[0], client: newTaskClient(t, host)}
}

func newTaskClient(t *testing.T, host *container.PluginRoles) numenv1connect.HostQueueServiceClient {
	t.Helper()
	port, err := host.Services.OpenHostEndpoint(t.Context())
	if err != nil || port == 0 {
		t.Fatalf("the host endpoint is not serving: port %d, %v", port, err)
	}
	token := endpointOf(host, workingPlugin).Token
	return numenv1connect.NewHostQueueServiceClient(
		http.DefaultClient, "http://127.0.0.1:"+strconv.Itoa(port), connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func (w taskHost) reportTask(t *testing.T, id string, done, total int64) {
	t.Helper()
	_, err := w.client.ReportTask(t.Context(), connect.NewRequest(&v1.ReportTaskRequest{Task: &v1.Task{Id: id, Done: done, Total: total}}))
	if err != nil {
		t.Fatal(err)
	}
}

func (w taskHost) finishTask(t *testing.T, id string) {
	t.Helper()
	if _, err := w.client.FinishTask(t.Context(), connect.NewRequest(&v1.FinishTaskRequest{Id: id})); err != nil {
		t.Fatal(err)
	}
}

func (w taskHost) listEvictedIDs(t *testing.T) []string {
	t.Helper()
	return w.host.Teardown.EvictIdle(t.Context(), 0, 50*time.Millisecond)
}

func TestAnOpenTaskBlocksIdleEviction(t *testing.T) {
	w := newTaskHost(t)
	w.reportTask(t, "scan.sample", 1, 5)

	if evicted := w.listEvictedIDs(t); len(evicted) != 0 {
		t.Fatalf("evicted %v while a task is open", evicted)
	}
	w.reportTask(t, "scan.sample", 3, 5)
	if evicted := w.listEvictedIDs(t); len(evicted) != 0 {
		t.Fatalf("evicted %v while a task is open after a second report", evicted)
	}

	w.finishTask(t, "scan.sample")
	if evicted := w.listEvictedIDs(t); len(evicted) != 1 || evicted[0] != workingPlugin {
		t.Fatalf("evicted %v after the task finished, want the plugin", evicted)
	}
}

func TestATaskEndedByItsReportAllowsIdleEviction(t *testing.T) {
	for name, report := range map[string]*v1.Task{
		"counted to its total": {Id: "scan.sample", Done: 5, Total: 5},
		"failed":               {Id: "scan.sample", Done: 1, Total: 5, Error: "disk full"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newTaskHost(t)
			w.reportTask(t, "scan.sample", 0, 0)
			if evicted := w.listEvictedIDs(t); len(evicted) != 0 {
				t.Fatalf("evicted %v while a task with nothing to count is open", evicted)
			}
			if _, err := w.client.ReportTask(t.Context(), connect.NewRequest(&v1.ReportTaskRequest{Task: report})); err != nil {
				t.Fatal(err)
			}
			if evicted := w.listEvictedIDs(t); len(evicted) != 1 {
				t.Fatalf("evicted %v after the task ended, want the plugin", evicted)
			}
		})
	}
}

func TestTwoOpenTasksNeedTwoFinishes(t *testing.T) {
	w := newTaskHost(t)
	w.reportTask(t, "first", 0, 4)
	w.reportTask(t, "second", 0, 4)

	w.finishTask(t, "first")
	if evicted := w.listEvictedIDs(t); len(evicted) != 0 {
		t.Fatalf("evicted %v with one task still open", evicted)
	}
	w.finishTask(t, "first")
	if evicted := w.listEvictedIDs(t); len(evicted) != 0 {
		t.Fatalf("evicted %v after the same task finished twice", evicted)
	}

	w.finishTask(t, "second")
	if evicted := w.listEvictedIDs(t); len(evicted) != 1 {
		t.Fatalf("evicted %v after both tasks finished, want the plugin", evicted)
	}
}

func TestARunningQueueTurnBlocksIdleEviction(t *testing.T) {
	w := newTaskHost(t)
	ctx, closeStream := context.WithCancel(t.Context())
	defer closeStream()
	stream, err := w.client.EnqueueWork(ctx, connect.NewRequest(&v1.EnqueueWorkRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Receive() || stream.Msg().GetState() != v1.WorkState_WORK_STATE_RUNNING {
		t.Fatalf("the turn did not start: %v", stream.Err())
	}
	if evicted := w.listEvictedIDs(t); len(evicted) != 0 {
		t.Fatalf("evicted %v during a running turn", evicted)
	}

	closeStream()
	wiretest.WaitUntil(t, "the plugin is evictable after its turn", func() bool {
		return len(w.listEvictedIDs(t)) == 1
	})
}

func TestACrashClearsTheOpenTasks(t *testing.T) {
	w := newTaskHost(t)
	w.reportTask(t, "scan.sample", 0, 4)

	wiretest.KillProcess(t, w.process)
	waitForProcessExit(t, supervisorOf(w.host, workingPlugin).GetExitSignal())

	if _, err := w.host.Activation.EnsureActive(t.Context(), workingPlugin); err != nil {
		t.Fatalf("restart after the crash: %v", err)
	}
	w.host.Activation.Release(workingPlugin)
	if evicted := w.listEvictedIDs(t); len(evicted) != 1 || evicted[0] != workingPlugin {
		t.Fatalf("evicted %v, want the restarted plugin without the crashed process's task", evicted)
	}
}
