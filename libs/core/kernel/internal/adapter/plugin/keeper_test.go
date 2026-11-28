package plugin_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	keptPlugin     = "org.example.kept"
	demandPlugin   = "org.example.demand"
	keptStartLimit = 150 * time.Millisecond
	quietPeriod    = 400 * time.Millisecond
)

type keptLauncher struct {
	t        *testing.T
	port     int
	mu       sync.Mutex
	procs    []*os.Process
	launches atomic.Int32
	mode     atomic.Int32
}

const (
	launchHandshakes = iota
	launchSilent
	launchDiesBeforeHandshake
)

func (l *keptLauncher) launch(ctx context.Context, _ catalogmanifest.DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
	l.launches.Add(1)
	cmd := wiretest.NewIdleCommand(context.WithoutCancel(ctx))
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	l.mu.Lock()
	l.procs = append(l.procs, cmd.Process)
	l.mu.Unlock()
	l.t.Cleanup(func() { wiretest.KillProcess(l.t, cmd.Process) })
	switch l.mode.Load() {
	case launchSilent:
		silent, writer := io.Pipe()
		l.t.Cleanup(func() {
			if err := writer.Close(); err != nil {
				l.t.Error(err)
			}
		})
		return cmd.Process, silent, nil
	case launchDiesBeforeHandshake:
		wiretest.KillProcess(l.t, cmd.Process)
		silent, writer := io.Pipe()
		l.t.Cleanup(func() {
			if err := writer.Close(); err != nil {
				l.t.Error(err)
			}
		})
		return cmd.Process, silent, nil
	}
	return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", l.port))), nil
}

func (l *keptLauncher) killLast() {
	l.mu.Lock()
	defer l.mu.Unlock()
	wiretest.KillProcess(l.t, l.procs[len(l.procs)-1])
}

type keptHost struct {
	*pluginhost.Host
	clock    *testsupport.ManualClock
	launcher *keptLauncher
}

func newKeptHost(t *testing.T) keptHost {
	t.Helper()
	clock := testsupport.NewClock()
	host := newEventHostWithClock(t, clock, map[string]catalogmanifest.Manifest{
		keptPlugin:   {Backend: catalogmanifest.Backend{Lifecycle: catalogmanifest.LifecyclePersistent}},
		demandPlugin: {},
	}, map[string]int{keptPlugin: servePlugin(t, nil), demandPlugin: servePlugin(t, nil)})
	launcher := &keptLauncher{t: t, port: servePlugin(t, nil)}
	host.Supervision.Processes.SetLauncher(func(ctx context.Context, p catalogmanifest.DiscoveredPlugin, port int, token string) (*os.Process, io.ReadCloser, error) {
		return launcher.launch(ctx, p, port, token)
	})
	policy := lifecycle.DefaultActivationPolicy()
	policy.Timeout = keptStartLimit
	host.Supervision.Processes.SetActivationPolicy(policy)
	return keptHost{Host: host, clock: clock, launcher: launcher}
}

func (h keptHost) isRunning(id string) bool {
	sup := supervisorOf(h.Management, id)
	if sup == nil {
		return false
	}
	state := sup.GetState()
	return state == lifecycle.StateActive || state == lifecycle.StateIdle
}

func (h keptHost) waitForRunning(t *testing.T, id string) {
	t.Helper()
	wiretest.WaitUntil(t, id+" to run", func() bool { return h.isRunning(id) })
}

func (h keptHost) setEnabled(t *testing.T, id string, enabled bool) {
	t.Helper()
	err := settingsusecase.NewApplyPluginSetting(settingsusecase.ApplyPluginSettingDeps{
		Values: h.Management.Settings, Stopper: h.Management.Teardown, Failures: h.Supervision.Failures, Listeners: h.Management.Listeners,
		Push: h.Management.Config, Health: h.Supervision.Health, Keeper: h.Management.Keeper,
	}).Execute(t.Context(), id, "enabled", enabled)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAPersistentPluginRunsOnceTheKeeperStarts(t *testing.T) {
	host := newKeptHost(t)

	host.Management.Keeper.KeepAll(t.Context())

	host.waitForRunning(t, keptPlugin)
	if got := host.launcher.launches.Load(); got != 1 {
		t.Errorf("launches = %d, want 1", got)
	}
}

func TestThePluginsThatAreNotPersistentAreNotStartedByTheKeeper(t *testing.T) {
	host := newKeptHost(t)

	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)

	if host.isRunning(demandPlugin) {
		t.Error("a plugin that starts on demand runs after the keeper started")
	}
	if got := host.launcher.launches.Load(); got != 1 {
		t.Errorf("launches = %d, want only the persistent plugin", got)
	}
}

func TestTheKeeperStartDoesNotWaitForAPluginToConnect(t *testing.T) {
	host := newKeptHost(t)
	host.launcher.mode.Store(launchSilent)
	returned := make(chan struct{})

	go func() {
		host.Management.Keeper.KeepAll(t.Context())
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(keptStartLimit / 2):
		t.Fatal("KeepAll waited for the handshake of the plugin")
	}
}

func TestAPersistentPluginHasALeaseForTheKeeperAndIsNotStoppedAsIdle(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)

	host.clock.Advance(supervision.IdleThreshold + time.Second)
	evicted := host.Management.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, releaseGrace)
	host.Management.Teardown.ReleaseProcess(t.Context(), keptPlugin)

	if len(evicted) != 0 {
		t.Errorf("evicted %v after 181 seconds, want none", evicted)
	}
	if !host.isRunning(keptPlugin) {
		t.Error("the persistent plugin stopped after its release")
	}
	processes := host.Supervision.RunningProcesses.ListProcesses()
	if len(processes) != 1 || !processes[0].IdleStopTime.IsZero() {
		t.Fatalf("running processes = %+v, want one with no idle stop time", processes)
	}
	holdsKeeperLease := false
	for _, holder := range processes[0].Holders {
		holdsKeeperLease = holdsKeeperLease || holder.Who == plugindomain.WhoPersistent
	}
	if !holdsKeeperLease {
		t.Errorf("holders = %+v, want the persistent holder", processes[0].Holders)
	}
}

func TestAStartThatDoesNotConnectIsStoppedAndNotRepeatedByATimer(t *testing.T) {
	host := newKeptHost(t)
	host.launcher.mode.Store(launchSilent)

	host.Management.Keeper.KeepAll(t.Context())

	wiretest.WaitUntil(t, "the silent start to fail", func() bool {
		sup := supervisorOf(host.Management, keptPlugin)
		return host.launcher.launches.Load() == 1 && sup != nil && sup.GetState() == lifecycle.StateDormant
	})
	time.Sleep(quietPeriod)
	if got := host.launcher.launches.Load(); got != 1 {
		t.Errorf("launches = %d after the failed start, want 1: nothing may retry by a timer", got)
	}
}

func TestTheNextUseStartsAPersistentPluginWhoseStartFailed(t *testing.T) {
	host := newKeptHost(t)
	host.launcher.mode.Store(launchSilent)
	host.Management.Keeper.KeepAll(t.Context())
	wiretest.WaitUntil(t, "the silent start to fail", func() bool {
		sup := supervisorOf(host.Management, keptPlugin)
		return sup != nil && sup.GetState() == lifecycle.StateDormant && host.launcher.launches.Load() == 1
	})

	host.launcher.mode.Store(launchHandshakes)
	if _, err := host.Management.Activation.EnsureActive(t.Context(), keptPlugin, getTestHolder()); err != nil {
		t.Fatal(err)
	}

	host.waitForRunning(t, keptPlugin)
}

func TestAPersistentPluginThatExitsAfterItWasUpIsStartedOnceAgain(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)

	host.launcher.killLast()

	wiretest.WaitUntil(t, "a second start", func() bool { return host.launcher.launches.Load() == 2 })
	host.waitForRunning(t, keptPlugin)
	time.Sleep(quietPeriod)
	if got := host.launcher.launches.Load(); got != 2 {
		t.Errorf("launches = %d, want 2: one start for one exit", got)
	}
}

func TestAProcessThatDiesBeforeItConnectsIsNotStartedAgain(t *testing.T) {
	host := newKeptHost(t)
	host.launcher.mode.Store(launchDiesBeforeHandshake)

	host.Management.Keeper.KeepAll(t.Context())

	wiretest.WaitUntil(t, "the first launch", func() bool { return host.launcher.launches.Load() >= 1 })
	time.Sleep(keptStartLimit + quietPeriod)
	if got := host.launcher.launches.Load(); got != 1 {
		t.Errorf("launches = %d, want 1: a start that fails is not an exit", got)
	}
}

func TestADisabledPersistentPluginIsStoppedAndNotStartedAgainWhenItsProcessHasEnded(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)

	host.setEnabled(t, keptPlugin, false)

	wiretest.WaitUntil(t, "the plugin to stop", func() bool { return !host.isRunning(keptPlugin) })
	time.Sleep(quietPeriod)
	if host.isRunning(keptPlugin) || host.launcher.launches.Load() != 1 {
		t.Errorf("running = %v, launches = %d, want stopped after one start", host.isRunning(keptPlugin), host.launcher.launches.Load())
	}
}

func TestAnEnabledPersistentPluginRunsAgain(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)
	host.setEnabled(t, keptPlugin, false)
	wiretest.WaitUntil(t, "the plugin to stop", func() bool { return !host.isRunning(keptPlugin) })

	host.setEnabled(t, keptPlugin, true)

	host.waitForRunning(t, keptPlugin)
	if got := host.launcher.launches.Load(); got != 2 {
		t.Errorf("launches = %d, want 2", got)
	}
}

func TestADisabledPersistentPluginIsNotStartedByTheKeeper(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Settings.SetSettings(map[string]map[string]any{keptPlugin: {"enabled": false}})

	host.Management.Keeper.KeepAll(t.Context())

	time.Sleep(quietPeriod)
	if got := host.launcher.launches.Load(); got != 0 {
		t.Errorf("launches = %d, want none for a disabled plugin", got)
	}
}

func TestTheKeeperStartsNoPluginInSafeMode(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Teardown.SetSafeMode(true)

	host.Management.Keeper.KeepAll(t.Context())

	time.Sleep(quietPeriod)
	if got := host.launcher.launches.Load(); got != 0 {
		t.Errorf("launches = %d, want none in safe mode", got)
	}
}

func TestAShutDownStopsAPersistentPluginAndStartsNoneAgain(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)

	if err := host.Management.Teardown.TeardownAll(t.Context(), releaseGrace); err != nil {
		t.Fatal(err)
	}

	time.Sleep(quietPeriod)
	if host.isRunning(keptPlugin) || host.launcher.launches.Load() != 1 {
		t.Errorf("running = %v, launches = %d, want stopped after one start", host.isRunning(keptPlugin), host.launcher.launches.Load())
	}
}

func TestAPersistentPluginOfAnotherVersionIsStartedAgainWhenTheKeeperKeepsItAgain(t *testing.T) {
	host := newKeptHost(t)
	host.Management.Keeper.KeepAll(t.Context())
	host.waitForRunning(t, keptPlugin)

	if err := host.Management.Teardown.StopPlugin(t.Context(), keptPlugin); err != nil {
		t.Fatal(err)
	}
	host.Management.Keeper.Keep(t.Context(), keptPlugin)

	host.waitForRunning(t, keptPlugin)
	if got := host.launcher.launches.Load(); got != 2 {
		t.Errorf("launches = %d, want 2", got)
	}
}
