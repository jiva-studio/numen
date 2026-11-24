package container

import (
	"context"
	"slices"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/absent"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func applyingHost(applied *[]string) settingsusecase.PluginSettingApplier {
	return func(_ context.Context, pluginID, key string, _ any) error {
		*applied = append(*applied, pluginID+"."+key)
		return nil
	}
}

func TestAPluginChangeIsReportedOnceTheHostAppliedTheSettings(t *testing.T) {
	var applied []string
	c := NewTestInstallation(Config{}, &TestPlugins{
		Execution:   &PluginExecution{Runtime: pluginRuntime{pluginBinding: absent.Runtime{}}},
		Preferences: &PluginPreferences{ApplySetting: applyingHost(&applied)},
	})
	changes, stop := c.Resources.subscriptions.Changes.Subscribe()
	defer stop()

	notifyHost(c, []settingsdomain.Setting{{Path: []string{"org.example.scans", "isAutomatic"}, JSON: "true"}})

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
	c := NewTestInstallation(Config{}, &TestPlugins{
		Execution:   &PluginExecution{Runtime: pluginRuntime{pluginBinding: absent.Runtime{}}},
		Preferences: &PluginPreferences{ApplySetting: applyingHost(new([]string))},
	})
	changes, stop := c.Resources.subscriptions.Changes.Subscribe()
	stop()

	notifyHost(c, nil)

	select {
	case <-changes:
		t.Error("a stopped watch received a change")
	default:
	}
}
