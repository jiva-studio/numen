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
	s.applyConfig(map[string]string{"executablePath": "/usr/bin/my-acp"})
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
			"acp:gemini":  {"/custom/gemini"},
			"acp:aliased": {"/opt/bin/codex-acp"},
		},
		lookPath: fakePath{"gemini": "/usr/local/bin/gemini", "codex-acp": "/opt/bin/codex-acp", "qwen": "/usr/local/bin/qwen"}.lookPath,
	}
	s.applyConfig(map[string]string{
		"executablePath": "/opt/bin/codex-acp\n/usr/bin/unknown-acp",
	})
	agents := s.getACPAgents()
	var names []string
	for _, a := range agents {
		names = append(names, a.Program)
	}
	wantNames := []string{"aliased", "gemini", "qwen", "unknown-acp"}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("got programs %v, want %v (agents: %+v)", names, wantNames, agents)
	}
	if !slices.Equal(agents[1].Command, []string{"/custom/gemini"}) {
		t.Errorf("gemini command = %v, want [/custom/gemini]", agents[1].Command)
	}
	if !slices.Equal(agents[2].Command, []string{"/usr/local/bin/qwen", "--acp"}) {
		t.Errorf("qwen command = %v, want [/usr/local/bin/qwen --acp]", agents[2].Command)
	}
	if agents[3].Title != "unknown-acp" || !slices.Equal(agents[3].Command, []string{"/usr/bin/unknown-acp"}) {
		t.Errorf("unknown-acp = %+v", agents[3])
	}

	s2 := &agentServer{}
	s2.applyConfig(map[string]string{
		"executablePath": "/usr/bin/claude-code-acp\n/usr/bin/gemini",
	})
	agents2 := s2.getACPAgents()
	if len(agents2) != 2 || agents2[0].Program != "claude-code" || !slices.Equal(agents2[0].Command, []string{"/usr/bin/claude-code-acp"}) ||
		agents2[1].Program != "gemini" || !slices.Equal(agents2[1].Command, []string{"/usr/bin/gemini", "--acp"}) {
		t.Errorf("got %+v", agents2)
	}
}
