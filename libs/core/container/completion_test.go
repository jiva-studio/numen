package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

// model and the limit.
func TestTheDefaultIsReadFromTheAISection(t *testing.T) {
	cfg, _ := writeSettings(t, `{"ai": {"default": {"agent": "a", "model": "m", "concurrency": 3}}}`)
	loaded, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if want := (settings.AIRoute{Agent: "a", Model: "m", Concurrency: 3}); loaded.AI.Default != want {
		t.Errorf("the default reads %+v, want %+v", loaded.AI.Default, want)
	}
}

// A file written with routes by purpose loads, and keeps its default route.
func TestAFileWithRoutesByPurposeLoads(t *testing.T) {
	cfg, _ := writeSettings(t, `{"ai": {"routes": {"default": {"agent": "a", "model": "m"}, "proofreading": {"agent": "b", "concurrency": 2}}}}`)
	loaded, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if want := (settings.AIRoute{Agent: "a", Model: "m"}); loaded.AI.Default != want {
		t.Errorf("the default reads %+v, want %+v", loaded.AI.Default, want)
	}
}
