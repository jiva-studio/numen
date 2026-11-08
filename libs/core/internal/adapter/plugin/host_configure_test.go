package plugin

import "testing"

// newPlainHost returns a host whose one running plugin declares no secrets, with no key store and
// no secret source set.
func newPlainHost(t *testing.T) (*Host, *configured) {
	t.Helper()
	host, probe := newSecretHost(t, nil)
	plugin := host.plugins[secretPluginID]
	plugin.Manifest.Secrets = nil
	host.plugins[secretPluginID] = plugin
	return host, probe
}

func TestAPluginWithoutSecretsReceivesConfigureWithItsSettings(t *testing.T) {
	host, probe := newPlainHost(t)

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	call := probe.last(t)
	if call.GetValues()["model"] != "sonnet" {
		t.Errorf("values = %v, want the default of the model setting", call.GetValues())
	}
	if len(call.GetSecrets()) != 0 {
		t.Errorf("secrets = %v, want none", call.GetSecrets())
	}
}

func TestARefreshConfiguresAPluginWithoutSecretsOnlyWhenForced(t *testing.T) {
	host, probe := newPlainHost(t)
	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	calls := probe.count()

	if err := host.Refresh(t.Context()); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls {
		t.Errorf("%d Configure calls after a plain refresh, want %d", probe.count(), calls)
	}

	if err := host.Refresh(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if probe.count() != calls+1 {
		t.Errorf("%d Configure calls after a forced refresh, want %d", probe.count(), calls+1)
	}
}

func TestAChangeOfASettingReachesAPluginWithoutSecrets(t *testing.T) {
	host, probe := newPlainHost(t)

	if err := host.UpdatePluginSetting(t.Context(), secretPluginID, "model", "opus"); err != nil {
		t.Fatal(err)
	}
	if got := probe.last(t).GetValues()["model"]; got != "opus" {
		t.Errorf("model = %q, want opus", got)
	}
}

func TestAWatcherRunsWhenAPluginWithoutSecretsIsConfigured(t *testing.T) {
	host, _ := newPlainHost(t)
	ran := make(chan struct{}, 1)
	stop := host.WatchPluginConfig(secretPluginID, func() { ran <- struct{}{} })
	defer stop()

	if err := host.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	<-ran
}
