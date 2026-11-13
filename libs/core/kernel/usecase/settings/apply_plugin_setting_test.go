package settings_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// pluginRuntime holds the settings of one plugin in memory and records what the use case does to the
// plugin around them.
type pluginRuntime struct {
	values         map[string]any
	events         []string
	changesDeps    bool
	syncError      error
	stopError      error
	valuesAtNotify map[string]any
	health         *pluginhealth.Items
}

func newPluginRuntime() *pluginRuntime {
	return &pluginRuntime{values: map[string]any{"mode": "a"}, health: pluginhealth.NewItems()}
}

func (r *pluginRuntime) ReplaceSetting(_, key string, value any) (any, bool, bool) {
	previous, had := r.values[key]
	r.values[key] = value
	r.events = append(r.events, "replace "+key)
	return previous, had, r.changesDeps
}

func (r *pluginRuntime) RestoreSetting(_, key string, previous any, had bool) {
	if had {
		r.values[key] = previous
	} else {
		delete(r.values, key)
	}
	r.events = append(r.events, "restore "+key)
}

func (r *pluginRuntime) FillGeneratedSettings() { r.events = append(r.events, "fill generated") }

func (r *pluginRuntime) StopPlugin(context.Context, string) error {
	r.events = append(r.events, "stop")
	return r.stopError
}

func (r *pluginRuntime) ClearFailure(string) { r.events = append(r.events, "clear failure") }

func (r *pluginRuntime) SyncListeners(context.Context, string) error {
	r.events = append(r.events, "sync listeners")
	return r.syncError
}

func (r *pluginRuntime) RefreshListeners(context.Context, string) {
	r.events = append(r.events, "refresh listeners")
}

func (r *pluginRuntime) NotifyPluginConfig(context.Context, string) error {
	r.events = append(r.events, "notify")
	r.valuesAtNotify = maps.Clone(r.values)
	return nil
}

func newApplier(r *pluginRuntime) *settings.ApplyPluginSetting {
	return settings.NewApplyPluginSetting(settings.ApplyPluginSettingDeps{
		Values: r, Stopper: r, Failures: r, Listeners: r, Push: r, Health: r.health,
	})
}

func TestAnOrdinarySettingIsStoredThenItsListenersAreSyncedAndThePluginIsNotified(t *testing.T) {
	runtime := newPluginRuntime()

	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "mode", "b"); err != nil {
		t.Fatal(err)
	}

	if want := []string{"replace mode", "sync listeners", "notify"}; !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
	if runtime.valuesAtNotify["mode"] != "b" {
		t.Errorf("the plugin was notified with %v, want the new value", runtime.valuesAtNotify)
	}
}

func TestASettingThatChangesTheDependenciesStopsThePluginBeforeItIsNotified(t *testing.T) {
	runtime := newPluginRuntime()
	runtime.changesDeps = true

	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "model", "small"); err != nil {
		t.Fatal(err)
	}

	if want := []string{"replace model", "stop", "sync listeners", "notify"}; !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
}

func TestAPluginThatCannotBeStoppedForNewDependenciesIsNotNotified(t *testing.T) {
	runtime := newPluginRuntime()
	runtime.changesDeps = true
	runtime.stopError = errors.New("stuck")

	err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "model", "small")

	if !errors.Is(err, runtime.stopError) {
		t.Errorf("error = %v, want the stop error", err)
	}
	if want := []string{"replace model", "stop"}; !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
}

func TestADisabledPluginIsStoppedAndForgetsItsFailure(t *testing.T) {
	runtime := newPluginRuntime()

	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "enabled", false); err != nil {
		t.Fatal(err)
	}

	want := []string{"replace enabled", "stop", "clear failure", "refresh listeners", "notify"}
	if !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
}

func TestADisabledPluginLosesItsHealthItemsAndAnEnabledOneKeepsThem(t *testing.T) {
	runtime := newPluginRuntime()
	runtime.health.Set(plugin.HealthItem{PluginID: "org.example.scans", Key: "scanner", Severity: plugin.HealthError, Text: "Choose a scanner"})
	runtime.health.Set(plugin.HealthItem{PluginID: "org.example.maps", Key: "tiles", Severity: plugin.HealthWarning, Text: "Tiles are old"})

	if err := newApplier(runtime).Execute(t.Context(), "org.example.maps", "enabled", true); err != nil {
		t.Fatal(err)
	}
	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "enabled", false); err != nil {
		t.Fatal(err)
	}

	items := runtime.health.List()
	if len(items) != 1 || items[0].PluginID != "org.example.maps" {
		t.Errorf("items = %+v, want only the item of the plugin left enabled", items)
	}
}

func TestAnEnabledPluginGetsItsGeneratedSettingsAndForgetsItsFailure(t *testing.T) {
	runtime := newPluginRuntime()

	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "enabled", true); err != nil {
		t.Fatal(err)
	}

	want := []string{"replace enabled", "fill generated", "clear failure", "refresh listeners", "notify"}
	if !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
}

func TestADisabledPluginThatCannotBeStoppedReportsItAndIsNotNotified(t *testing.T) {
	runtime := newPluginRuntime()
	runtime.stopError = errors.New("stuck")

	err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "enabled", false)

	if !errors.Is(err, runtime.stopError) {
		t.Errorf("error = %v, want the stop error", err)
	}
	if slices.Contains(runtime.events, "notify") {
		t.Errorf("events = %v, want no notification", runtime.events)
	}
}

func TestAnEnabledValueThatIsNotABooleanOnlyNotifiesThePlugin(t *testing.T) {
	runtime := newPluginRuntime()

	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "enabled", "yes"); err != nil {
		t.Fatal(err)
	}

	if want := []string{"replace enabled", "notify"}; !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
}

func TestAValueTheListenersCannotBeSyncedWithIsReplacedByThePreviousOne(t *testing.T) {
	runtime := newPluginRuntime()
	runtime.syncError = errors.New("port in use")

	err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "mode", "b")

	if !errors.Is(err, runtime.syncError) {
		t.Errorf("error = %v, want the sync error", err)
	}
	if runtime.values["mode"] != "a" {
		t.Errorf("mode = %v, want the previous value", runtime.values["mode"])
	}
	want := []string{"replace mode", "sync listeners", "restore mode", "refresh listeners"}
	if !slices.Equal(runtime.events, want) {
		t.Errorf("events = %v, want %v", runtime.events, want)
	}
}

func TestASettingTheListenersCannotBeSyncedWithIsRemovedWhereThereWasNone(t *testing.T) {
	runtime := newPluginRuntime()
	runtime.syncError = errors.New("port in use")

	if err := newApplier(runtime).Execute(t.Context(), "org.example.scans", "port", 80); err == nil {
		t.Error("the sync error was not returned")
	}

	if _, stored := runtime.values["port"]; stored {
		t.Errorf("values = %v, want no port", runtime.values)
	}
}
