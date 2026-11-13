package pluginhealth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

func receiveItems(t *testing.T, watch <-chan []plugin.HealthItem) []plugin.HealthItem {
	t.Helper()
	select {
	case list := <-watch:
		return list
	case <-time.After(time.Second):
		t.Fatal("no list arrived")
		return nil
	}
}

func TestAWatchGetsTheListAtOnceAndAfterEachChange(t *testing.T) {
	items := pluginhealth.NewItems()
	items.Set(plugin.HealthItem{PluginID: atlasID, Key: "tiles", Severity: plugin.HealthWarning, Text: "The tiles are old"})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	watch := items.Watch(ctx)
	if list := receiveItems(t, watch); len(list) != 1 {
		t.Fatalf("first list = %+v, want the one item", list)
	}

	items.Remove(atlasID, "tiles")
	if list := receiveItems(t, watch); len(list) != 0 {
		t.Fatalf("list after the removal = %+v, want none", list)
	}
}

func TestRemovingAPluginKeepsTheItemsOfTheOthers(t *testing.T) {
	items := pluginhealth.NewItems()
	items.Set(plugin.HealthItem{PluginID: atlasID, Key: "tiles", Severity: plugin.HealthWarning, Text: "The tiles are old"})
	items.Set(plugin.HealthItem{PluginID: atlasID, Key: plugin.HealthKeyStart, Severity: plugin.HealthError, Text: "Failed to start"})
	items.Set(plugin.HealthItem{PluginID: gazetteID, Key: "names", Severity: plugin.HealthError, Text: "The names are missing"})

	items.RemovePlugin(atlasID)

	list := items.List()
	if len(list) != 1 || list[0].PluginID != gazetteID {
		t.Fatalf("items = %+v, want only the item of the other plugin", list)
	}
	items.RemoveAll()
	if list := items.List(); len(list) != 0 {
		t.Errorf("items after removing all = %+v", list)
	}
}

func TestAFailedReadOfTheSearchSourceIsTheSearchItemUntilAReadSucceeds(t *testing.T) {
	items := pluginhealth.NewItems()
	report := pluginhealth.NewReportSearchSource(items)

	report.Execute(atlasID, errors.New("choose a connection in the settings of Atlas"))
	list := items.List()
	if len(list) != 1 || list[0].Key != plugin.HealthKeySearch || list[0].PluginID != atlasID || list[0].Text != "choose a connection in the settings of Atlas" {
		t.Fatalf("items = %+v, want the search item of the plugin with the reason", list)
	}
	if action := list[0].Action; action == nil || action.Command != plugin.AppCommandOpenSettings || action.Values[plugin.OpenSettingsPlugin] != atlasID {
		t.Fatalf("action = %+v, want the settings of the plugin", list[0].Action)
	}

	report.Execute(atlasID, nil)
	if list := items.List(); len(list) != 0 {
		t.Errorf("items after a good read = %+v, want none", list)
	}
}
