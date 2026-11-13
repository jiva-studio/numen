package plugin_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/eventbus"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newRunningHost(t *testing.T) (*container.PluginRoles, <-chan struct{}) {
	t.Helper()
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.Processes.SetLauncher(startSleepLauncher(t, &started, nil))
	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Activation.Release("org.example.sample")
	return host, supervisorOf(host, "org.example.sample").GetExitSignal()
}

func waitForProcessExit(t *testing.T, exited <-chan struct{}) {
	t.Helper()
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the process to exit")
	}
}

func TestAShutDownHostDoesNotStartPluginsAgain(t *testing.T) {
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.Processes.SetLauncher(startSleepLauncher(t, &started, nil))

	if err := host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); !errors.Is(err, plugin.ErrHostClosed) {
		t.Fatalf("expected plugin.ErrHostClosed, got %v", err)
	}
	if started.Load() != 0 {
		t.Fatalf("a plugin was launched after shutdown")
	}
}

func TestClosingTheEventBusWaitsForDispatchesInFlight(t *testing.T) {
	bus := eventbus.New(nil, testsupport.NewClock(), time.Millisecond, 10*time.Millisecond)
	entered := make(chan struct{})
	release := make(chan struct{})
	bus.Subscribe("onSlow", func(_ context.Context, _ string) error {
		close(entered)
		<-release
		return nil
	})

	bus.Publish("onSlow")
	<-entered

	closed := make(chan error, 1)
	go func() { closed <- bus.Close(t.Context()) }()

	select {
	case err := <-closed:
		t.Fatalf("Close returned while a dispatch was in flight: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	if err := <-closed; err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestADisabledPluginIsStopped(t *testing.T) {
	host, exited := newRunningHost(t)

	if err := setEnabled(t.Context(), host, "org.example.sample", false); err != nil {
		t.Fatalf("switching the plugin failed: %v", err)
	}

	waitForProcessExit(t, exited)
	if state := supervisorOf(host, "org.example.sample").GetState(); state != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func TestApplyingTheEnabledSettingAsFalseStopsThePlugin(t *testing.T) {
	host, exited := newRunningHost(t)

	if err := updateSetting(t.Context(), host, "org.example.sample", "enabled", false); err != nil {
		t.Fatalf("applying the setting failed: %v", err)
	}

	if host.Registry.IsPluginEnabled("org.example.sample") {
		t.Fatal("the plugin is still enabled after its enabled setting was cleared")
	}
	waitForProcessExit(t, exited)
	if _, err := host.Activation.EnsureActive(t.Context(), "org.example.sample"); err == nil {
		t.Fatal("a disabled plugin was activated")
	}

	if err := setEnabled(t.Context(), host, "org.example.sample", true); err != nil {
		t.Fatalf("switching the plugin failed: %v", err)
	}
	if got := host.Settings.GetPluginSettings("org.example.sample")["enabled"]; got != true {
		t.Fatalf("expected the enabled setting to follow the change, got %v", got)
	}
}

func TestSettingsLoadedAsDisabledDisableThePlugin(t *testing.T) {
	host := newDiscoveredHost(t)
	host.Settings.SetSettings(map[string]map[string]any{"org.example.sample": {"enabled": false}})

	if host.Registry.IsPluginEnabled("org.example.sample") {
		t.Fatal("expected the plugin to be disabled by its stored setting")
	}
	if _, ok := host.Registry.GetPlugin("org.example.sample"); ok {
		t.Fatal("GetPlugin resolved a disabled plugin")
	}
}

func TestEnteringSafeModeStopsRunningPlugins(t *testing.T) {
	host, exited := newRunningHost(t)

	host.Teardown.SetSafeMode(true)

	waitForProcessExit(t, exited)
	if state := supervisorOf(host, "org.example.sample").GetState(); state != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

// ResolveCapability reports why no plugin responds: none declares the
// capability, or those that do are disabled or turned off by safe mode.
func TestResolveCapabilityReturnsWhyNoPluginAnswers(t *testing.T) {
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor")
	for _, id := range []string{"md.numen.b", "md.numen.a"} {
		setPlugin(host, id, manifest.Discovered{Manifest: manifest.Manifest{ID: id, Capabilities: []string{"org.example.widget"}}})
	}
	getReason := func(capability string) plugindomain.UnavailableReason {
		t.Helper()
		_, err := host.Registry.ResolveCapability(capability)
		var missing *plugindomain.UnavailableError
		if !errors.As(err, &missing) || !errors.Is(err, domain.ErrUnavailable) || missing.Capability != capability {
			t.Fatalf("ResolveCapability(%q) = %v, want an UnavailableError for it", capability, err)
		}
		return missing.Reason
	}

	if found, err := host.Registry.ResolveCapability("org.example.widget"); err != nil || found.Manifest.ID != "md.numen.a" {
		t.Fatalf("ResolveCapability = %q, %v, want the lowest id", found.Manifest.ID, err)
	}
	if got := getReason("org.example.gadget"); got != plugindomain.NotInstalled {
		t.Errorf("a capability nothing declares is %v, want not installed", got)
	}
	if err := setEnabled(t.Context(), host, "md.numen.a", false); err != nil {
		t.Fatal(err)
	}
	if found, err := host.Registry.ResolveCapability("org.example.widget"); err != nil || found.Manifest.ID != "md.numen.b" {
		t.Fatalf("ResolveCapability = %q, %v, want the enabled one", found.Manifest.ID, err)
	}
	if err := setEnabled(t.Context(), host, "md.numen.b", false); err != nil {
		t.Fatal(err)
	}
	if got := getReason("org.example.widget"); got != plugindomain.Disabled {
		t.Errorf("a capability every declaring plugin of is disabled is %v, want disabled", got)
	}
	for _, id := range []string{"md.numen.a", "md.numen.b"} {
		if err := setEnabled(t.Context(), host, id, true); err != nil {
			t.Fatal(err)
		}
	}
	host.Teardown.SetSafeMode(true)
	if got := getReason("org.example.widget"); got != plugindomain.Disabled {
		t.Errorf("a capability in safe mode is %v, want disabled", got)
	}
	if _, ok := host.Registry.FindPluginByCapability("org.example.widget"); ok {
		t.Error("FindPluginByCapability found a plugin ResolveCapability does not")
	}
}

// A plugin EnsureActive cannot start is distinguished from one that is absent
// or turned off.
func TestEnsureActiveReturnsWhyAPluginIsUnavailable(t *testing.T) {
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor")
	setPlugin(host, "md.numen.a", manifest.Discovered{Manifest: manifest.Manifest{ID: "md.numen.a"}})
	if err := setEnabled(t.Context(), host, "md.numen.a", false); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]plugindomain.UnavailableReason{
		"md.numen.absent": plugindomain.NotInstalled,
		"md.numen.a":      plugindomain.Disabled,
	} {
		_, err := host.Activation.EnsureActive(t.Context(), id)
		var missing *plugindomain.UnavailableError
		if !errors.As(err, &missing) || missing.Reason != want || missing.PluginID != id {
			t.Errorf("EnsureActive(%q) = %v, want reason %v", id, err, want)
		}
	}
}

func TestTriggerEventContinuesPastAFailingPlugin(t *testing.T) {
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor")
	for _, id := range []string{"a.failing", "b.working"} {
		setPlugin(host, id, manifest.Discovered{Manifest: manifest.Manifest{ID: id, ActivationEvents: []string{manifest.ActivationEventSearchSourceReady}, Backend: manifest.Backend{Entrypoint: "bin/plugin"}}})
	}
	var started atomic.Int32
	working := startSleepLauncher(t, &started, nil)
	host.Processes.SetLauncher(func(ctx context.Context, p manifest.Discovered, portNumber int, token string) (*os.Process, io.ReadCloser, error) {
		if p.Manifest.ID == "a.failing" {
			return nil, nil, errors.New("exec format error")
		}
		return working(ctx, p, portNumber, token)
	})

	activated, err := host.Events.TriggerEvent(t.Context(), manifest.ActivationEventSearchSourceReady)

	if err == nil {
		t.Fatal("expected the failing plugin to be reported")
	}
	if len(activated) != 1 || activated[0] != "b.working" {
		t.Fatalf("expected the working plugin to be activated, got %v", activated)
	}
	if err := host.Teardown.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAFailedEvictionIsReportedAndNotCountedAsEvicted(t *testing.T) {
	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor")
	var reported []error
	host.Failures.SetErrorHandler(func(err error) { reported = append(reported, err) })
	cmd := wiretest.NewIdleCommand(t.Context())
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	t.Cleanup(func() { wiretest.KillProcess(t, cmd.Process) })
	sup := lifecycle.New(nil)
	if err := sup.Attach(t.Context(), cmd.Process, testsupport.GetMachine().Now()); err != nil {
		t.Fatalf("attach: %v", err)
	}
	release := make(chan struct{})
	teardownResult := make(chan error, 1)
	t.Cleanup(func() {
		close(release)
		if err := <-teardownResult; err != nil {
			t.Error(err)
		}
	})
	go func() {
		teardownResult <- sup.Teardown(context.WithoutCancel(t.Context()), 100*time.Millisecond, func(context.Context) error {
			<-release
			return nil
		})
	}()
	for sup.GetState() != lifecycle.StateDeactivating {
		time.Sleep(time.Millisecond)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	evicted := plugin.GetStopper(host.Teardown).EvictTargets(canceled, []supervision.Target{{ID: "p", PID: cmd.Process.Pid, Sup: sup, Endpoint: supervision.NoEndpoint()}}, time.Millisecond)

	if len(evicted) != 0 {
		t.Fatalf("a failed eviction was counted: %v", evicted)
	}
	if len(reported) != 1 {
		t.Fatalf("expected one reported error, got %v", reported)
	}
}
