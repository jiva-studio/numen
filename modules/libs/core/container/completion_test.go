package container

import (
	"errors"
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The routes are read from the settings file at every question, so a route
// written while the application runs applies to the next one.
func TestARouteWrittenToTheSettingsAppliesToTheNextQuestion(t *testing.T) {
	cfg, path := writeSettings(t, `{}`)
	cfg.PluginHost = NewPluginHost("desktop.editor", t.TempDir())
	router, err := cfg.newCompletionRouter()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := router.Complete(t.Context(), port.Question{Purpose: "proofreading"}); !errors.Is(err, port.ErrNoRoute) {
		t.Fatalf("without routes: error = %v, want port.ErrNoRoute", err)
	}

	routes := `{"ai": {"routes": {"proofreading": {"agent": "md.numen.absent", "model": "m", "concurrency": 2}}}}`
	if err := os.WriteFile(path, []byte(routes), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = router.Complete(t.Context(), port.Question{Purpose: "proofreading"})
	if errors.Is(err, port.ErrNoRoute) || !errors.Is(err, port.ErrUnavailable) {
		t.Errorf("with a route to an absent agent: error = %v, want port.ErrUnavailable and not port.ErrNoRoute", err)
	}
	if _, err := router.Complete(t.Context(), port.Question{Purpose: "summary"}); !errors.Is(err, port.ErrNoRoute) {
		t.Errorf("a purpose without a route or a default: error = %v, want port.ErrNoRoute", err)
	}
}

// A route stands in the settings under ai.routes, with the agent, the model and the limit.
func TestTheRoutesAreReadFromTheAISection(t *testing.T) {
	cfg, _ := writeSettings(t, `{"ai": {"routes": {"default": {"agent": "a", "model": "m", "concurrency": 3}}}}`)
	held, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	route, found := held.AI.Routes["default"]
	if !found || route.Agent != "a" || route.Model != "m" || route.Concurrency != 3 {
		t.Errorf("the default route reads %+v, %v", route, found)
	}
}
