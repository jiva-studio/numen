package settings_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

var errRead = errors.New("the file cannot be read")

func read() (string, string, error) { return `{"v":1}`, "/home/settings.json", nil }

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
		read, func() []plugin.Info { return pluginList }, nil,
	)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
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
	get := settings.NewGet(read, nil, nil)

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
		func() (string, string, error) { return "", "", errRead },
		nil, nil,
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
	)

	got, err := get.Execute()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.App) != 1 || got.App[0].Key != "appearance.mode" {
		t.Errorf("the declaration of the shell is %v", got.App)
	}
}
