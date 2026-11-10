package arguments

import (
	"slices"
	"testing"
)

func TestBuildClaudeArguments_SettingSources(t *testing.T) {
	args := Build(Options{Configuration: "cfg.json", Tools: []string{"note_read"}})
	assertPair(t, args, "--setting-sources", "")
	args = Build(Options{Configuration: "cfg.json", Tools: []string{"note_read"}, MayReadHooks: true})
	assertPair(t, args, "--setting-sources", "user")
}

func TestTheAllowanceNamesTheSearchAndTheTasksTools(t *testing.T) {
	args := Build(Options{Configuration: "cfg.json", Tools: []string{"note_read", "note_edit"}})
	assertPair(t, args, "--allowedTools", "WebSearch,mcp__numen__note_edit,mcp__numen__note_read")
	assertPair(t, args, "--tools", "WebSearch")
}

func TestTheDefaultNumberOfTurnsIsOneHundred(t *testing.T) {
	assertPair(t, Build(Options{Tools: []string{"note_read"}}), "--max-turns", "100")
	assertPair(t, Build(Options{Turns: 1, Tools: []string{"note_read"}}), "--max-turns", "1")
	assertPair(t, Build(Options{Turns: 7, Tools: []string{"note_read"}}), "--max-turns", "7")

	withFlags := Build(Options{
		Tools:        []string{"note_read"},
		Instructions: "inst",
		Session:      "sess-1",
	})
	assertPair(t, withFlags, "--append-system-prompt", "inst")
	assertPair(t, withFlags, "--resume", "sess-1")

	withoutFlags := Build(Options{Tools: []string{"note_read"}})
	if slices.Contains(withoutFlags, "--append-system-prompt") || slices.Contains(withoutFlags, "--resume") {
		t.Errorf("unexpected flags in %v", withoutFlags)
	}
}

func assertPair(t *testing.T, args []string, flag, want string) {
	t.Helper()
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			if args[i+1] != want {
				t.Errorf("%s = %q, want %q", flag, args[i+1], want)
			}
			return
		}
	}
	t.Errorf("%s absent from %v", flag, args)
}

func TestTheLevelOfReasoningIsHandedToTheProgramAsItsEffort(t *testing.T) {
	for _, tools := range [][]string{{"note_read"}, nil} {
		args := Build(Options{Reasoning: "high", Tools: tools})
		assertPair(t, args, "--effort", "high")
		if slices.Contains(Build(Options{Tools: tools}), "--effort") {
			t.Error("a run that names no level is given an effort")
		}
	}
}
