package claudecode

import (
	"slices"
	"strings"
)

// dropped names the environment variables that describe a Claude Code session
// somebody else is running.
//
// A window is not one, so these are stripped from the environment the agent is
// started with. What says how to reach a model is not here: that belongs to the
// installation and is passed on.
var dropped = []string{
	"CLAUDECODE",
	"CLAUDE_CODE_SESSION_ID",
	"CLAUDE_CODE_CHILD_SESSION",
	"CLAUDE_CODE_ENTRYPOINT",
	"CLAUDE_CODE_EXECPATH",
	"CLAUDE_CODE_MESSAGING_SOCKET",
	"CLAUDE_CODE_MESSAGING_TOKEN",
	"CLAUDE_PID",
	"CLAUDE_EFFORT",
}

// environment is what the agent is started with: everything the machine holds,
// less what belongs to a session it is not part of.
func environment(held []string) []string {
	out := make([]string, 0, len(held))
	for _, entry := range held {
		name, _, found := strings.Cut(entry, "=")
		if found && slices.Contains(dropped, name) {
			continue
		}
		out = append(out, entry)
	}
	return out
}
