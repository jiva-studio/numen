package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

// model and the limit.
func TestTheDefaultIsReadFromTheAISection(t *testing.T) {
	cfg, _ := writeSettings(t, `{"app": {"ai": {"default": {"agent": "a", "model": "m", "concurrency": 3}}}}`)
	loaded, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if want := (settings.AIRoute{Agent: "a", Model: "m", Concurrency: 3}); loaded.AI.Default != want {
		t.Errorf("the default reads %+v, want %+v", loaded.AI.Default, want)
	}
}
