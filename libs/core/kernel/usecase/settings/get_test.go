package settings_test

import (
	"errors"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

var errRead = errors.New("the file cannot be read")

func read() (settingsdomain.Snapshot, error) {
	return settingsdomain.Snapshot{Content: `{"v":1}`, Path: "/home/settings.json", Version: 3}, nil
}

func TestGetAnswersTheSettingsAndThePlugins(t *testing.T) {
	t.Parallel()
	pluginList := []plugin.Info{
		{
			ID:   "md.numen.test",
			Name: "Test Plugin",
			Settings: []plugin.Setting{
				{Key: "active", Type: "boolean", Title: "Active"},
			},
		},
	}
	get := settings.NewGet(
		read, func() []plugin.Info { return pluginList }, nil, nil,
	)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if got.Version != 3 {
		t.Errorf("the version is %d, want 3", got.Version)
	}
	if got.Written != `{"v":1}` || got.Path != "/home/settings.json" {
		t.Errorf("the settings stand as %q at %q", got.Written, got.Path)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].ID != "md.numen.test" {
		t.Errorf("the plugins are %v", got.Plugins)
	}
}

func TestGetAnswersNoPluginsWhereNoneAreBound(t *testing.T) {
	t.Parallel()
	get := settings.NewGet(read, nil, nil, nil)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Plugins) != 0 {
		t.Errorf("the plugins are %v", got.Plugins)
	}
}

func TestGet_ErrorWhenFileUnreadable(t *testing.T) {
	t.Parallel()
	get := settings.NewGet(
		func() (settingsdomain.Snapshot, error) { return settingsdomain.Snapshot{}, errRead },
		nil, nil, nil,
	)

	if _, err := get.Execute(); !errors.Is(err, errRead) {
		t.Fatalf("error is %v, want the read error", err)
	}
}

func TestGetAnswersTheDeclarationOfTheShell(t *testing.T) {
	t.Parallel()
	get := settings.NewGet(
		read, nil,
		func() []plugin.Setting {
			return []plugin.Setting{{Key: "appearance.mode", Type: "enum", Title: "Mode"}}
		},
		func() []manifest.SettingGroup { return []manifest.SettingGroup{{ID: "a", Title: "A"}} },
	)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.App) != 1 || got.App[0].Key != "appearance.mode" {
		t.Errorf("the declaration of the shell is %v", got.App)
	}
	if len(got.AppGroups) != 1 || got.AppGroups[0].ID != "a" {
		t.Errorf("the groups of the shell are %v", got.AppGroups)
	}
}
