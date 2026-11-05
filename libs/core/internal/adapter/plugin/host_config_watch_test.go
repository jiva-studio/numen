package plugin

import (
	"sync/atomic"
	"testing"
	"time"
)

// waitForCount waits until the counter reaches want.
func waitForCount(t *testing.T, got *atomic.Int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for got.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("%d runs, want %d", got.Load(), want)
		}
		time.Sleep(time.Millisecond)
	}
}

// A function registered for a plugin runs each time the plugin has been given its configuration.
func TestAWatcherRunsWhenThePluginIsConfigured(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var runs atomic.Int32
	host.WatchPluginConfig(secretPluginID, func() { runs.Add(1) })

	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}

// A function registered for one plugin does not run when another plugin is configured.
func TestAWatcherDoesNotRunForAnotherPlugin(t *testing.T) {
	host, _ := newSecretHost(t, nil)
	var runs, others atomic.Int32
	host.WatchPluginConfig("md.numen.other", func() { runs.Add(1) })
	host.WatchPluginConfig(secretPluginID, func() { others.Add(1) })

	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
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
	var stopped, kept atomic.Int32
	stop := host.WatchPluginConfig(secretPluginID, func() { stopped.Add(1) })
	host.WatchPluginConfig(secretPluginID, func() { kept.Add(1) })
	stop()

	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &kept, 1)
	if got := stopped.Load(); got != 0 {
		t.Errorf("a stopped watcher ran %d times", got)
	}
}

// A secret handed to the plugin is a configuration of it.
func TestAWatcherRunsWhenASecretIsHandedToThePlugin(t *testing.T) {
	host, _ := newSecretHost(t, newKeys(map[string]string{}))
	var runs atomic.Int32
	host.WatchPluginConfig(secretPluginID, func() { runs.Add(1) })

	if err := host.SetPluginSecret(t.Context(), secretPluginID, "apiKey", "value"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}
