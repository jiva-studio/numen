package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
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

// What puts the tools on a port: an agent named for the panel, or a person
// asking for the port itself. An installation asking for neither opens none.
func TestWhatOpensThePort(t *testing.T) {
	if agent.Defaults().ShouldServeTools {
		t.Error("the defaults serve the tools to an agent nobody has configured")
	}

	for name, said := range map[string]struct {
		file      string
		isServing bool
	}{
		"the defaults":              {`{}`, true},
		"no agent named":            {`{"use": ""}`, false},
		"no agent, tools asked for": {`{"use": "", "serve_tools": true}`, true},
		"an agent named":            {`{"use": "claude"}`, true},
	} {
		t.Run(name, func(t *testing.T) {
			held := agent.Defaults()
			if err := json.Unmarshal([]byte(said.file), &held); err != nil {
				t.Fatal(err)
			}
			if got := held.IsServingTools(); got != said.isServing {
				t.Errorf("the tools are served: %v", got)
			}
		})
	}
}
