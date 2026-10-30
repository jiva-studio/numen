package service

import (
	"path/filepath"
	"slices"
	"strings"
)

// acpProgramPrefix marks the programs of the settings that serve ACP. The rest of the name is the
// program a completion model names.
const acpProgramPrefix = "acp:"

// acpSubcommand is the argument that starts the ACP server of an agent.
const acpSubcommand = "acp"

// defaultACPProgram is the program name of an agent whose executable gives no name.
const defaultACPProgram = "acp"

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
	{program: "claude-code", title: "Claude Code (ACP)", binary: "claude-code-acp", args: nil},
	{program: "codex", title: "Codex (ACP)", binary: "codex-acp", args: nil},
	{program: "cursor-agent", title: "Cursor Agent", binary: "cursor-agent", args: []string{acpSubcommand}},
	{program: "gemini", title: "Gemini CLI", binary: "gemini", args: []string{"--acp"}},
	{program: "goose", title: "Goose", binary: "goose", args: []string{acpSubcommand}},
	{program: "kimi", title: "Kimi CLI", binary: "kimi", args: []string{acpSubcommand}},
	{program: "opencode", title: "OpenCode", binary: "opencode", args: []string{acpSubcommand}},
	{program: "qwen", title: "Qwen Code", binary: "qwen", args: []string{"--acp"}},
}

func getExecutableProgramName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return defaultACPProgram
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
