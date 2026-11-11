package plugin

import "testing"

// A running plugin receives the profile in its Configure call, and a change of the profile pushes
// a new call without a restart.
func TestAChangeOfTheProfileReachesARunningPlugin(t *testing.T) {
	host, probe := newSecretHost(t, nil)

	host.Config.SetPerformanceProfile(t.Context(), "eco")
	if got := probe.last(t).GetValues()["host.performanceProfile"]; got != "eco" {
		t.Errorf("profile = %q, want eco", got)
	}

	host.Config.SetPerformanceProfile(t.Context(), "maximum")
	if got := probe.last(t).GetValues()["host.performanceProfile"]; got != "maximum" {
		t.Errorf("profile = %q after the change, want maximum", got)
	}
}

// The same profile again calls no plugin.
func TestTheSameProfileAgainConfiguresNoPlugin(t *testing.T) {
	host, probe := newSecretHost(t, nil)
	host.Config.SetPerformanceProfile(t.Context(), "eco")
	calls := len(probe.calls)

	host.Config.SetPerformanceProfile(t.Context(), "eco")
	if len(probe.calls) != calls {
		t.Errorf("%d Configure calls, want %d", len(probe.calls), calls)
	}
}

// A plugin that starts later receives the profile in its first Configure call.
func TestAPluginConfiguredLaterReceivesTheProfile(t *testing.T) {
	host, probe := newSecretHost(t, nil)
	host.Config.SetPerformanceProfile(t.Context(), "eco")

	if err := host.Config.NotifyPluginConfig(t.Context(), secretPluginID); err != nil {
		t.Fatal(err)
	}
	if got := probe.last(t).GetValues()["host.performanceProfile"]; got != "eco" {
		t.Errorf("profile = %q, want eco", got)
	}
}
