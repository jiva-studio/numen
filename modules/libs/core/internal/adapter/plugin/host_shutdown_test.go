package plugin

import (
	"context"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func newRunningHost(t *testing.T) (*Host, []*os.Process) {
	t.Helper()
	host := newDiscoveredHost(t)
	var started atomic.Int32
	procs := make([]*os.Process, 0, 4)
	host.SetLauncher(startSleepLauncher(t, &started, &procs))
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
	host.Release("md.numen.ocr")
	return host, procs
}

func waitForProcessExit(t *testing.T, p *os.Process) {
	t.Helper()
	waitUntil(t, "the process to exit", func() bool { return isProcessGone(p) })
}

func TestAShutDownHostDoesNotStartPluginsAgain(t *testing.T) {
	host := newDiscoveredHost(t)
	var started atomic.Int32
	host.SetLauncher(startSleepLauncher(t, &started, nil))

	if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
		t.Fatalf("TeardownAll failed: %v", err)
	}

	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); !errors.Is(err, errHostClosed) {
		t.Fatalf("expected errHostClosed, got %v", err)
	}
	if started.Load() != 0 {
		t.Fatalf("a plugin was launched after shutdown")
	}
}

func TestClosingTheEventBusWaitsForDispatchesInFlight(t *testing.T) {
	bus := NewEventBus(nil, time.Millisecond, 10*time.Millisecond)
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
	host, procs := newRunningHost(t)

	if err := host.SetPluginEnabled("md.numen.ocr", false); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}

	waitForProcessExit(t, procs[0])
	if state := host.supervisors["md.numen.ocr"].GetState(); state != StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func TestDisablingThroughTheSettingStopsThePluginAndIsOneSourceOfTruth(t *testing.T) {
	host, procs := newRunningHost(t)

	if err := host.SetPluginSetting(t.Context(), "md.numen.ocr", "enabled", false); err != nil {
		t.Fatalf("SetPluginSetting failed: %v", err)
	}

	if host.IsPluginEnabled("md.numen.ocr") {
		t.Fatal("the plugin is still enabled after its enabled setting was cleared")
	}
	waitForProcessExit(t, procs[0])
	if _, err := host.EnsureActive(t.Context(), "md.numen.ocr"); err == nil {
		t.Fatal("a disabled plugin was activated")
	}

	if err := host.SetPluginEnabled("md.numen.ocr", true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	if got := host.GetPluginSettings("md.numen.ocr")["enabled"]; got != true {
		t.Fatalf("expected the enabled setting to follow SetPluginEnabled, got %v", got)
	}
}

func TestSettingsLoadedAsDisabledDisableThePlugin(t *testing.T) {
	host := newDiscoveredHost(t)
	host.SetSettings(map[string]map[string]any{"md.numen.ocr": {"enabled": false}})

	if host.IsPluginEnabled("md.numen.ocr") {
		t.Fatal("expected the plugin to be disabled by its stored setting")
	}
	if _, ok := host.GetPlugin("md.numen.ocr"); ok {
		t.Fatal("GetPlugin resolved a disabled plugin")
	}
}

func TestEnteringSafeModeStopsRunningPlugins(t *testing.T) {
	host, procs := newRunningHost(t)

	host.SetSafeMode(true)

	waitForProcessExit(t, procs[0])
	if state := host.supervisors["md.numen.ocr"].GetState(); state != StateDormant {
		t.Fatalf("expected StateDormant, got %v", state)
	}
}

func TestFindPluginByFileExtensionHonoursEnabledAndSafeMode(t *testing.T) {
	host := NewHost("desktop.editor")
	manifest := port.PluginManifest{ID: "md.numen.epub", FileExtensions: []string{".epub"}}
	host.plugins["md.numen.epub"] = DiscoveredPlugin{Manifest: manifest}

	if _, ok := host.FindPluginByFileExtension(".epub"); !ok {
		t.Fatal("expected an enabled plugin to be found")
	}
	if err := host.SetPluginEnabled("md.numen.epub", false); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	if _, ok := host.FindPluginByFileExtension(".epub"); ok {
		t.Fatal("a disabled plugin was found by extension")
	}
	if err := host.SetPluginEnabled("md.numen.epub", true); err != nil {
		t.Fatalf("SetPluginEnabled failed: %v", err)
	}
	host.SetSafeMode(true)
	if _, ok := host.FindPluginByFileExtension(".epub"); ok {
		t.Fatal("a plugin was found by extension in safe mode")
	}
}

func TestTriggerEventContinuesPastAFailingPlugin(t *testing.T) {
	host := NewHost("desktop.editor")
	for _, id := range []string{"a.failing", "b.working"} {
		host.plugins[id] = DiscoveredPlugin{Manifest: port.PluginManifest{ID: id, ActivationEvents: []string{"onOpen"}}}
	}
	var started atomic.Int32
	working := startSleepLauncher(t, &started, nil)
	host.SetLauncher(func(ctx context.Context, p DiscoveredPlugin, portNumber int, token string) (*os.Process, io.ReadCloser, error) {
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
