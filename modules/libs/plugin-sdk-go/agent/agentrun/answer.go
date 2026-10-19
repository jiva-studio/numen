package agentrun

import (
	"strings"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// HasTools reports whether the request names a tool of the vault. A request naming none is one
// answer, and the program is given no tool.
func HasTools(req *numenv1.AskAgentRequest) bool { return len(req.GetTools()) > 0 }

// WritePrompt is the prompt with the instructions before it, for a program that takes no system
// prompt.
func WritePrompt(instructions, prompt string) string {
	if strings.TrimSpace(instructions) == "" {
		return prompt
	}
	return instructions + "\n\n" + prompt
}
