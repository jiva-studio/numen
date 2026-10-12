package agent_test

import (
	"encoding/json"
	"slices"
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
		"antigravity named":         {`{"use": "antigravity"}`, true},
		"codex named":               {`{"use": "codex"}`, true},
		"an unknown agent named":    {`{"use": "other"}`, false},
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

// A section a file leaves empty keeps the step limit the defaults give it, for
// each agent alike.
func TestAnEmptySectionKeepsItsStepLimit(t *testing.T) {
	held := agent.Defaults()
	if err := json.Unmarshal([]byte(`{"claude": {}, "antigravity": {}, "codex": {}}`), &held); err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]int{
		"claude":      held.Claude.MaxSteps,
		"antigravity": held.Antigravity.MaxSteps,
		"codex":       held.Codex.MaxSteps,
	} {
		if got != 100 {
			t.Errorf("%s may go to the model %d times", name, got)
		}
	}
}

// A section of the wrong type is a settings file that cannot be read.
func TestASectionOfTheWrongTypeIsRefused(t *testing.T) {
	for _, file := range []string{
		`{"claude": 5}`,
		`{"antigravity": 5}`,
		`{"codex": 5}`,
		`{"claude": {"max_steps": "many"}}`,
		`{"codex": {"command": 3}}`,
	} {
		held := agent.Defaults()
		if err := json.Unmarshal([]byte(file), &held); err == nil {
			t.Errorf("%s was read", file)
		}
	}
}

// Every field a section names is taken over the defaults, for each agent alike.
func TestTheFieldsASectionNamesAreTaken(t *testing.T) {
	held := agent.Defaults()
	file := `{
		"claude": {"command": ["c", "--x"], "model": "opus", "max_steps": 7, "reads_hooks_and_skills": true},
		"antigravity": {"command": ["a"], "model": "gem", "max_steps": 8, "reads_hooks_and_skills": true},
		"codex": {"command": ["x", "y"], "model": "gpt", "max_steps": 9, "reads_hooks_and_skills": true}
	}`
	if err := json.Unmarshal([]byte(file), &held); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(held.Claude.Command, []string{"c", "--x"}) || held.Claude.Model != "opus" ||
		held.Claude.MaxSteps != 7 || !held.Claude.ShouldReadHooksAndSkills {
		t.Errorf("claude: %+v", held.Claude)
	}
	if !slices.Equal(held.Antigravity.Command, []string{"a"}) || held.Antigravity.Model != "gem" ||
		held.Antigravity.MaxSteps != 8 || !held.Antigravity.ShouldReadHooksAndSkills {
		t.Errorf("antigravity: %+v", held.Antigravity)
	}
	if !slices.Equal(held.Codex.Command, []string{"x", "y"}) || held.Codex.Model != "gpt" ||
		held.Codex.MaxSteps != 9 || !held.Codex.ShouldReadHooksAndSkills {
		t.Errorf("codex: %+v", held.Codex)
	}
}
