package plugin_test

import (
	"maps"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
)

func getEco() plugindomain.Performance {
	return plugindomain.Performance{Profile: "eco", Threads: 1, Nice: 19}
}

func getPerformanceValues(values map[string]string) map[string]string {
	return map[string]string{
		"host.performanceProfile": values["host.performanceProfile"],
		"host.performanceThreads": values["host.performanceThreads"],
		"host.performanceNice":    values["host.performanceNice"],
	}
}

// A running plugin receives the limits in its Configure call, and a change of the profile pushes
// a new call without a restart.
func TestAChangeOfTheProfileReachesARunningPlugin(t *testing.T) {
	host, probe := newConfiguredHost(t)

	host.Config.SetPerformance(t.Context(), getEco())
	want := map[string]string{"host.performanceProfile": "eco", "host.performanceThreads": "1", "host.performanceNice": "19"}
	if got := getPerformanceValues(probe.last(t).GetValues()); !maps.Equal(got, want) {
		t.Errorf("values = %v, want %v", got, want)
	}

	host.Config.SetPerformance(t.Context(), plugindomain.Performance{Profile: "maximum", Threads: 16, Nice: 0})
	want = map[string]string{"host.performanceProfile": "maximum", "host.performanceThreads": "16", "host.performanceNice": "0"}
	if got := getPerformanceValues(probe.last(t).GetValues()); !maps.Equal(got, want) {
		t.Errorf("values = %v after the change, want %v", got, want)
	}
}

func TestTheSameProfileAgainConfiguresNoPlugin(t *testing.T) {
	host, probe := newConfiguredHost(t)
	host.Config.SetPerformance(t.Context(), getEco())
	calls := len(probe.calls)

	host.Config.SetPerformance(t.Context(), getEco())
	if len(probe.calls) != calls {
		t.Errorf("%d Configure calls, want %d", len(probe.calls), calls)
	}
}

// A plugin that starts later receives the limits in its first Configure call.
func TestAPluginConfiguredLaterReceivesTheProfile(t *testing.T) {
	host, probe := newConfiguredHost(t)
	plugin.SetPerformance(host.Config, getEco())

	if err := host.Config.NotifyPluginConfig(t.Context(), configuredPluginID); err != nil {
		t.Fatal(err)
	}
	if got := probe.last(t).GetValues()["host.performanceThreads"]; got != "1" {
		t.Errorf("threads = %q, want 1", got)
	}
}
