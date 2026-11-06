package plugin

import (
	"context"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// openListenerTarget opens the target of a connection to the listener mcp and returns its release.
func openListenerTarget(t *testing.T, host *Host) func() {
	t.Helper()
	_, release, err := host.createListenerTarget(listenerPlugin, "mcp")(t.Context())
	if err != nil {
		t.Fatalf("open the target of the listener: %v", err)
	}
	return release
}

func isEvicted(t *testing.T, host *Host) bool {
	t.Helper()
	for _, id := range host.EvictIdle(t.Context(), 0, 50*time.Millisecond) {
		if id == listenerPlugin {
			return true
		}
	}
	return false
}

func TestTheReleaseOfAStoppedProcessKeepsTheLeaseOfTheNextProcess(t *testing.T) {
	w := newListenerHost(t)
	releaseFirst := openListenerTarget(t, w.host)
	if err := w.host.stopPlugin(t.Context(), listenerPlugin); err != nil {
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

// startKillableLauncher returns a launcher like startRelayLauncher that also returns the processes it
// started.
func startKillableLauncher(t *testing.T, echoPort int) (Launcher, func() []*os.Process) {
	t.Helper()
	ready := "NUMEN_READY port=" + strconv.Itoa(plugintest.ServeHostPing(t)) + " listener.mcp=" + strconv.Itoa(echoPort)
	var mu sync.Mutex
	var procs []*os.Process
	launcher := func(ctx context.Context, _ manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		cmd := plugintest.NewIdleCommand(context.WithoutCancel(ctx))
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() {
			plugintest.KillProcess(t, cmd.Process)
			plugintest.ReapProcess(t, cmd.Process)
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
	host.SetLauncher(launcher)
	discover(t, host)
	releaseFirst := openListenerTarget(t, host)
	if err := getProcesses()[0].Kill(); err != nil {
		t.Fatal(err)
	}
	plugintest.WaitUntil(t, "the crashed plugin to stop", func() bool {
		status, _ := host.GetPluginStatus(listenerPlugin)
		return !status.IsActive
	})
	releaseSecond := openListenerTarget(t, host)
	t.Cleanup(releaseSecond)

	releaseFirst()

	if isEvicted(t, host) {
		t.Error("the plugin was evicted while a connection of its second process is open")
	}
}

func TestAPortSettingFixedAfterABindFailureClearsTheFailureAndServes(t *testing.T) {
	w := newUndiscoveredListenerHost(t, "mcp", getFreePort(t), "listener.mcp="+strconv.Itoa(serveEcho(t)))
	w.host.SetSettings(map[string]map[string]any{listenerPlugin: {"port": float64(80)}})
	discover(t, w.host)
	waitForFailure(t, w.host)
	var cleared []string
	w.host.SetOnFailureCleared(func(id string) { cleared = append(cleared, id) })
	next := getFreePort(t)

	if err := w.host.SetPluginSetting(t.Context(), listenerPlugin, "port", float64(next)); err != nil {
		t.Fatalf("SetPluginSetting %d: %v", next, err)
	}

	if status, _ := w.host.GetPluginStatus(listenerPlugin); status.Failed {
		t.Errorf("status = %+v after the port setting was fixed, want not failed", status)
	}
	if !slices.Equal(cleared, []string{listenerPlugin}) {
		t.Errorf("cleared failures = %v, want %s", cleared, listenerPlugin)
	}
	if reply := sendLine(t, connectListener(t, next), "fixed"); reply != "fixed" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
}

func TestAListenerPortFreedAfterABindFailureClearsTheFailureOnTheNextDiscovery(t *testing.T) {
	taken, takenPort := listenLoopback(t)
	w := newUndiscoveredListenerHost(t, "mcp", takenPort, "listener.mcp="+strconv.Itoa(serveEcho(t)))
	discover(t, w.host)
	waitForFailure(t, w.host)
	testsupport.Close(t, taken)

	discover(t, w.host)

	if status, _ := w.host.GetPluginStatus(listenerPlugin); status.Failed {
		t.Errorf("status = %+v after the port was freed, want not failed", status)
	}
	if reply := sendLine(t, connectListener(t, takenPort), "freed"); reply != "freed" {
		t.Errorf("reply = %q, want the echo of the line", reply)
	}
}

func TestASyncKeepsTheFailureOfAPluginThatFailedToStart(t *testing.T) {
	w := newListenerHost(t)
	cause := errListenerNotReported
	if failure := w.host.recordActivationFailure(listenerPlugin, cause); failure == nil {
		t.Fatal("the failure was not recorded")
	}

	w.host.SetSettings(map[string]map[string]any{listenerPlugin: {"port": float64(getFreePort(t))}})

	if status, _ := w.host.GetPluginStatus(listenerPlugin); !status.Failed {
		t.Errorf("status = %+v, want the failure of the start kept", status)
	}
}
