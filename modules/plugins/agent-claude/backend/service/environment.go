package service

import (
	"slices"
	"strings"
)

var droppedEnvVars = []string{
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

func buildEnvironment(held []string) []string {
	out := make([]string, 0, len(held))
	for _, entry := range held {
		name, _, found := strings.Cut(entry, "=")
		if found && slices.Contains(droppedEnvVars, name) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

// getVariables returns vars as the entries of an environment. A host that links the plugin in
// gives it the variables a launched process of it would be started with.
func getVariables(vars map[string]string) []string {
	out := make([]string, 0, len(vars))
	for name, value := range vars {
		out = append(out, name+"="+value)
	}
	return out
}
