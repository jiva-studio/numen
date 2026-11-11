package agents

import (
	"slices"
	"strings"
	"sync"
)

const executablePathSetting = "executablePath"

// Registry holds the agents the plugin can run: the commands of the settings, the known agents
// found on the PATH and the executables of the settings.
type Registry struct {
	lookPath func(string) (string, error)

	mu       sync.RWMutex
	found    []Agent
	paths    []string
	commands map[string][]string
}

// NewRegistry returns a Registry that finds the known agents with lookPath.
func NewRegistry(lookPath func(string) (string, error)) *Registry {
	return &Registry{
		lookPath: lookPath,
		mu:       sync.RWMutex{},
		found:    FindInstalled(lookPath),
		paths:    nil,
		commands: nil,
	}
}

// SetConfig reads the plugin settings and finds the known agents again.
func (r *Registry) SetConfig(values map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if value, ok := values[executablePathSetting]; ok {
		r.paths = ParsePaths(value)
	}
	r.found = FindInstalled(r.lookPath)
}

// SetCommands replaces the command lines of the programs the settings name.
func (r *Registry) SetCommands(commands map[string][]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = commands
}

// List returns every agent, sorted by program name. A command of the settings wins over a known
// agent of the same program or binary, and over an executable of the same path.
func (r *Registry) List() []Agent {
	r.mu.RLock()
	commands := r.commands
	paths := slices.Clone(r.paths)
	found := slices.Clone(r.found)
	r.mu.RUnlock()
	return merge(GetFromCommands(commands), found, paths)
}

// Find returns the agent a program of the form ProgramPrefix+name runs. A non-empty command starts
// the agent where the settings name none.
func (r *Registry) Find(program string, command []string) (Agent, error) {
	name, isACP := strings.CutPrefix(program, ProgramPrefix)
	if !isACP || name == "" {
		return Agent{}, ErrNotInstalled
	}
	if len(command) > 0 {
		return Agent{Program: name, Title: name, Command: command}, nil
	}
	for _, agent := range r.List() {
		if agent.Program == name {
			return agent, nil
		}
	}
	return Agent{}, ErrNotInstalled
}

func merge(named, found []Agent, paths []string) []Agent {
	agents := named
	for _, candidate := range found {
		hasAgent := slices.ContainsFunc(agents, func(a Agent) bool {
			return a.Program == candidate.Program || (len(a.Command) > 0 && a.Command[0] == candidate.Command[0])
		})
		if !hasAgent {
			agents = append(agents, candidate)
		}
	}
	for _, execPath := range paths {
		hasCustom := slices.ContainsFunc(agents, func(a Agent) bool {
			return len(a.Command) > 0 && a.Command[0] == execPath
		})
		if !hasCustom {
			agents = append(agents, createCustom(execPath, len(paths) > 1))
		}
	}
	sortByProgram(agents)
	return agents
}
