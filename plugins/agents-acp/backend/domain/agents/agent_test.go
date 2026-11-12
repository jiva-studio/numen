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

func TestTheProgramIsFoundWithoutTheSpacesAroundIt(t *testing.T) {
	path := fakePath{"/home/ada/bin/orchard-agent": "/home/ada/bin/orchard-agent"}

	agent, err := Find(TypeACP, " /home/ada/bin/orchard-agent ", path.lookPath)

	if err != nil || !slices.Equal(agent.Command, []string{"/home/ada/bin/orchard-agent"}) {
		t.Errorf("Find = %v, %v", agent.Command, err)
	}
}

func TestAnACPProgramOfAKnownNameIsStartedWithItsArguments(t *testing.T) {
	path := fakePath{"/opt/gemini": "/opt/gemini", "opencode": "/usr/bin/opencode", "/opt/orchard-agent": "/opt/orchard-agent"}

	for program, want := range map[string][]string{
		"/opt/gemini":        {"/opt/gemini", "--acp"},
		"opencode":           {"/usr/bin/opencode", "acp"},
		"/opt/orchard-agent": {"/opt/orchard-agent"},
	} {
		agent, err := Find(TypeACP, program, path.lookPath)
		if err != nil || !slices.Equal(agent.Command, want) {
			t.Errorf("Find %s = %v, %v, want %v", program, agent.Command, err, want)
		}
	}
}

func TestAnACPConnectionRequiresAProgram(t *testing.T) {
	if _, err := Find(TypeACP, "  ", fakePath{}.lookPath); !errors.Is(err, ErrNoProgram) {
		t.Errorf("Find = %v, want %v", err, ErrNoProgram)
	}
}

func TestAProgramThatIsNotThereIsNotFound(t *testing.T) {
	for _, lookPath := range []func(string) (string, error){fakePath{}.lookPath, nil} {
		if _, err := Find(TypeACP, "/opt/orchard-agent", lookPath); !errors.Is(err, ErrNotFound) {
			t.Errorf("Find = %v, want %v", err, ErrNotFound)
		}
	}
}

func TestTheKeyTellsTheProgramsApart(t *testing.T) {
	one := Agent{Type: TypeACP, Command: []string{"/opt/gemini", "--acp"}}
	other := Agent{Type: TypeACP, Command: []string{"/opt/gemini"}}
	if one.GetKey() == other.GetKey() {
		t.Error("two command lines share a key")
	}
}
