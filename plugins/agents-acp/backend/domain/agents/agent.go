// Package agents knows the connection type of the plugin: the command that starts the ACP server of
// the program a connection names.
package agents

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
)

// TypeACP is the connection type of the plugin.
const TypeACP = "acp"

// acpSubcommand is the argument that starts the ACP server of several agents.
const acpSubcommand = "acp"

var (
	// ErrNoProgram is the reason a connection names no program.
	ErrNoProgram = errors.New("the connection names no program")
	// ErrNotFound is the reason the program of a connection is not on the machine.
	ErrNotFound = errors.New("the program of the connection is not on the machine")
)

// Agent is a program that serves the Agent Client Protocol over its standard streams.
type Agent struct {
	Type    string
	Command []string
}

func getArguments(executable string) []string {
	base := filepath.Base(executable)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	for _, one := range ListKnown() {
		if one.Program == name {
			return slices.Clone(one.Arguments)
		}
	}
	return nil
}

// Find returns the agent of a connection of the type with the program field, which the type requires. A
// nil lookPath finds nothing.
func Find(connectionType, program string, lookPath func(string) (string, error)) (Agent, error) {
	program = strings.TrimSpace(program)
	if program == "" {
		return Agent{Type: connectionType, Command: nil}, ErrNoProgram
	}
	if lookPath == nil {
		return Agent{Type: connectionType, Command: nil}, ErrNotFound
	}
	found, err := lookPath(program)
	if err != nil || found == "" {
		return Agent{Type: connectionType, Command: nil}, ErrNotFound
	}
	return Agent{Type: connectionType, Command: append([]string{found}, getArguments(program)...)}, nil
}

// GetKey returns what tells the programs of two agents apart.
func (a Agent) GetKey() string {
	return a.Type + "\x00" + strings.Join(a.Command, "\x00")
}
