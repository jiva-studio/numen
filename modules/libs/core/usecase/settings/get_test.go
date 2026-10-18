package settings_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

var errRead = errors.New("the file cannot be read")

func read() (string, string, error) { return `{"v":1}`, "/home/settings.json", nil }

func TestGetAnswersTheSettingsTheirModelsAndTheDay(t *testing.T) {
	t.Parallel()
	morning := time.Date(2026, 9, 5, 1, 0, 0, 0, time.UTC)
	models := []port.Model{{Name: "base"}}
	bounds := settings.Bounds{Least: 1, Most: 6}
	pluginList := []port.PluginInfo{
		{
			ID:   "md.numen.test",
			Name: "Test Plugin",
			Settings: []port.PluginSetting{
				{Key: "active", Type: "boolean", Title: "Active"},
			},
		},
	}
	get := settings.NewGet(
		read, func() []port.Model { return models }, bounds, "08:00",
		review.Day{Starts: review.DayStartTime, In: time.UTC}, func() time.Time { return morning },
		func() []port.PluginInfo { return pluginList },
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
	if got.NestedHeadingLimitBounds != bounds || got.LatestDayStartTime != "08:00" {
		t.Errorf("the limits are %v and %q", got.NestedHeadingLimitBounds, got.LatestDayStartTime)
	}
	if got.Day != "2026-09-04" {
		t.Errorf("an hour past midnight is the day %q", got.Day)
	}
}

func TestGetAnswersNoModelsWhereNoneAreBound(t *testing.T) {
	t.Parallel()
	get := settings.NewGet(read, nil, settings.Bounds{}, "", review.Day{In: time.UTC}, time.Now, nil)

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
		nil, settings.Bounds{}, "", review.Day{In: time.UTC}, time.Now, nil,
	)

	if _, err := get.Execute(); !errors.Is(err, errRead) {
		t.Fatalf("error is %v, want the read error", err)
	}
}
