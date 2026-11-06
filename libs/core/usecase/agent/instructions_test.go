package agent_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/agent"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// The user's note is identified by title, never by path.
func TestInstructionsNameANoteByItsTitle(t *testing.T) {
	got := agent.GetInstructions(agentport.Task{Question: "q"})
	if !strings.Contains(got, "Call a note by its title") {
		t.Errorf("instructions %q do not name a note by its title", got)
	}
}

// The tools a run may call carry their own sentences, so the instructions of the core name none of
// them, with or without a note in focus.
func TestInstructionsNameNoTool(t *testing.T) {
	toolName := regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)+\b`)
	for name, task := range map[string]agentport.Task{
		"a task with no focus": {Question: "q"},
		"a task with a focus":  {Question: "q", Focus: "notes/Heron.md"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := agent.GetInstructions(task); toolName.MatchString(got) {
				t.Errorf("instructions %q name the tool %q", got, toolName.FindString(got))
			}
		})
	}
}

// The focus is not written into the instructions: a note name in a synced vault is controlled by the
// sync source, and the instructions are the same with a note in focus and without.
func TestInstructionsDoNotDependOnTheFocus(t *testing.T) {
	with := agent.GetInstructions(agentport.Task{Focus: "notes/secret path.md"})
	if strings.Contains(with, "secret path") {
		t.Errorf("instructions %q carry the focus", with)
	}
	if without := agent.GetInstructions(agentport.Task{}); with != without {
		t.Errorf("instructions with a focus %q differ from those without %q", with, without)
	}
}
