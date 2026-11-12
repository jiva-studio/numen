package pluginhealth_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

const (
	atlasID   = "org.example.atlas"
	gazetteID = "org.example.gazette"
)

// plugins is the enabled plugins with the commands their manifests declare.
type plugins map[string][]manifest.Command

func (p plugins) GetPlugin(id string) (manifest.Discovered, bool) {
	commands, isEnabled := p[id]
	if !isEnabled {
		return manifest.Discovered{}, false
	}
	return manifest.Discovered{Manifest: manifest.Manifest{ID: id, Contributes: manifest.Contributes{Commands: commands}}}, true
}

func newReport() (*pluginhealth.Report, *pluginhealth.Items) {
	items := pluginhealth.NewItems()
	installed := plugins{
		atlasID:   {{ID: "tiles.download", Title: "Download tiles", Arguments: []manifest.CommandArgument{{Name: "region", Title: "Region", Type: "string"}}}},
		gazetteID: {{ID: "names.refresh", Title: "Refresh names"}},
	}
	return pluginhealth.NewReport(items, installed), items
}

func connectionItem() plugin.HealthItem {
	return plugin.HealthItem{Key: "connection", Severity: plugin.HealthError, Text: "Choose a connection for the tiles"}
}

func TestAReportReplacesTheItemUnderItsKeyAndKeepsItsPlace(t *testing.T) {
	report, items := newReport()
	tiles := plugin.HealthItem{Key: "tiles", Severity: plugin.HealthWarning, Text: "The tiles are a year old"}
	for _, item := range []plugin.HealthItem{connectionItem(), tiles} {
		if err := report.Execute(atlasID, item); err != nil {
			t.Fatal(err)
		}
	}

	again := connectionItem()
	again.Text = "The connection refused the key"
	if err := report.Execute(atlasID, again); err != nil {
		t.Fatal(err)
	}

	list := items.List()
	if len(list) != 2 || list[0].Key != "connection" || list[0].Text != "The connection refused the key" || list[1].Key != "tiles" {
		t.Fatalf("items = %+v, want the replaced connection item first and the tiles item", list)
	}
	if list[0].PluginID != atlasID {
		t.Errorf("plugin = %q, want the reporting plugin", list[0].PluginID)
	}
}

func TestAnItemAtOKAndAClearTakeTheKeyOut(t *testing.T) {
	report, items := newReport()
	if err := report.Execute(atlasID, connectionItem()); err != nil {
		t.Fatal(err)
	}
	if err := report.Execute(gazetteID, plugin.HealthItem{Key: "names", Severity: plugin.HealthError, Text: "The names are missing"}); err != nil {
		t.Fatal(err)
	}

	if err := report.Execute(atlasID, plugin.HealthItem{Key: "connection", Severity: plugin.HealthOK}); err != nil {
		t.Fatal(err)
	}
	if err := pluginhealth.NewClear(items).Execute(gazetteID, "names"); err != nil {
		t.Fatal(err)
	}

	if list := items.List(); len(list) != 0 {
		t.Errorf("items = %+v, want none", list)
	}
}

func TestAnItemPastTheLimitsIsRefused(t *testing.T) {
	cases := map[string]plugin.HealthItem{
		"a long text":       {Key: "connection", Severity: plugin.HealthError, Text: strings.Repeat("a", plugin.MaxHealthText+1)},
		"a long detail":     {Key: "connection", Severity: plugin.HealthError, Text: "Choose a connection", Detail: strings.Repeat("d", plugin.MaxHealthDetail+1)},
		"a long key":        {Key: strings.Repeat("k", plugin.MaxHealthKeyLength+1), Severity: plugin.HealthError, Text: "Choose a connection"},
		"a bad key":         {Key: "Connection Row", Severity: plugin.HealthError, Text: "Choose a connection"},
		"no text":           {Key: "connection", Severity: plugin.HealthError},
		"no severity":       {Key: "connection", Text: "Choose a connection"},
		"a key of the core": {Key: plugin.HealthKeyStart, Severity: plugin.HealthError, Text: "Failed to start"},
	}
	for name, item := range cases {
		t.Run(name, func(t *testing.T) {
			report, items := newReport()
			if err := report.Execute(atlasID, item); !errors.Is(err, plugin.ErrInvalidHealth) {
				t.Fatalf("report = %v, want %v", err, plugin.ErrInvalidHealth)
			}
			if list := items.List(); len(list) != 0 {
				t.Errorf("a refused item is listed: %+v", list)
			}
		})
	}
}

func TestANinthKeyIsRefusedAndAReplacementOfAnEighthIsTaken(t *testing.T) {
	report, items := newReport()
	keys := []string{"tiles", "names", "routes", "heights", "rivers", "roads", "borders", "labels"}
	for _, key := range keys {
		if err := report.Execute(atlasID, plugin.HealthItem{Key: key, Severity: plugin.HealthWarning, Text: "The " + key + " are old"}); err != nil {
			t.Fatal(err)
		}
	}
	items.Set(plugin.HealthItem{PluginID: atlasID, Key: plugin.HealthKeyConfigure, Severity: plugin.HealthError, Text: "The key is refused"})

	if err := report.Execute(atlasID, plugin.HealthItem{Key: "lakes", Severity: plugin.HealthWarning, Text: "The lakes are old"}); !errors.Is(err, plugin.ErrInvalidHealth) {
		t.Fatalf("a ninth key = %v, want %v", err, plugin.ErrInvalidHealth)
	}
	if err := report.Execute(atlasID, plugin.HealthItem{Key: "labels", Severity: plugin.HealthError, Text: "The labels are missing"}); err != nil {
		t.Fatalf("a replacement of an eighth key = %v", err)
	}
	if err := report.Execute(gazetteID, plugin.HealthItem{Key: "lakes", Severity: plugin.HealthWarning, Text: "The lakes are old"}); err != nil {
		t.Fatalf("the first key of another plugin = %v", err)
	}
	if count := len(items.List()); count != 10 {
		t.Errorf("%d items, want the 8 of the plugin, the item of the core and the item of the other plugin", count)
	}
}

func TestAnActionNamesAnAppCommandOrACommandOfThePluginItself(t *testing.T) {
	cases := map[string]struct {
		action  plugin.HealthAction
		isTaken bool
	}{
		"an app command":                        {plugin.HealthAction{Command: "app:file.open", Values: map[string]string{"placement": "split"}}, true},
		"a command of the plugin":               {plugin.HealthAction{Command: atlasID + ":tiles.download", Values: map[string]string{"region": "north"}}, true},
		"a command of another plugin":           {plugin.HealthAction{Command: gazetteID + ":names.refresh"}, false},
		"a command the plugin does not have":    {plugin.HealthAction{Command: atlasID + ":tiles.delete"}, false},
		"no command":                            {plugin.HealthAction{}, false},
		"a value the app command does not take": {plugin.HealthAction{Command: "app:file.open", Values: map[string]string{"row": "2"}}, false},
		"a value outside the options":           {plugin.HealthAction{Command: "app:file.open", Values: map[string]string{"placement": "floating"}}, false},
		"a value the command does not declare":  {plugin.HealthAction{Command: atlasID + ":tiles.download", Values: map[string]string{"zoom": "4"}}, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			report, items := newReport()
			item := connectionItem()
			item.Action = &tc.action
			err := report.Execute(atlasID, item)
			if tc.isTaken && err != nil {
				t.Fatalf("report = %v, want the item taken", err)
			}
			if !tc.isTaken && (!errors.Is(err, plugin.ErrInvalidHealth) || len(items.List()) != 0) {
				t.Fatalf("report = %v with items %+v, want the item refused", err, items.List())
			}
		})
	}
}

func TestOpenSettingsOpensTheSettingsOfTheReportingPluginOnly(t *testing.T) {
	report, items := newReport()
	item := connectionItem()
	item.Action = &plugin.HealthAction{
		Command: plugin.AppCommandOpenSettings,
		Values:  map[string]string{plugin.OpenSettingsPlugin: gazetteID, plugin.OpenSettingsSetting: "source"},
	}

	if err := report.Execute(atlasID, item); err != nil {
		t.Fatal(err)
	}

	action := items.List()[0].Action
	if action.Values[plugin.OpenSettingsPlugin] != atlasID || action.Values[plugin.OpenSettingsSetting] != "source" {
		t.Errorf("values = %v, want the settings of the reporting plugin at source", action.Values)
	}
}

func TestACoreKeyCannotBeClearedByThePlugin(t *testing.T) {
	items := pluginhealth.NewItems()
	items.Set(plugin.HealthItem{PluginID: atlasID, Key: plugin.HealthKeyConfigure, Severity: plugin.HealthError, Text: "The key is refused"})

	if err := pluginhealth.NewClear(items).Execute(atlasID, plugin.HealthKeyConfigure); !errors.Is(err, plugin.ErrInvalidHealth) {
		t.Fatalf("clear = %v, want %v", err, plugin.ErrInvalidHealth)
	}
	if len(items.List()) != 1 {
		t.Error("the plugin cleared an item of the core")
	}
}
