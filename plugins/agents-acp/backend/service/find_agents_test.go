package service

import (
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

func getCommands(agents []acpAgent) [][]string {
	var out [][]string
	for _, one := range agents {
		out = append(out, one.Command)
	}
	return out
}

func TestThePluginFindsTheAgentsOnThePathWhenItStarts(t *testing.T) {
	path := fakePath{"gemini": "/usr/local/bin/gemini", "codex-acp": "/opt/bin/codex-acp"}

	agents := newAgentServer(path.lookPath).getACPAgents()

	want := [][]string{{"/opt/bin/codex-acp"}, {"/usr/local/bin/gemini", "--acp"}}
	if got := getCommands(agents); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("agents = %+v, want the commands %v", agents, want)
	}
}

func TestThePluginFindsTheAgentsAgainWhenItsConfigChanges(t *testing.T) {
	path := fakePath{}
	s := newAgentServer(path.lookPath)
	path["qwen"] = "/usr/local/bin/qwen"

	s.applyConfig(map[string]string{})

	want := [][]string{{"/usr/local/bin/qwen", "--acp"}}
	if got := getCommands(s.getACPAgents()); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("agents = %v, want %v", got, want)
	}
}
