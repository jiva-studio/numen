package service

import (
	"path/filepath"
	"slices"
	"strings"
)

// acpProgramPrefix marks the programs of the settings that serve ACP. The rest of the name is the
// program a completion model names.
const acpProgramPrefix = "acp:"

// acpAgent is an agent that serves the Agent Client Protocol over its standard streams.
type acpAgent struct {
	// Program is what a completion model names the agent by.
	Program string
	Title   string
	// Command starts the agent's ACP server.
	Command []string
}

type knownACPAgent struct {
	program string
	title   string
	binary  string
	args    []string
}

var knownACPAgents = []knownACPAgent{
	{program: "claude-code", title: "Claude Code (ACP)", binary: "claude-code-acp"},
	{program: "codex", title: "Codex (ACP)", binary: "codex-acp"},
	{program: "cursor-agent", title: "Cursor Agent", binary: "cursor-agent", args: []string{programACP}},
	{program: "gemini", title: "Gemini CLI", binary: "gemini", args: []string{"--acp"}},
	{program: "goose", title: "Goose", binary: "goose", args: []string{programACP}},
	{program: "kimi", title: "Kimi CLI", binary: "kimi", args: []string{programACP}},
	{program: "opencode", title: "OpenCode", binary: "opencode", args: []string{programACP}},
	{program: "qwen", title: "Qwen Code", binary: "qwen", args: []string{"--acp"}},
}

func getExecutableProgramName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return programACP
	}
	return name
}

// getACPAgentsFromCommands returns the agents the command lines of the settings name, in the order
// of their programs: each program with the ACP prefix and a command line.
func getACPAgentsFromCommands(commands map[string][]string) []acpAgent {
	var agents []acpAgent
	for name, command := range commands {
		program, isACP := strings.CutPrefix(name, acpProgramPrefix)
		if !isACP || program == "" || len(command) == 0 {
			continue
		}
		agents = append(agents, acpAgent{Program: program, Title: program, Command: command})
	}
	slices.SortFunc(agents, func(a, b acpAgent) int { return strings.Compare(a.Program, b.Program) })
	return agents
}
