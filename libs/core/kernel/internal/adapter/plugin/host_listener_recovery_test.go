package plugin

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func openListenerTarget(t *testing.T, host *Host) func() {
	t.Helper()
	_, release, err := host.Listeners.createTarget(listenerPlugin, "mcp")(t.Context())
	if err != nil {
		t.Fatalf("open the target of the listener: %v", err)
	}
	return release
}

func isEvicted(t *testing.T, host *Host) bool {
	t.Helper()
	return slices.Contains(host.Teardown.EvictIdle(t.Context(), 0, 50*time.Millisecond), listenerPlugin)
}

func TestTheReleaseOfAStoppedProcessKeepsTheLeaseOfTheNextProcess(t *testing.T) {
	w := newListenerHost(t)
	releaseFirst := openListenerTarget(t, w.host)
	if err := w.host.Teardown.StopPlugin(t.Context(), listenerPlugin); err != nil {
		t.Fatal(err)
	}
	releaseSecond := openListenerTarget(t, w.host)
	t.Cleanup(releaseSecond)
	if got := w.started.Load(); got != 2 {
		t.Fatalf("the plugin started %d times, want 2", got)
	}

	releaseFirst()

	if isEvicted(t, w.host) {
		t.Error("the plugin was evicted while a connection of its second process is open")
	}
}

func startKillableLauncher(t *testing.T, echoPort int) (supervision.Launcher, func() []*os.Process) {
	t.Helper()
	ready := "NUMEN_READY port=" + strconv.Itoa(wiretest.ServeHostPing(t)) + " listener.mcp=" + strconv.Itoa(echoPort)
	var mu sync.Mutex
	var procs []*os.Process
	launcher := func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := wiretest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() {
			wiretest.KillProcess(t, cmd.Process)
			wiretest.ReapProcess(t, cmd.Process)
		})
		mu.Lock()
		procs = append(procs, cmd.Process)
		mu.Unlock()
		return cmd.Process, io.NopCloser(strings.NewReader(ready + "\n")), nil
	}
	return launcher, func() []*os.Process {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(procs)
	}
}

func TestTheReleaseOfACrashedProcessKeepsTheLeaseOfTheNextProcess(t *testing.T) {
	root := t.TempDir()
	createListenerPlugin(t, root, "mcp", getFreePort(t))
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	teardownAfter(t, host)
	launcher, getProcesses := startKillableLauncher(t, serveEcho(t))
	host.Processes.SetLauncher(launcher)
	discover(t, host)
	releaseFirst := openListenerTarget(t, host)
	if err := getProcesses()[0].Kill(); err != nil {
		t.Fatal(err)
	}
	wiretest.WaitUntil(t, "the crashed plugin to stop", func() bool {
		status, _ := host.Registry.GetPluginStatus(listenerPlugin)
		return !status.IsActive
	})
	releaseSecond := openListenerTarget(t, host)
	t.Cleanup(releaseSecond)

	releaseFirst()

	if isEvicted(t, host) {
		t.Error("the plugin was evicted while a connection of its second process is open")
	}
}

func TestPortSettingFixedAfterBindFailureClearsFailureAndServes(t *testing.T) {
	w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), "listener.mcp="+strconv.Itoa(serveEcho(t)))
	w.host.Settings.SetSettings(map[string]map[string]any{listenerPlugin: {"port": float64(80)}})
	discover(t, w.host)
	waitForFailure(t, w.host)
	var cleared []string
	w.host.Failures.SetOnFailureCleared(func(id string) { cleared = append(cleared, id) })
	next := getFreePort(t)

	if err := updateSetting(t.Context(), w.host, listenerPlugin, "port", float64(next)); err != nil {
		t.Fatalf("applying port %d: %v", next, err)
	}

	if status, _ := w.host.Registry.GetPluginStatus(listenerPlugin); status.Failed {
		t.Errorf("status = %+v after the port setting was fixed, want not failed", status)
	}
	if !slices.Equal(cleared, []string{listenerPlugin}) {
		t.Errorf("cleared failures = %v, want %s", cleared, listenerPlugin)
	}
	if reply := sendLine(t, connectListener(t, next), "fixed"); reply != "fixed" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
}

func TestListenerPortFreedAfterBindFailureClearsItOnDiscovery(t *testing.T) {
	taken, takenPort := listenLoopback(t)
	w := newUndiscoveredListenerHost(t, "mcp", takenPort, "listener.mcp="+strconv.Itoa(serveEcho(t)))
	discover(t, w.host)
	waitForFailure(t, w.host)
	testsupport.Close(t, taken)

	discover(t, w.host)

	if status, _ := w.host.Registry.GetPluginStatus(listenerPlugin); status.Failed {
		t.Errorf("status = %+v after the port was freed, want not failed", status)
	}
	if reply := sendLine(t, connectListener(t, takenPort), "freed"); reply != "freed" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
}

func TestASyncKeepsTheFailureOfAPluginThatFailedToStart(t *testing.T) {
	w := newListenerHost(t)
	cause := supervision.ErrListenerNotReported
	if failure := w.host.Failures.RecordActivationFailure(listenerPlugin, w.host.Registry.set.getName(listenerPlugin), cause); failure == nil {
		t.Fatal("the failure was not recorded")
	}

	w.host.Settings.SetSettings(map[string]map[string]any{listenerPlugin: {"port": float64(getFreePort(t))}})

	if status, _ := w.host.Registry.GetPluginStatus(listenerPlugin); !status.Failed {
		t.Errorf("status = %+v, want the failure of the start kept", status)
	}
}

// listenerTargetErrors collects the reported errors of the targets of the listener console.
type listenerTargetErrors struct {
	mu   sync.Mutex
	errs []error
}

func (e *listenerTargetErrors) add(err error) {
	if !strings.Contains(err.Error(), "activate plugin "+listenerPlugin+" for listener console") {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.errs = append(e.errs, err)
}

func (e *listenerTargetErrors) get() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.errs)
}

func connectUntilClosed(t *testing.T, port int) {
	t.Helper()
	conn := connectListener(t, port)
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("the connection stayed open after the plugin failed to activate")
	}
}

func TestRepeatedActivationFailureIsReportedOnceAndAfterRetry(t *testing.T) {
	w := newUndiscoveredListenerHost(t, "console", getFreePort(t), "")
	var reports listenerTargetErrors
	w.host.Failures.SetErrorHandler(reports.add)
	discover(t, w.host)

	for i := range 3 {
		connectUntilClosed(t, w.port)
		if errs := reports.get(); len(errs) != 1 {
			t.Fatalf("after connection %d: reported %d activation failures (%v), want 1", i+1, len(errs), errs)
		}
	}

	failure, err := w.host.Retry.RetryPlugin(t.Context(), listenerPlugin)
	if failure == nil && err == nil {
		t.Fatal("RetryPlugin succeeded for a plugin that does not report its listener")
	}

	for i := range 2 {
		connectUntilClosed(t, w.port)
		if errs := reports.get(); len(errs) != 2 {
			t.Fatalf("after connection %d since the retry: reported %d activation failures (%v), want 2", i+1, len(errs), errs)
		}
	}
}
