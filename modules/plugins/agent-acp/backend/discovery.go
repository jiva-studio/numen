package main

import (
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
