package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent"
)

// A settings file that names no agent names no agent. The defaults fill in what
// a file leaves out, and an empty name is what a person wrote.
func TestNamingNoAgentIsKept(t *testing.T) {
	held := agent.Defaults()
	if held.Use != agent.UseClaude {
		t.Fatalf("the defaults answer with %q", held.Use)
	}
	if err := json.Unmarshal([]byte(`{"use": ""}`), &held); err != nil {
		t.Fatal(err)
	}
	if held.Use != "" {
		t.Errorf("a file naming no agent answers with %q", held.Use)
	}
	if held.Claude.MaxSteps != agent.Defaults().Claude.MaxSteps {
		t.Errorf("the section the file left out lost its defaults: %+v", held.Claude)
	}
}
