package container

import (
	"context"
	"slices"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// applyingHost records each setting the host applies.
type applyingHost struct {
	pluginport.SettingUpdater
	applied *[]string
}

func (h applyingHost) UpdatePluginSetting(_ context.Context, pluginID, key string, _ any) error {
	*h.applied = append(*h.applied, pluginID+"."+key)
	return nil
}

func TestAPluginChangeIsReportedOnceTheHostAppliedTheSettings(t *testing.T) {
	var applied []string
	c := NewConfig(Config{PluginHost: &PluginHost{SettingUpdater: applyingHost{applied: &applied}}})
	changes, stop := c.WatchPluginChanges()
	defer stop()

	c.settings().NotifyHost([]settingsdomain.Setting{{Path: []string{"org.example.scans", "isAutomatic"}, JSON: "true"}})

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
	c := NewConfig(Config{PluginHost: &PluginHost{SettingUpdater: applyingHost{applied: new([]string)}}})
	changes, stop := c.WatchPluginChanges()
	stop()

	c.settings().NotifyHost(nil)

	select {
	case <-changes:
		t.Error("a stopped watch received a change")
	default:
	}
}
