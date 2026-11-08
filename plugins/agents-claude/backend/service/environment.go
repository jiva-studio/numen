package service

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

func buildEnvironment(target []string) []string {
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

func getVariables(vars map[string]string) []string {
	out := make([]string, 0, len(vars))
	for name, value := range vars {
		out = append(out, name+"="+value)
	}
	return out
}
