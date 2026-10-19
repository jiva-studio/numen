package container

import (
	"errors"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
)

// The default is read from the settings file at every question, so a default
// written while the application runs applies to the next one; an agent that is
// not installed is no route, and no other agent answers in its place.
func TestADefaultWrittenToTheSettingsAppliesToTheNextQuestion(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	cfg.PluginHost = NewPluginHost("desktop.editor", t.TempDir())
	router, err := cfg.newCompletionRouter()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := router.Complete(t.Context(), "md.numen.asking", port.Question{}); !errors.Is(err, port.ErrNoRoute) {
		t.Fatalf("without a provider: error = %v, want port.ErrNoRoute", err)
	}

	written := `{"ai": {"default": {"agent": "md.numen.absent", "model": "m", "concurrency": 2}}}`
	if err := os.WriteFile(path, []byte(written), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := router.Complete(t.Context(), "md.numen.asking", port.Question{}); !errors.Is(err, port.ErrNoRoute) {
		t.Errorf("with a default that is not installed: error = %v, want port.ErrNoRoute", err)
	}
}

// The default stands in the settings under ai.default, with the agent, the
// model and the limit.
func TestTheDefaultIsReadFromTheAISection(t *testing.T) {
	cfg, _ := writeSettings(t, `{"ai": {"default": {"agent": "a", "model": "m", "concurrency": 3}}}`)
	loaded, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if want := (completion.Route{Agent: "a", Model: "m", Concurrency: 3}); loaded.AI.Default != want {
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
	if want := (completion.Route{Agent: "a", Model: "m"}); loaded.AI.Default != want {
		t.Errorf("the default reads %+v, want %+v", loaded.AI.Default, want)
	}
}
