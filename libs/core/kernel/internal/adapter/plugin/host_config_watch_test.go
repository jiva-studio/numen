package plugin_test

import (
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func waitForCount(t *testing.T, got *atomic.Int32, want int32) {
	t.Helper()
	wiretest.WaitUntil(t, "the watcher to run", func() bool { return got.Load() >= want })
}

// A function registered for a plugin runs each time the plugin has been given its configuration.
func TestAWatcherRunsWhenThePluginIsConfigured(t *testing.T) {
	host, _ := newConfiguredHost(t)
	var runs atomic.Int32
	host.Management.Config.WatchPluginConfig(configuredPluginID, func() { runs.Add(1) })

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}

// A function registered for one plugin does not run when another plugin is configured.
func TestAWatcherDoesNotRunForAnotherPlugin(t *testing.T) {
	host, _ := newConfiguredHost(t)
	var runs, others atomic.Int32
	host.Management.Config.WatchPluginConfig("md.numen.other", func() { runs.Add(1) })
	host.Management.Config.WatchPluginConfig(configuredPluginID, func() { others.Add(1) })

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
	stop := host.Management.Config.WatchPluginConfig(configuredPluginID, func() { stopped.Add(1) })
	host.Management.Config.WatchPluginConfig(configuredPluginID, func() { remaining.Add(1) })
	stop()

	if err := updateSetting(t.Context(), host, configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &remaining, 1)
	if got := stopped.Load(); got != 0 {
		t.Errorf("a stopped watcher ran %d times", got)
	}
}

// stoppedPluginID is a plugin the host knows that has no process.
const stoppedPluginID = "md.numen.engines.stopped"

func newStoppedHost(t *testing.T) *pluginhost.Host {
	t.Helper()
	host := pluginhost.New(testsupport.GetMachine(), "desktop.editor")
	setPlugin(host.Management, stoppedPluginID, manifest.DiscoveredPlugin{Manifest: manifest.Manifest{
		ID:       stoppedPluginID,
		Version:  "2026.1.1",
		Settings: []manifest.Setting{{Key: "model", Type: "string", Title: "Model", Default: "small"}},
	}})
	return host
}

// A setting of a plugin that does not run changes a configuration the plugin has not been given, and its
// watchers run.
func TestAWatcherRunsWhenASettingOfAStoppedPluginChanges(t *testing.T) {
	host := newStoppedHost(t)
	var runs atomic.Int32
	host.Management.Config.WatchPluginConfig(stoppedPluginID, func() { runs.Add(1) })

	if err := updateSetting(t.Context(), host, stoppedPluginID, "model", "large"); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
}

// A forced refresh, as after an edit of a connection the plugin uses, runs the watchers of a plugin that
// does not run; a plain refresh does not.
func TestAForcedRefreshRunsTheWatcherOfAStoppedPlugin(t *testing.T) {
	host := newStoppedHost(t)
	var runs atomic.Int32
	host.Management.Config.WatchPluginConfig(stoppedPluginID, func() { runs.Add(1) })

	if err := host.Management.Config.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := host.Management.Config.Refresh(t.Context(), stoppedPluginID); err != nil {
		t.Fatal(err)
	}

	waitForCount(t, &runs, 1)
	if got := runs.Load(); got != 1 {
		t.Errorf("the watcher ran %d times, want once for the forced refresh", got)
	}
}

// The revision of the configuration changes with a setting and with the performance profile, and stays when
// only the enabled flag changes.
func TestTheConfigRevisionFollowsTheSettingsButNotTheEnabledFlag(t *testing.T) {
	host := newStoppedHost(t)
	first := host.Management.Config.GetPluginConfigRevision(stoppedPluginID)

	if err := setEnabled(t.Context(), host, stoppedPluginID, false); err != nil {
		t.Fatal(err)
	}
	if got := host.Management.Config.GetPluginConfigRevision(stoppedPluginID); got != first {
		t.Errorf("disabling the plugin changed the revision")
	}
	if err := updateSetting(t.Context(), host, stoppedPluginID, "model", "large"); err != nil {
		t.Fatal(err)
	}
	afterModel := host.Management.Config.GetPluginConfigRevision(stoppedPluginID)
	if afterModel == first {
		t.Errorf("a new model left the revision as it was")
	}
	host.Management.Config.SetPerformance(t.Context(), plugindomain.Performance{Profile: "low", Threads: 2, Nice: 15})
	if got := host.Management.Config.GetPluginConfigRevision(stoppedPluginID); got == afterModel {
		t.Errorf("a new performance profile left the revision as it was")
	}
}
