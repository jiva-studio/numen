package agents

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

func getCommands(agents []Agent) [][]string {
	var out [][]string
	for _, one := range agents {
		out = append(out, one.Command)
	}
	return out
}

func TestTheAgentsAreTheProgramsOfTheSettingsWithTheACPPrefix(t *testing.T) {
	for range 20 {
		got := GetFromCommands(map[string][]string{
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

func TestNoAgentIsNamedWithoutCommands(t *testing.T) {
	if got := GetFromCommands(nil); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestGetProgramNameHandlesEmptyStem(t *testing.T) {
	if got := GetProgramName(".exe"); got != defaultProgram {
		t.Errorf("got %q, want %q", got, defaultProgram)
	}
}

func TestTheKnownAgentsAreFoundOnThePath(t *testing.T) {
	path := fakePath{"gemini": "/usr/local/bin/gemini", "codex-acp": "/opt/bin/codex-acp"}

	agents := FindInstalled(path.lookPath)

	want := [][]string{{"/opt/bin/codex-acp"}, {"/usr/local/bin/gemini", "--acp"}}
	if got := getCommands(agents); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("agents = %+v, want the commands %v", agents, want)
	}
}

func TestNoAgentIsFoundWithoutAPathLookup(t *testing.T) {
	if got := FindInstalled(nil); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestExecutablePathsAreOneOnALine(t *testing.T) {
	got := ParsePaths("/opt/bin/first-acp\n  \n /opt/bin/second-acp \n")

	if want := []string{"/opt/bin/first-acp", "/opt/bin/second-acp"}; !slices.Equal(got, want) {
		t.Errorf("executable paths = %v, want %v", got, want)
	}
}
