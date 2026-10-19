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
