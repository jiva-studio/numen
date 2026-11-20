package plugin_test

import (
	"maps"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestTheLimitsBecomeConfigureValues(t *testing.T) {
	got := plugin.Performance{Provider: "cuda", Profile: "high", Threads: 6, Nice: 5}.GetConfigValues()
	want := map[string]string{"host.computeProvider": "cuda", "host.performanceProfile": "high", "host.performanceThreads": "6", "host.performanceNice": "5"}
	if !maps.Equal(got, want) {
		t.Errorf("values = %v, want %v", got, want)
	}
}

func TestGetConfigValuesIsEmptyWithoutAProfile(t *testing.T) {
	if got := (plugin.Performance{}).GetConfigValues(); len(got) != 0 {
		t.Errorf("values = %v, want none", got)
	}
}
