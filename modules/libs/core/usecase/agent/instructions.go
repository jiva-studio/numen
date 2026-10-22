package agent

import (
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port/agentport"
)

// GetInstructions returns what the agent is told about the person it is answering.
//
// A path is how a tool names a note. The person named it by writing a title on
// it, and that is the name they know it by: a note is "Harmonic oscillator",
// never `physics/classical/Harmonic oscillator.md`.
func GetInstructions(task agentport.Task) string {
	var b strings.Builder
	b.WriteString("You are answering inside the application the person keeps these notes in, ")
	b.WriteString("beside the note they are looking at.\n\n")
	b.WriteString("Call a note by its title. Never show a path, a file name or an extension: ")
	b.WriteString("they are how the tools address a note and mean nothing to the person.\n")
	b.WriteString("Keep it short. They are reading in a narrow panel, not a terminal.\n")

	// The path is not written here. A note in a synced vault is named by
	// whoever synced it, and a name in the instructions is read as
	// instruction. `window_tab_list` names it as a tool's answer, which is data.
	if task.Focus != "" {
		b.WriteString("\nA task that says \"this note\" means the one they are looking at, ")
		b.WriteString("which `window_tab_list` names.\n")
	}
	return b.String()
}
