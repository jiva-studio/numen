package service

import (
	"slices"
	"testing"
)

func TestTheACPAgentsAreTheProgramsOfTheSettingsWithTheACPPrefix(t *testing.T) {
	for range 20 {
		got := getACPAgentsFromCommands(map[string][]string{
			"acp:zed":   {"/bin/zed", "acp"},
			"acp:alpha": {"/bin/alpha"},
			"acp:none":  nil,
			"acp:":      {"/bin/empty"},
			"plain":     {"/bin/plain"},
		})

		var programs []string
		for _, agent := range got {
			programs = append(programs, agent.Program)
		}
		if want := []string{"alpha", "zed"}; !slices.Equal(programs, want) {
			t.Fatalf("got %v, want %v", programs, want)
		}
		if got[1].Title != "zed" || !slices.Equal(got[1].Command, []string{"/bin/zed", "acp"}) {
			t.Errorf("got %+v", got[1])
		}
	}
}

func TestNoACPAgentIsNamedWithoutCommands(t *testing.T) {
	if got := getACPAgentsFromCommands(nil); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestACPAgentFromConfig(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"executable_path": "/usr/bin/my-acp"})
	agents := s.getACPAgents()
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Program != "acp" || agents[0].Title != "ACP Agent" || agents[0].Command[0] != "/usr/bin/my-acp" {
		t.Errorf("got %+v", agents[0])
	}
}

func TestGetExecutableProgramNameHandlesEmptyStem(t *testing.T) {
	if got := getExecutableProgramName(".exe"); got != defaultACPProgram {
		t.Errorf("got %q, want %q", got, defaultACPProgram)
	}
}

func TestMultipleACPAgentsFromConfigAndAutoDiscovery(t *testing.T) {
	s := &agentServer{
		commands: map[string][]string{
			"acp:gemini": {"/custom/gemini"},
			"acp:aliased": {"/opt/bin/codex-acp"},
		},
		discover: func() []acpAgent {
			discovered := discoverACPAgentsWithLookup(func(binary string) (string, bool) {
				switch binary {
				case "gemini":
					return "/usr/local/bin/gemini", true
				case "codex-acp":
					return "/opt/bin/codex-acp", true
				case "qwen":
					return "/usr/local/bin/qwen", true
				}
				return "", false
			})
			return append([]acpAgent{
				{Program: "empty-a", Title: "Empty A", Command: nil},
				{Program: "empty-b", Title: "Empty B", Command: nil},
			}, discovered...)
		},
	}
	s.applyConfig(map[string]string{
		"executable_path": "/opt/bin/codex-acp\n/usr/bin/unknown-acp",
	})
	agents := s.getACPAgents()
	var names []string
	for _, a := range agents {
		names = append(names, a.Program)
	}
	wantNames := []string{"aliased", "empty-a", "empty-b", "gemini", "qwen", "unknown-acp"}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("got programs %v, want %v (agents: %+v)", names, wantNames, agents)
	}
	if !slices.Equal(agents[3].Command, []string{"/custom/gemini"}) {
		t.Errorf("gemini command = %v, want [/custom/gemini]", agents[3].Command)
	}
	if !slices.Equal(agents[4].Command, []string{"/usr/local/bin/qwen", "--acp"}) {
		t.Errorf("qwen command = %v, want [/usr/local/bin/qwen --acp]", agents[4].Command)
	}
	if agents[5].Title != "unknown-acp" || !slices.Equal(agents[5].Command, []string{"/usr/bin/unknown-acp"}) {
		t.Errorf("unknown-acp = %+v", agents[5])
	}

	s2 := &agentServer{}
	s2.applyConfig(map[string]string{
		"executable_path": "/usr/bin/claude-code-acp\n/usr/bin/gemini",
	})
	agents2 := s2.getACPAgents()
	if len(agents2) != 2 || agents2[0].Program != "claude-code" || !slices.Equal(agents2[0].Command, []string{"/usr/bin/claude-code-acp"}) ||
		agents2[1].Program != "gemini" || !slices.Equal(agents2[1].Command, []string{"/usr/bin/gemini", "--acp"}) {
		t.Errorf("got %+v", agents2)
	}
}

func discoverACPAgentsWithLookup(lookup func(binary string) (string, bool)) []acpAgent {
	var agents []acpAgent
	for _, candidate := range getKnownACPAgents() {
		binaryPath, ok := lookup(candidate.binary)
		if !ok || binaryPath == "" {
			continue
		}
		command := make([]string, 0, 1+len(candidate.args))
		command = append(command, binaryPath)
		command = append(command, candidate.args...)
		agents = append(agents, acpAgent{
			Program: candidate.program,
			Title:   candidate.title,
			Command: command,
		})
	}
	return agents
}
