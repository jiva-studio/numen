package wire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestEverySeverityOfAnItemRoundTripsTheWire(t *testing.T) {
	for number := range v1.HealthSeverity_name {
		severity := v1.HealthSeverity(number)
		read := ReadHealthItem(&v1.HealthItem{Key: "tiles", Severity: severity})
		described := DescribeHealthItems([]plugin.HealthItem{read})
		if got := described[0].GetSeverity(); got != severity {
			t.Errorf("%s came back as %s", severity, got)
		}
	}
}

func TestAnItemCarriesItsActionBothWays(t *testing.T) {
	read := ReadHealthItem(&v1.HealthItem{
		Key: "tiles", Severity: v1.HealthSeverity_HEALTH_SEVERITY_ERROR, Text: "The tiles are missing", Detail: "No file is on disk.",
		Action: &v1.HealthAction{Command: "app:settings.open", Values: map[string]string{"setting": "source"}, Title: "Choose a source"},
	})
	read.PluginID = "org.example.atlas"

	described := DescribeHealthItems([]plugin.HealthItem{read})[0]
	if described.GetPluginId() != "org.example.atlas" || described.GetText() != "The tiles are missing" || described.GetDetail() != "No file is on disk." {
		t.Fatalf("described %v", described)
	}
	if action := described.GetAction(); action.GetCommand() != "app:settings.open" || action.GetValues()["setting"] != "source" || action.GetTitle() != "Choose a source" {
		t.Errorf("action %v", action)
	}
}
