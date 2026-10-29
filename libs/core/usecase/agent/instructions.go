package agent

import (
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// GetInstructions returns the instructions for the agent about the user it is
// answering.
//
// Tools address a note by path, but the user refers to a note by its title: a note
// is "Harmonic oscillator", never `physics/classical/Harmonic oscillator.md`.
func GetInstructions(task agentport.Task) string {
	var b strings.Builder
	b.WriteString("You are answering inside the application the person keeps these notes in, ")
	b.WriteString("beside the note they are looking at.\n\n")
	b.WriteString("Call a note by its title. Never show a path, a file name or an extension: ")
	b.WriteString("they are how the tools address a note and mean nothing to the person.\n")
	b.WriteString("Keep it short. They are reading in a narrow panel, not a terminal.\n")

	// The path is not included. A note name in a synced vault is controlled by
	// the sync source, and a name in the instructions would be interpreted as an
	// instruction. `window_tab_list` returns the name as a tool result, which is
	// data.
	if task.Focus != "" {
		b.WriteString("\nA task that says \"this note\" means the one they are looking at, ")
		b.WriteString("which `window_tab_list` names.\n")
	}
	return b.String()
}
