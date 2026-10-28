package service

import (
	"os"
	"os/exec"
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
	{program: "cursor-agent", title: "Cursor Agent", binary: "cursor-agent", args: []string{"acp"}},
	{program: "gemini", title: "Gemini CLI", binary: "gemini", args: []string{"--acp"}},
	{program: "goose", title: "Goose", binary: "goose", args: []string{"acp"}},
	{program: "kimi", title: "Kimi CLI", binary: "kimi", args: []string{"acp"}},
	{program: "opencode", title: "OpenCode", binary: "opencode", args: []string{"acp"}},
	{program: "qwen", title: "Qwen Code", binary: "qwen", args: []string{"--acp"}},
}

// discoverInstalledACPAgents finds known ACP-compatible agents installed on the machine.
func discoverInstalledACPAgents() []acpAgent {
	return discoverACPAgentsWithLookup(findACPExecutable)
}

func discoverACPAgentsWithLookup(lookup func(binary string) (string, bool)) []acpAgent {
	var agents []acpAgent
	for _, candidate := range knownACPAgents {
		binaryPath, ok := lookup(candidate.binary)
		if !ok || binaryPath == "" {
			continue
		}
		command := make([]string, 0, 1+len(candidate.args))
		command = append(command, binaryPath)
		command = append(command, candidate.args...)
		agents = append(agents, acpAgent{
			Program: candidate.program,
			Title:   candidate.title,
			Command: command,
		})
	}
	return agents
}

func findACPExecutable(binary string) (string, bool) {
	if binaryPath, err := exec.LookPath(binary); err == nil && binaryPath != "" {
		return binaryPath, true
	}
	home, _ := os.UserHomeDir()
	var dirs []string
	if home != "" {
		dirs = append(dirs,
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".npm-global", "bin"),
			filepath.Join(home, ".bun", "bin"),
			filepath.Join(home, ".cargo", "bin"),
		)
	}
	dirs = append(dirs, "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin")
	for _, dir := range dirs {
		candidate := filepath.Join(dir, binary)
		if binaryPath, err := exec.LookPath(candidate); err == nil && binaryPath != "" {
			return binaryPath, true
		}
	}
	return "", false
}

func getExecutableProgramName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return "acp"
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
