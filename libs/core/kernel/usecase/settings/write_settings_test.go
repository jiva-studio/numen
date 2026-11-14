package settings_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func newWriter(file *memoryFile, plugins settingsport.PluginSettings, events *[]string) *settings.WriteSettings {
	validator := refusingValidator{declared: [][]string{{"app", "appearance", "theme"}, {"org.example.scans", "isAutomatic"}}}
	return settings.NewWriteSettings(settings.NewCheckSettings(jsonCodec{}, validator, plugins), file, refreshRecorder{events})
}

// The file is writes first, and the readers of the changes are notified after it.
func TestWriteSettingsWritesTheFileThenRefreshesTheEvents(t *testing.T) {
	var events []string
	file := &memoryFile{events: &events}

	err := newWriter(file, none.PluginSettings{}, &events).Execute([]settingsdomain.Setting{{Path: []string{"app", "appearance", "theme"}, JSON: `"preset:night"`}})

	if err != nil || !slices.Equal(events, []string{"write file", "refresh events"}) {
		t.Fatalf("err = %v, events = %v", err, events)
	}
	if len(file.writes) != 1 || file.writes[0].JSON != `"preset:night"` {
		t.Errorf("writes = %+v", file.writes)
	}
}

// A setting no owner declares, a value that is no JSON and a value its plugin refuses all leave the
// file as it was and notify no reader.
func TestWriteSettingsRefusesWhatCannotBeReadBack(t *testing.T) {
	reason := errors.New("not a boolean")
	cases := map[string]struct {
		entry settingsdomain.Setting
		host  settingsport.PluginSettings
	}{
		"undeclared":     {settingsdomain.Setting{Path: []string{"app", "nothing"}, JSON: `1`}, nil},
		"not JSON":       {settingsdomain.Setting{Path: []string{"app", "appearance", "theme"}, JSON: `{`}, nil},
		"plugin refuses": {settingsdomain.Setting{Path: []string{"org.example.scans", "isAutomatic"}, JSON: `"yes"`}, &pluginHost{reason: reason}},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			var events []string
			file := &memoryFile{events: &events}

			err := newWriter(file, one.host, &events).Execute([]settingsdomain.Setting{one.entry})

			if !errors.Is(err, settingsdomain.ErrNotASetting) {
				t.Errorf("err = %v, want ErrNotASetting", err)
			}
			if len(events) != 0 {
				t.Errorf("events = %v, want none", events)
			}
		})
	}
}

// A plugin setting is checked by the plugin that owns it, with the value decoded.
func TestCheckSettingsAsksThePluginThatOwnsTheSetting(t *testing.T) {
	host := &pluginHost{}
	check := settings.NewCheckSettings(jsonCodec{}, refusingValidator{declared: [][]string{{"org.example.scans", "isAutomatic"}}}, host)

	err := check.Execute([]settingsdomain.Setting{{Path: []string{"org.example.scans", "isAutomatic"}, JSON: `true`}})

	if err != nil || !slices.Equal(host.checks, []string{"org.example.scans.isAutomatic"}) {
		t.Fatalf("err = %v, checks = %v", err, host.checks)
	}
}
