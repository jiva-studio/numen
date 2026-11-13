package wire

import (
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestAPluginProcessIsSentWithItsTimesInMillisecondsAndNoStopWhileItIsHeld(t *testing.T) {
	start := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	got := DescribePluginProcesses([]plugin.ProcessStatus{
		{
			Process:    plugin.Process{PluginID: "org.example.maps", StartTime: start, LastUseTime: start.Add(time.Second), LeaseCount: 2},
			PluginName: "Maps", Health: plugin.HealthWarning, HealthText: "The tiles are old",
		},
		{Process: plugin.Process{PluginID: "org.example.atlas", StartTime: start, LastUseTime: start, IdleStopTime: start.Add(3 * time.Minute)}, Health: plugin.HealthOK},
	})

	if len(got) != 2 {
		t.Fatalf("got %d processes, want 2", len(got))
	}
	maps, atlas := got[0], got[1]
	if maps.GetStartTimeUnixMs() != start.UnixMilli() || maps.GetLastUseTimeUnixMs() != start.UnixMilli()+1000 || maps.GetLeaseCount() != 2 {
		t.Errorf("maps = %v", maps)
	}
	if maps.GetIdleStopTimeUnixMs() != 0 || maps.GetHealth() != v1.HealthSeverity_HEALTH_SEVERITY_WARNING || maps.GetHealthText() != "The tiles are old" {
		t.Errorf("maps = %v", maps)
	}
	if atlas.GetIdleStopTimeUnixMs() != start.Add(3*time.Minute).UnixMilli() || atlas.GetHealth() != v1.HealthSeverity_HEALTH_SEVERITY_OK {
		t.Errorf("atlas = %v", atlas)
	}
}
