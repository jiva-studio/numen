package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/eventbus"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// newRunningHost returns a host with the plugin running and the channel its supervisor closes when the process is reaped.
func newRunningHost(t *testing.T) (*Host, <-chan struct{}) {
	t.Helper()
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("org.example.sample")
	return host, host.supervisors["org.example.sample"].GetExitSignal()
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
	host.SetLauncher(startSleepLauncher(t, &started, nil))

	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); !errors.Is(err, errHostClosed) {
		t.Fatalf("expected errHostClosed, got %v", err)
	}
	if started.Load() != 0 {
		t.Fatalf("a plugin was launched after shutdown")
	}
}

func TestClosingTheEventBusWaitsForDispatchesInFlight(t *testing.T) {
	bus := eventbus.New(nil, time.Millisecond, 10*time.Millisecond)
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

	if err := host.SetPluginEnabled(t.Context(), "org.example.sample", false); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}

	waitForProcessExit(t, exited)
	if state := host.supervisors["org.example.sample"].GetState(); state != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func TestDisablingThroughTheSettingStopsThePluginAndIsOneSourceOfTruth(t *testing.T) {
	host, exited := newRunningHost(t)

	if err := host.SetPluginSetting(t.Context(), "org.example.sample", "enabled", false); err != nil {
		t.Fatalf("SetPluginSetting failed: %v", err)
	}

	if host.IsPluginEnabled("org.example.sample") {
		t.Fatal("the plugin is still enabled after its enabled setting was cleared")
	}
	waitForProcessExit(t, exited)
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err == nil {
		t.Fatal("a disabled plugin was activated")
	}

	if err := host.SetPluginEnabled(t.Context(), "org.example.sample", true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	if got := host.GetPluginSettings("org.example.sample")["enabled"]; got != true {
		t.Fatalf("expected the enabled setting to follow SetPluginEnabled, got %v", got)
	}
}

func TestSettingsLoadedAsDisabledDisableThePlugin(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetSettings(map[string]map[string]any{"org.example.sample": {"enabled": false}})

	if host.IsPluginEnabled("org.example.sample") {
		t.Fatal("expected the plugin to be disabled by its stored setting")
	}
	if _, ok := host.GetPlugin("org.example.sample"); ok {
		t.Fatal("GetPlugin resolved a disabled plugin")
	}
}

func TestEnteringSafeModeStopsRunningPlugins(t *testing.T) {
	host, exited := newRunningHost(t)

	host.SetSafeMode(true)

	waitForProcessExit(t, exited)
	if state := host.supervisors["org.example.sample"].GetState(); state != lifecycle.StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func TestGetContentTypeHonoursEnabledAndSafeMode(t *testing.T) {
	host := NewHost("desktop.editor")
	manifest := pluginport.PluginManifest{ID: "org.example.samples", ContentTypes: []pluginport.PluginContentType{
		{Type: "application/x-example-sample", Globs: []string{"*.sample"}},
	}}
	host.plugins[manifest.ID] = plugindir.DiscoveredPlugin{Manifest: manifest}

	if got := host.GetContentType("samples/Verbs.SAMPLE"); got != "application/x-example-sample" {
		t.Fatalf("GetContentType = %q; want the type the enabled plugin brings", got)
	}
	if err := host.SetPluginEnabled(t.Context(), manifest.ID, false); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	if got := host.GetContentType("samples/Verbs.sample"); got != domain.UnknownContentType {
		t.Fatalf("GetContentType = %q; want unknown with the plugin disabled", got)
	}
	if err := host.SetPluginEnabled(t.Context(), manifest.ID, true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	host.SetSafeMode(true)
	if got := host.GetContentType("samples/Verbs.sample"); got != domain.UnknownContentType {
		t.Fatalf("GetContentType = %q; want unknown in safe mode", got)
	}
}

// ResolveCapability says why no plugin answers: none declares the capability,
// or the ones that do are disabled or switched off by safe mode.
func TestResolveCapabilityReturnsWhyNoPluginAnswers(t *testing.T) {
	host := NewHost("desktop.editor")
	for _, id := range []string{"md.numen.b", "md.numen.a"} {
		host.plugins[id] = plugindir.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id, Capabilities: []string{"org.example.widget"}}}
	}
	getReason := func(capability string) pluginport.UnavailableReason {
		t.Helper()
		_, err := host.ResolveCapability(capability)
		var missing *pluginport.UnavailableError
		if !errors.As(err, &missing) || !errors.Is(err, completionport.ErrUnavailable) || missing.Capability != capability {
			t.Fatalf("ResolveCapability(%q) = %v, want an UnavailableError for it", capability, err)
		}
		return missing.Reason
	}

	if found, err := host.ResolveCapability("org.example.widget"); err != nil || found.Manifest.ID != "md.numen.a" {
		t.Fatalf("ResolveCapability = %q, %v, want the lowest id", found.Manifest.ID, err)
	}
	if got := getReason("org.example.gadget"); got != pluginport.PluginNotInstalled {
		t.Errorf("a capability nothing declares is %v, want not installed", got)
	}
	if err := host.SetPluginEnabled(t.Context(), "md.numen.a", false); err != nil {
		t.Fatal(err)
	}
	if found, err := host.ResolveCapability("org.example.widget"); err != nil || found.Manifest.ID != "md.numen.b" {
		t.Fatalf("ResolveCapability = %q, %v, want the enabled one", found.Manifest.ID, err)
	}
	if err := host.SetPluginEnabled(t.Context(), "md.numen.b", false); err != nil {
		t.Fatal(err)
	}
	if got := getReason("org.example.widget"); got != pluginport.PluginDisabled {
		t.Errorf("a capability every declaring plugin of is disabled is %v, want disabled", got)
	}
	for _, id := range []string{"md.numen.a", "md.numen.b"} {
		if err := host.SetPluginEnabled(t.Context(), id, true); err != nil {
			t.Fatal(err)
		}
	}
	host.SetSafeMode(true)
	if got := getReason("org.example.widget"); got != pluginport.PluginDisabled {
		t.Errorf("a capability in safe mode is %v, want disabled", got)
	}
	if _, ok := host.FindPluginByCapability("org.example.widget"); ok {
		t.Error("FindPluginByCapability found a plugin ResolveCapability does not")
	}
}

// A plugin EnsureActive cannot start is told apart from one that is not there
// or is switched off.
func TestEnsureActiveReturnsWhyAPluginIsUnavailable(t *testing.T) {
	host := NewHost("desktop.editor")
	host.plugins["md.numen.a"] = plugindir.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: "md.numen.a"}}
	if err := host.SetPluginEnabled(t.Context(), "md.numen.a", false); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]pluginport.UnavailableReason{
		"md.numen.absent": pluginport.PluginNotInstalled,
		"md.numen.a":      pluginport.PluginDisabled,
	} {
		_, err := host.EnsureActive(t.Context(), id)
		var missing *pluginport.UnavailableError
		if !errors.As(err, &missing) || missing.Reason != want || missing.PluginID != id {
			t.Errorf("EnsureActive(%q) = %v, want reason %v", id, err, want)
		}
	}
}

func TestTriggerEventContinuesPastAFailingPlugin(t *testing.T) {
	host := NewHost("desktop.editor")
	for _, id := range []string{"a.failing", "b.working"} {
		host.plugins[id] = plugindir.DiscoveredPlugin{Manifest: pluginport.PluginManifest{ID: id, ActivationEvents: []string{"onOpen"}, Backend: pluginport.PluginBackend{Entrypoint: "bin/plugin"}}}
	}
	var started atomic.Int32
	working := startSleepLauncher(t, &started, nil)
	host.SetLauncher(func(ctx context.Context, p plugindir.DiscoveredPlugin, portNumber int, token string) (*os.Process, io.ReadCloser, error) {
		if p.Manifest.ID == "a.failing" {
			return nil, nil, errors.New("exec format error")
		}
		return working(ctx, p, portNumber, token)
	})

	activated, err := host.TriggerEvent(t.Context(), "onOpen")

	if err == nil {
		t.Fatal("expected the failing plugin to be reported")
	}
	if len(activated) != 1 || activated[0] != "b.working" {
		t.Fatalf("expected the working plugin to be activated, got %v", activated)
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestADisableThatCannotBePersistedPreservesThePluginRunning(t *testing.T) {
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))
	if _, err := host.EnsureActive(t.Context(), "org.example.sample"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("org.example.sample")
	host.SetPersistHandler(func(string, string, any) error { return errors.New("disk full") })

	if err := host.SetPluginEnabled(t.Context(), "org.example.sample", false); err == nil {
		t.Fatal("expected the persist error to be returned")
	}

	if !host.IsPluginEnabled("org.example.sample") {
		t.Fatal("the plugin is disabled in memory although the choice was not stored")
	}
	if state := host.supervisors["org.example.sample"].GetState(); state != lifecycle.StateIdle {
		t.Fatalf("expected the plugin to keep running, got state %v", state)
	}
	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}
}

func TestAFailedEvictionIsReportedAndNotCountedAsEvicted(t *testing.T) {
	host := NewHost("desktop.editor")
	var reported []error
	host.SetErrorHandler(func(err error) { reported = append(reported, err) })
	cmd := plugintest.NewIdleCommand()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start process: %v", err)
	}
	t.Cleanup(func() { plugintest.KillProcess(t, cmd.Process) })
	sup := lifecycle.New()
	if err := sup.Attach(cmd.Process, time.Now()); err != nil {
		t.Fatalf("attach: %v", err)
	}
	sup.BeginTeardown()
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	evicted := host.evictTargets(canceled, []pluginTeardownTarget{{id: "p", pid: cmd.Process.Pid, sup: sup, endpoint: emptyEndpoint()}}, time.Millisecond)

	if len(evicted) != 0 {
		t.Fatalf("a failed eviction was counted: %v", evicted)
	}
	if len(reported) != 1 {
		t.Fatalf("expected one reported error, got %v", reported)
	}
}
