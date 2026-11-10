package container

import (
	"context"
	"slices"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

func applyingHost(applied *[]string) settingsport.PluginSettingApplier {
	return func(_ context.Context, pluginID, key string, _ any) error {
		*applied = append(*applied, pluginID+"."+key)
		return nil
	}
}

func TestAPluginChangeIsReportedOnceTheHostAppliedTheSettings(t *testing.T) {
	var applied []string
	c := NewConfig(Config{PluginHost: &PluginHost{ApplySetting: applyingHost(&applied)}})
	changes, stop := c.WatchPluginChanges()
	defer stop()

	notifyHost(c.settings(), []settingsdomain.Setting{{Path: []string{"org.example.scans", "isAutomatic"}, JSON: "true"}})

	select {
	case <-changes:
		if !slices.Equal(applied, []string{"org.example.scans.isAutomatic"}) {
			t.Errorf("reported with %v applied, want the setting applied first", applied)
		}
	default:
		t.Error("the settings applied reported no change")
	}
}

func TestAPluginChangeIsNotReportedAfterTheWatchStops(t *testing.T) {
	c := NewConfig(Config{PluginHost: &PluginHost{ApplySetting: applyingHost(new([]string))}})
	changes, stop := c.WatchPluginChanges()
	stop()

	notifyHost(c.settings(), nil)

	select {
	case <-changes:
		t.Error("a stopped watch received a change")
	default:
	}
}
