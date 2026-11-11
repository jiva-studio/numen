package plugin

import (
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/plugintest"
)

func waitForCount(t *testing.T, got *atomic.Int32, want int32) {
	t.Helper()
	plugintest.WaitUntil(t, "the watcher to run", func() bool { return got.Load() >= want })
}

// A function registered for a plugin runs each time the plugin has been given its configuration.
func TestAWatcherRunsWhenThePluginIsConfigured(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var runs atomic.Int32
	host.Config.WatchPluginConfig(secretPluginID, func() { runs.Add(1) })

	if err := updateSetting(t.Context(), host, secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}

// A function registered for one plugin does not run when another plugin is configured.
func TestAWatcherDoesNotRunForAnotherPlugin(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var runs, others atomic.Int32
	host.Config.WatchPluginConfig("md.numen.other", func() { runs.Add(1) })
	host.Config.WatchPluginConfig(secretPluginID, func() { others.Add(1) })

	if err := updateSetting(t.Context(), host, secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &others, 1)
	if got := runs.Load(); got != 0 {
		t.Errorf("the watcher of another plugin ran %d times", got)
	}
}

// A stopped watcher runs no more.
func TestAStoppedWatcherRunsNoMore(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var stopped, remaining atomic.Int32
	stop := host.Config.WatchPluginConfig(secretPluginID, func() { stopped.Add(1) })
	host.Config.WatchPluginConfig(secretPluginID, func() { remaining.Add(1) })
	stop()

	if err := updateSetting(t.Context(), host, secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &remaining, 1)
	if got := stopped.Load(); got != 0 {
		t.Errorf("a stopped watcher ran %d times", got)
	}
}

// A secret handed to the plugin is a configuration of it.
func TestAWatcherRunsWhenASecretIsHandedToThePlugin(t *testing.T) {
	host, _ := newSecretHost(t, newKeys(map[string]string{}))
	var runs atomic.Int32
	host.Config.WatchPluginConfig(secretPluginID, func() { runs.Add(1) })

	err := applyChange(host, t.Context(), func() ([]string, error) { return []string{secretPluginID}, nil })
	if err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}
