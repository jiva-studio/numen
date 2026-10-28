package service

import (
	"slices"
	"testing"
)

func TestTheACPAgentsAreTheProgramsOfTheSettingsWithTheACPPrefix(t *testing.T) {
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
	if agents[0].Program != "acp" || agents[0].Command[0] != "/usr/bin/my-acp" {
		t.Errorf("got %+v", agents[0])
	}
}

func TestMultipleACPAgentsFromConfigAndAutoDiscovery(t *testing.T) {
	s := &agentServer{
		discover: func() []acpAgent {
			return discoverACPAgentsWithLookup(func(binary string) (string, bool) {
				if binary == "gemini" {
					return "/usr/local/bin/gemini", true
				}
				return "", false
			})
		},
	}
	s.applyConfig(map[string]string{
		"executable_path": "/usr/bin/claude-code-acp\n/opt/bin/codex-acp",
	})
	agents := s.getACPAgents()
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d: %+v", len(agents), agents)
	}
	if agents[0].Program != "claude-code-acp" || agents[1].Program != "codex-acp" || agents[2].Program != "gemini" {
		t.Errorf("unexpected agents: %+v", agents)
	}
}
