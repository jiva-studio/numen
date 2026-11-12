package agents

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
)

// fakePath is a PATH that holds the programs of its keys.
type fakePath map[string]string

func (p fakePath) lookPath(name string) (string, error) {
	if found, ok := p[name]; ok {
		return found, nil
	}
	return "", errors.New("not found")
}

func TestATypeWithoutAProgramFindsItsKnownBinaryOnThePath(t *testing.T) {
	path := fakePath{
		"claude-code-acp": "/opt/bin/claude-code-acp",
		"codex-acp":       "/opt/bin/codex-acp",
		"agy":             "/usr/local/bin/agy",
	}
	for connectionType, want := range map[string][]string{
		TypeClaudeCode:  {"/opt/bin/claude-code-acp"},
		TypeCodex:       {"/opt/bin/codex-acp"},
		TypeAntigravity: {"/usr/local/bin/agy", "acp"},
	} {
		agent, err := Find(connectionType, "", path.lookPath)
		if err != nil || !slices.Equal(agent.Command, want) {
			t.Errorf("%s: Find = %v, %v, want %v", connectionType, agent.Command, err, want)
		}
	}
}

func TestClaudeCodePrefersTheAgentSDKAdapter(t *testing.T) {
	path := fakePath{"claude-code-acp": "/opt/bin/claude-code-acp", "claude-agent-acp": "/opt/bin/claude-agent-acp"}

	agent, err := Find(TypeClaudeCode, "", path.lookPath)

	if err != nil || !slices.Equal(agent.Command, []string{"/opt/bin/claude-agent-acp"}) {
		t.Errorf("Find = %v, %v", agent.Command, err)
	}
}

func TestTheProgramFieldWinsOverTheKnownBinary(t *testing.T) {
	path := fakePath{"agy": "/usr/local/bin/agy", "/home/ada/bin/agy-nightly": "/home/ada/bin/agy-nightly"}

	agent, err := Find(TypeAntigravity, " /home/ada/bin/agy-nightly ", path.lookPath)

	if err != nil || !slices.Equal(agent.Command, []string{"/home/ada/bin/agy-nightly", "acp"}) {
		t.Errorf("Find = %v, %v", agent.Command, err)
	}
}

func TestAnACPProgramOfAKnownNameIsStartedWithItsArguments(t *testing.T) {
	path := fakePath{"/opt/gemini": "/opt/gemini", "/opt/orchard-agent": "/opt/orchard-agent"}

	gemini, err := Find(TypeACP, "/opt/gemini", path.lookPath)
	if err != nil || !slices.Equal(gemini.Command, []string{"/opt/gemini", "--acp"}) {
		t.Errorf("Find gemini = %v, %v", gemini.Command, err)
	}
	other, err := Find(TypeACP, "/opt/orchard-agent", path.lookPath)
	if err != nil || !slices.Equal(other.Command, []string{"/opt/orchard-agent"}) {
		t.Errorf("Find another = %v, %v", other.Command, err)
	}
}

func TestAnACPConnectionRequiresAProgram(t *testing.T) {
	if _, err := Find(TypeACP, "  ", fakePath{}.lookPath); !errors.Is(err, ErrNoProgram) {
		t.Errorf("Find = %v, want %v", err, ErrNoProgram)
	}
}

func TestAProgramThatIsNotThereIsNotFound(t *testing.T) {
	for _, lookPath := range []func(string) (string, error){fakePath{}.lookPath, nil} {
		if _, err := Find(TypeCodex, "", lookPath); !errors.Is(err, ErrNotFound) {
			t.Errorf("Find = %v, want %v", err, ErrNotFound)
		}
	}
}

func TestClaudeCodeRunsATaskWithWebSearchAndTheServedToolsAndNoSettings(t *testing.T) {
	raw, err := json.Marshal(Agent{Type: TypeClaudeCode, Command: nil}.GetTaskMeta([]string{"note_read", "garden_plan"}))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"claudeCode":{"options":{"allowedTools":["WebSearch","mcp__numen__garden_plan","mcp__numen__note_read"],` +
		`"maxTurns":100,"settingSources":[],"tools":["WebSearch"]}}}`
	if string(raw) != want {
		t.Errorf("meta = %s, want %s", raw, want)
	}
}

func TestClaudeCodeAnswersWithNoToolOfItsOwn(t *testing.T) {
	raw, err := json.Marshal(Agent{Type: TypeClaudeCode, Command: nil}.GetAnswerMeta())
	if err != nil {
		t.Fatal(err)
	}

	if want := `{"claudeCode":{"options":{"maxTurns":1,"settingSources":[],"tools":[]}}}`; string(raw) != want {
		t.Errorf("meta = %s, want %s", raw, want)
	}
}

func TestOtherTypesOpenSessionsWithNoMeta(t *testing.T) {
	for _, connectionType := range []string{TypeACP, TypeCodex, TypeAntigravity} {
		agent := Agent{Type: connectionType, Command: nil}
		if agent.GetTaskMeta([]string{"note_read"}) != nil || agent.GetAnswerMeta() != nil {
			t.Errorf("%s has a meta", connectionType)
		}
	}
}
