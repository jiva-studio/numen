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
	agent, err := Find(TypeACP, "  ", fakePath{}.lookPath)
	if !errors.Is(err, ErrNoProgram) || agent.Type != TypeACP || agent.Command != nil {
		t.Errorf("Find = %+v, %v, want Type=%q, nil Command, %v", agent, err, TypeACP, ErrNoProgram)
	}
}

func TestAProgramThatIsNotThereIsNotFound(t *testing.T) {
	emptyLookPath := func(string) (string, error) { return "", nil }
	errWithPath := func(string) (string, error) { return "/opt/orchard-agent", errors.New("permission denied") }
	for _, lookPath := range []func(string) (string, error){fakePath{}.lookPath, nil, emptyLookPath, errWithPath} {
		agent, err := Find(TypeACP, "/opt/orchard-agent", lookPath)
		if !errors.Is(err, ErrNotFound) || agent.Type != TypeACP || agent.Command != nil {
			t.Errorf("Find = %+v, %v, want Type=%q, nil Command, %v", agent, err, TypeACP, ErrNotFound)
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

func TestAKnownAgentStartsWithTheArgumentsOfTheRegistry(t *testing.T) {
	path := fakePath{"kiro-cli": "/usr/bin/kiro-cli", "qwen": "/usr/bin/qwen", "amp-acp": "/usr/bin/amp-acp"}

	for program, want := range map[string][]string{
		"kiro-cli": {"/usr/bin/kiro-cli", "acp", "--agent-engine=v3"},
		"qwen":     {"/usr/bin/qwen", "--acp", "--experimental-skills"},
		"amp-acp":  {"/usr/bin/amp-acp"},
	} {
		agent, err := Find(TypeACP, program, path.lookPath)
		if err != nil || !slices.Equal(agent.Command, want) {
			t.Errorf("Find %s = %v, %v, want %v", program, agent.Command, err, want)
		}
	}
}

func TestTheKnownAgentsHaveOneNameAndOneProgramEach(t *testing.T) {
	names, programs := map[string]bool{}, map[string]bool{}
	for _, one := range ListKnown() {
		if names[one.Name] || programs[one.Program] {
			t.Errorf("%s (%s) is listed twice", one.Name, one.Program)
		}
		names[one.Name], programs[one.Program] = true, true
	}
}
