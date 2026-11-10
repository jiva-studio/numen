package agents

import (
	"errors"
	"slices"
	"testing"
)

func TestTheAgentOfAnExecutableInTheSettingsIsNamedAcp(t *testing.T) {
	registry := NewRegistry(nil)
	registry.SetConfig(map[string]string{"executablePath": "/usr/bin/my-acp"})

	agents := registry.List()

	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Program != "acp" || agents[0].Title != "ACP Agent" || agents[0].Command[0] != "/usr/bin/my-acp" {
		t.Errorf("got %+v", agents[0])
	}
}

func TestAgentsComeFromTheSettingsTheExecutablesAndThePath(t *testing.T) {
	registry := NewRegistry(fakePath{"gemini": "/usr/local/bin/gemini", "codex-acp": "/opt/bin/codex-acp", "qwen": "/usr/local/bin/qwen"}.lookPath)
	registry.SetCommands(map[string][]string{
		"acp:gemini":  {"/custom/gemini"},
		"acp:aliased": {"/opt/bin/codex-acp"},
	})
	registry.SetConfig(map[string]string{"executablePath": "/opt/bin/codex-acp\n/usr/bin/unknown-acp"})

	agents := registry.List()

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
}

func TestSeveralExecutablesOfKnownAgentsKeepTheirPrograms(t *testing.T) {
	registry := NewRegistry(nil)
	registry.SetConfig(map[string]string{"executablePath": "/usr/bin/claude-code-acp\n/usr/bin/gemini"})

	agents := registry.List()

	if len(agents) != 2 || agents[0].Program != "claude-code" || !slices.Equal(agents[0].Command, []string{"/usr/bin/claude-code-acp"}) ||
		agents[1].Program != "gemini" || !slices.Equal(agents[1].Command, []string{"/usr/bin/gemini", "--acp"}) {
		t.Errorf("got %+v", agents)
	}
}

func TestTheAgentsAreFoundAgainWhenTheConfigChanges(t *testing.T) {
	path := fakePath{}
	registry := NewRegistry(path.lookPath)
	path["qwen"] = "/usr/local/bin/qwen"

	registry.SetConfig(map[string]string{})

	want := [][]string{{"/usr/local/bin/qwen", "--acp"}}
	if got := getCommands(registry.List()); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("agents = %v, want %v", got, want)
	}
}

func TestExecutablePathsAreReadFromTheCamelCaseKey(t *testing.T) {
	registry := NewRegistry(nil)
	registry.SetConfig(map[string]string{"executablePath": "/opt/bin/first-acp\n/opt/bin/second-acp"})

	if want := []string{"/opt/bin/first-acp", "/opt/bin/second-acp"}; !slices.Equal(registry.paths, want) {
		t.Errorf("executable paths = %v, want %v", registry.paths, want)
	}
}

func TestExecutablePathsAreNotReadFromTheSnakeCaseKey(t *testing.T) {
	registry := NewRegistry(nil)
	registry.SetConfig(map[string]string{"executable_path": "/opt/bin/first-acp"})

	if len(registry.paths) != 0 {
		t.Errorf("executable paths = %v, want none", registry.paths)
	}
}

func TestAProgramWithACommandIsFoundWithoutTheSettings(t *testing.T) {
	found, err := NewRegistry(nil).Find("acp:single", []string{"/bin/single"})

	if err != nil || found.Program != "single" || !slices.Equal(found.Command, []string{"/bin/single"}) {
		t.Errorf("found %+v, %v", found, err)
	}
}

func TestAProgramIsFoundByItsNameInTheList(t *testing.T) {
	registry := NewRegistry(fakePath{"qwen": "/usr/local/bin/qwen"}.lookPath)

	found, err := registry.Find("acp:qwen", nil)

	if err != nil || !slices.Equal(found.Command, []string{"/usr/local/bin/qwen", "--acp"}) {
		t.Errorf("found %+v, %v", found, err)
	}
}

func TestAProgramNoAgentRunsIsNotInstalled(t *testing.T) {
	registry := NewRegistry(nil)

	for _, program := range []string{"acp:", "claude"} {
		if _, err := registry.Find(program, []string{"/bin/true"}); !errors.Is(err, ErrNotInstalled) {
			t.Errorf("%s: err = %v, want %v", program, err, ErrNotInstalled)
		}
	}
	if _, err := registry.Find("acp:absent", nil); !errors.Is(err, ErrNotInstalled) {
		t.Errorf("err = %v, want %v", err, ErrNotInstalled)
	}
}
