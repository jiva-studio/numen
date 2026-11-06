package agent

import (
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// GetInstructions returns the instructions for the agent about the user it is
// answering. They are the same for every task: the focus is not written into
// them, because a note name in a synced vault is controlled by the sync source.
//
// Tools address a note by path, but the user refers to a note by its title: a note
// is "Harmonic oscillator", never `physics/classical/Harmonic oscillator.md`.
func GetInstructions(agentport.Task) string {
	var b strings.Builder
	b.WriteString("You are answering inside the application the user keeps these notes in, ")
	b.WriteString("beside the note the user has open.\n\n")
	b.WriteString("Call a note by its title. Never show a path, a file name or an extension: ")
	b.WriteString("the tools address a note by them, and the user does not.\n")
	b.WriteString("Keep it short. The answer is shown in a narrow panel, not a terminal.\n")
	return b.String()
}
