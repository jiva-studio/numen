package plugin_test

import (
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
)

func waitForCount(t *testing.T, got *atomic.Int32, want int32) {
	t.Helper()
	wiretest.WaitUntil(t, "the watcher to run", func() bool { return got.Load() >= want })
}

// A function registered for a plugin runs each time the plugin has been given its configuration.
func TestAWatcherRunsWhenThePluginIsConfigured(t *testing.T) {
	host, _ := newConfiguredHost(t)
	var runs atomic.Int32
	host.Config.WatchPluginConfig(configuredPluginID, func() { runs.Add(1) })

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}

// A function registered for one plugin does not run when another plugin is configured.
func TestAWatcherDoesNotRunForAnotherPlugin(t *testing.T) {
	host, _ := newConfiguredHost(t)
	var runs, others atomic.Int32
	host.Config.WatchPluginConfig("md.numen.other", func() { runs.Add(1) })
	host.Config.WatchPluginConfig(configuredPluginID, func() { others.Add(1) })

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &others, 1)
	if got := runs.Load(); got != 0 {
		t.Errorf("the watcher of another plugin ran %d times", got)
	}
}

func TestAStoppedWatcherRunsNoMore(t *testing.T) {
	host, _ := newConfiguredHost(t)
	var stopped, remaining atomic.Int32
	stop := host.Config.WatchPluginConfig(configuredPluginID, func() { stopped.Add(1) })
	host.Config.WatchPluginConfig(configuredPluginID, func() { remaining.Add(1) })
	stop()

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &remaining, 1)
	if got := stopped.Load(); got != 0 {
		t.Errorf("a stopped watcher ran %d times", got)
	}
}
