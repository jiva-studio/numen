package plugin

import "testing"

func TestAPluginReceivesConfigureWithItsSettings(t *testing.T) {
	host, probe := newConfiguredHost(t)

	if err := host.NotifyPluginConfig(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	if call.GetValues()["model"] != "sonnet" {
		t.Errorf("values = %v, want the default of the model setting", call.GetValues())
	}
}

func TestARefreshConfiguresAPluginOnlyWhenForced(t *testing.T) {
	host, probe := newConfiguredHost(t)
	if err := host.NotifyPluginConfig(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	calls := probe.count()

	if err := host.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls {
		t.Errorf("%d Configure calls after a plain refresh, want %d", probe.count(), calls)
	}

	if err := host.Refresh(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Errorf("%d Configure calls after a forced refresh, want %d", probe.count(), calls+1)
	}
}

func TestAChangeOfASettingReachesAPlugin(t *testing.T) {
	host, probe := newConfiguredHost(t)

	if err := host.UpdatePluginSetting(t.Context(), configuredPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if got := probe.last(t).GetValues()["model"]; got != "opus" {
		t.Errorf("model = %q, want opus", got)
	}
}

func TestAWatcherRunsWhenAPluginIsConfigured(t *testing.T) {
	host, _ := newConfiguredHost(t)
	ran := make(chan struct{}, 1)
	stop := host.WatchPluginConfig(configuredPluginID, func() { ran <- struct{}{} })
	defer stop()

	if err := host.NotifyPluginConfig(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	<-ran
}
