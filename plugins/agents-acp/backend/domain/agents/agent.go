// Package agents knows the ACP agents the plugin can run and where the command of each comes from.
package agents

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

// ProgramPrefix marks the programs of the settings that serve ACP. The rest of the name is the
// program a completion model names.
const ProgramPrefix = "acp:"

// acpSubcommand is the argument that starts the ACP server of an agent.
const acpSubcommand = "acp"

// defaultProgram is the program name of an agent whose executable gives no name.
const defaultProgram = "acp"

// ErrNotInstalled is the reason a task names an agent that no command starts.
var ErrNotInstalled = errors.New("the ACP agent is not installed")

// Agent is an agent that serves the Agent Client Protocol over its standard streams.
type Agent struct {
	Program string
	Title   string
	Command []string
}

type known struct {
	program string
	title   string
	binary  string
	args    []string
}

func getKnown() []known {
	return []known{
		{program: "claude-code", title: "Claude Code (ACP)", binary: "claude-code-acp", args: nil},
		{program: "codex", title: "Codex (ACP)", binary: "codex-acp", args: nil},
		{program: "cursor-agent", title: "Cursor Agent", binary: "cursor-agent", args: []string{acpSubcommand}},
		{program: "gemini", title: "Gemini CLI", binary: "gemini", args: []string{"--acp"}},
		{program: "goose", title: "Goose", binary: "goose", args: []string{acpSubcommand}},
		{program: "kimi", title: "Kimi CLI", binary: "kimi", args: []string{acpSubcommand}},
		{program: "opencode", title: "OpenCode", binary: "opencode", args: []string{acpSubcommand}},
		{program: "qwen", title: "Qwen Code", binary: "qwen", args: []string{"--acp"}},
	}
}

// FindInstalled lists the known agents whose binary lookPath finds. A nil lookPath finds none.
func FindInstalled(lookPath func(string) (string, error)) []Agent {
	if lookPath == nil {
		return nil
	}
	var agents []Agent
	for _, candidate := range getKnown() {
		binaryPath, err := lookPath(candidate.binary)
		if err != nil || binaryPath == "" {
			continue
		}
		command := append([]string{binaryPath}, candidate.args...)
		agents = append(agents, Agent{Program: candidate.program, Title: candidate.title, Command: command})
	}
	return agents
}

// GetFromCommands lists the agents the settings name: the programs that carry ProgramPrefix and
// hold a command line, sorted by name.
func GetFromCommands(commands map[string][]string) []Agent {
	var agents []Agent
	for name, command := range commands {
		program, isACP := strings.CutPrefix(name, ProgramPrefix)
		if !isACP || program == "" || len(command) == 0 {
			continue
		}
		agents = append(agents, Agent{Program: program, Title: program, Command: command})
	}
	sortByProgram(agents)
	return agents
}

// GetProgramName returns the name of the program an executable file holds.
func GetProgramName(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return defaultProgram
	}
	return name
}

// ParsePaths reads the executable paths of the setting, one on a line.
func ParsePaths(value string) []string {
	var paths []string
	for one := range strings.SplitSeq(value, "\n") {
		entry := strings.TrimSpace(one)
		if entry != "" {
			paths = append(paths, entry)
		}
	}
	return paths
}

func createCustom(execPath string, isMulti bool) Agent {
	baseName := GetProgramName(execPath)
	for _, candidate := range getKnown() {
		if candidate.binary == baseName {
			program := defaultProgram
			if isMulti {
				program = candidate.program
			}
			return Agent{Program: program, Title: candidate.title, Command: append([]string{execPath}, candidate.args...)}
		}
	}
	if isMulti {
		return Agent{Program: baseName, Title: baseName, Command: []string{execPath}}
	}
	return Agent{Program: defaultProgram, Title: "ACP Agent", Command: []string{execPath}}
}

func sortByProgram(agents []Agent) {
	slices.SortFunc(agents, func(a, b Agent) int { return strings.Compare(a.Program, b.Program) })
}
