package programs

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

func TestFindCommandFindsTheCommandOfAProgram(t *testing.T) {
	cmds := []agentrun.ProgramCommand{
		{Program: "other", Command: []string{"other-bin"}},
		{Program: "codex", Command: []string{"codex-bin", "--arg"}},
	}
	if got := findCommand(cmds, "codex"); !slices.Equal(got, []string{"codex-bin", "--arg"}) {
		t.Errorf("findCommand(codex) = %v", got)
	}
	if got := findCommand(cmds, "missing"); got != nil {
		t.Errorf("findCommand(missing) = %v, want nil", got)
	}
}
