package health_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health/healthtest"
)

func TestAReportedItemReachesTheHostAndAClearRemovesIt(t *testing.T) {
	host := healthtest.NewHost(t)
	for name, value := range host.Vars("token-of-the-plugin") {
		t.Setenv(name, value)
	}
	client, err := health.NewClientFromEnv()
	if err != nil {
		t.Fatal(err)
	}

	err = client.Report(t.Context(), health.Item{
		Key: "connection", Severity: health.Error, Text: "Choose a connection", Detail: "No row is set.", Action: health.OpenSettings("source"),
	})
	if err != nil {
		t.Fatal(err)
	}
	items := host.List()
	if len(items) != 1 || items[0].Key != "connection" || items[0].Severity != health.Error || items[0].Text != "Choose a connection" || items[0].Detail != "No row is set." {
		t.Fatalf("items = %+v", items)
	}
	if action := items[0].Action; action == nil || action.Command != "app:settings.open" || action.Values["setting"] != "source" {
		t.Fatalf("action = %+v, want the settings at the setting source", items[0].Action)
	}

	if err := client.Clear(t.Context(), "connection"); err != nil {
		t.Fatal(err)
	}
	if items := host.List(); len(items) != 0 {
		t.Errorf("items after the clear = %+v", items)
	}
}

func TestOpenSettingsWithoutASettingOpensThePlugin(t *testing.T) {
	action := health.OpenSettings("")
	if action.Command != "app:settings.open" || len(action.Values) != 0 {
		t.Errorf("action = %+v, want the settings of the plugin with no setting", action)
	}
}
