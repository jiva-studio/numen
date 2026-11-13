package plugin

import "testing"

func TestTheWorstHealthIsTheHighestSeverityOfThePlugin(t *testing.T) {
	items := []HealthItem{
		{PluginID: "org.example.atlas", Key: "tiles", Severity: HealthWarning, Text: "The tiles are old"},
		{PluginID: "org.example.maps", Key: "start", Severity: HealthError, Text: "The maps did not start"},
		{PluginID: "org.example.atlas", Key: "start", Severity: HealthError, Text: "The atlas did not start"},
		{PluginID: "org.example.atlas", Key: "index", Severity: HealthError, Text: "The index is gone"},
	}

	worst, isFound := GetWorstHealth(items, "org.example.atlas")

	if !isFound || worst.Text != "The atlas did not start" {
		t.Errorf("got %+v, %v, want the first error of the atlas", worst, isFound)
	}
}

func TestAPluginWithNoItemHasNoWorstHealth(t *testing.T) {
	items := []HealthItem{{PluginID: "org.example.maps", Key: "start", Severity: HealthError, Text: "The maps did not start"}}

	if worst, isFound := GetWorstHealth(items, "org.example.atlas"); isFound {
		t.Errorf("got %+v, want none", worst)
	}
}
