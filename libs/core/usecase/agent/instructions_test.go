package agent_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/agent"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// The person's note is named by title, never by path.
func TestInstructionsNameANoteByItsTitle(t *testing.T) {
	got := agent.GetInstructions(agentport.Task{Question: "q"})
	if !strings.Contains(got, "Call a note by its title") {
		t.Errorf("instructions %q do not name a note by its title", got)
	}
}

// The focus is where "this note" is read from, and the note itself is not written into the
// instructions.
func TestInstructionsPointAtTheTabListWhenTheWindowShowsANote(t *testing.T) {
	with := agent.GetInstructions(agentport.Task{Focus: "notes/secret path.md"})
	if !strings.Contains(with, "window_tab_list") {
		t.Errorf("instructions %q do not point at the tab list", with)
	}
	if strings.Contains(with, "secret path") {
		t.Errorf("instructions %q carry the focus", with)
	}
	if instructions := agent.GetInstructions(agentport.Task{}); strings.Contains(instructions, "window_tab_list") {
		t.Errorf("instructions %q point at a tab list with no focus", instructions)
	}
}
