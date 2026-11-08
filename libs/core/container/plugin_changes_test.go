package container

import (
	"context"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// applyingHost records each setting the host applies.
type applyingHost struct {
	pluginport.PluginManager
	applied *[]string
}

func (h applyingHost) UpdatePluginSetting(_ context.Context, pluginID, key string, _ any) error {
	*h.applied = append(*h.applied, pluginID+"."+key)
	return nil
}

func TestAPluginChangeIsReportedOnceTheHostAppliedTheSettings(t *testing.T) {
	var applied []string
	c := NewConfig(Config{PluginHost: applyingHost{applied: &applied}})
	changes, stop := c.WatchPluginChanges()
	defer stop()

	c.settings().NotifyHost([]settingsport.Setting{{Path: []string{"org.example.scans", "isAutomatic"}, JSON: "true"}})

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
	c := NewConfig(Config{PluginHost: applyingHost{applied: new([]string)}})
	changes, stop := c.WatchPluginChanges()
	stop()

	c.settings().NotifyHost(nil)

	select {
	case <-changes:
		t.Error("a stopped watch received a change")
	default:
	}
}
