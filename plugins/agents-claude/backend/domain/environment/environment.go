// Package environment prepares the variables a Claude Code program is started with.
package environment

import (
	"slices"
	"strings"
)

func getDroppedEnvVars() []string {
	return []string{
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
}

// DropSession returns the entries of target without the variables that tie a program to the
// Claude Code session that starts it.
func DropSession(target []string) []string {
	droppedEnvVars := getDroppedEnvVars()
	out := make([]string, 0, len(target))
	for _, entry := range target {
		name, _, found := strings.Cut(entry, "=")
		if found && slices.Contains(droppedEnvVars, name) {
			continue
		}
		out = append(out, entry)
	}
	return out
}
