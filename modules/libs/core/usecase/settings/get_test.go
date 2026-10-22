package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/modules/libs/core/port/settingsport"
)

var errRead = errors.New("the file cannot be read")

func read() (string, string, error) { return `{"v":1}`, "/home/settings.json", nil }

func TestGetAnswersTheSettingsAndTheirModels(t *testing.T) {
	t.Parallel()
	models := []settingsport.Model{{Name: "base"}}
	bounds := settings.Bounds{Least: 1, Most: 6}
	pluginList := []pluginport.PluginInfo{
		{
			ID:   "md.numen.test",
			Name: "Test Plugin",
			Settings: []pluginport.PluginSetting{
				{Key: "active", Type: "boolean", Title: "Active"},
			},
		},
	}
	get := settings.NewGet(
		read, func() []settingsport.Model { return models }, bounds, func() []pluginport.PluginInfo { return pluginList },
	)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if got.Written != `{"v":1}` || got.Path != "/home/settings.json" {
		t.Errorf("the settings stand as %q at %q", got.Written, got.Path)
	}
	if len(got.Models) != 1 || got.Models[0].Name != "base" {
		t.Errorf("the models are %v", got.Models)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].ID != "md.numen.test" {
		t.Errorf("the plugins are %v", got.Plugins)
	}
	if got.NestedHeadingLimitBounds != bounds {
		t.Errorf("the limits are %v", got.NestedHeadingLimitBounds)
	}
}

func TestGetAnswersNoModelsWhereNoneAreBound(t *testing.T) {
	t.Parallel()
	get := settings.NewGet(read, nil, settings.Bounds{}, nil)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Models) != 0 {
		t.Errorf("the models are %v", got.Models)
	}
	if len(got.Plugins) != 0 {
		t.Errorf("the plugins are %v", got.Plugins)
	}
}

func TestGetRefusesWhenTheFileCannotBeRead(t *testing.T) {
	t.Parallel()
	get := settings.NewGet(
		func() (string, string, error) { return "", "", errRead },
		nil, settings.Bounds{}, nil,
	)

	if _, err := get.Execute(); !errors.Is(err, errRead) {
		t.Fatalf("error is %v, want the read error", err)
	}
}
