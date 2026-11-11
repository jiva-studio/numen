package programs

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-claude/backend/domain/settings"
)

// fixedFinder is the outer edge: the machine, where the program is installed at one path or none.
type fixedFinder struct{ found string }

func (f fixedFinder) Find(string) string { return f.found }

type namedModels struct{}

func (namedModels) List(context.Context, string, []string, []string) []*reasoning.Model {
	return []*reasoning.Model{reasoning.NewModel("sonnet", "Sonnet", nil, "")}
}

type noVariables struct{}

func (noVariables) GetVariables() []string { return nil }

func newPrograms(t *testing.T, found string) *Programs {
	t.Helper()
	p, err := New(Dependencies{
		Settings: new(settings.Settings), Finder: fixedFinder{found: found}, Models: namedModels{}, Environment: noVariables{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNewRefusesAMissingDependency(t *testing.T) {
	if _, err := New(Dependencies{}); err == nil {
		t.Error("programs without dependencies were created")
	}
}

func TestTheStateIsUnreachableWhereTheProgramIsNotFound(t *testing.T) {
	if newPrograms(t, "").GetState().Unreachable == "" {
		t.Error("a program that is not installed was said to be reachable")
	}
	if got := newPrograms(t, "/usr/bin/claude").GetState().Unreachable; got != "" {
		t.Errorf("an installed program is unreachable: %q", got)
	}
}

func TestResolveCommand_PrefersTheTaskCommand(t *testing.T) {
	p := newPrograms(t, "/usr/bin/claude")
	if command, extra := p.ResolveCommand([]string{"/opt/claude", "--first"}); command != "/opt/claude" || len(extra) != 1 || extra[0] != "--first" {
		t.Errorf("resolved %q %v", command, extra)
	}
	if command, extra := p.ResolveCommand(nil); command != "/usr/bin/claude" || extra != nil {
		t.Errorf("resolved %q %v", command, extra)
	}
}

func TestTheProgramListNamesClaudeAsInstalledWhereACommandIsConfigured(t *testing.T) {
	p := newPrograms(t, "")
	list := p.List(t.Context(), []agentrun.ProgramCommand{
		{Program: "codex", Command: []string{"/bin/codex"}},
		{Program: Name, Command: []string{"/opt/claude/bin/claude"}},
	})
	if len(list) != 1 || !list[0].IsInstalled || !list[0].IsDefault || !list[0].LimitsTools ||
		list[0].Name != Name || list[0].Title != "Claude Code" || len(list[0].Models) != 1 {
		t.Errorf("the programs are %v", list)
	}
	if other := p.List(t.Context(), []agentrun.ProgramCommand{{Program: "codex", Command: []string{"/bin/codex"}}}); other[0].IsInstalled {
		t.Errorf("a command for another program installed Claude Code: %v", other)
	}
}
