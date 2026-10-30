package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
)

// acpListLimit is how long an agent has to open the session its models are read from.
const acpListLimit = 15 * time.Second

// acpFailureBackoff is how long an agent that failed to list is left alone.
const acpFailureBackoff = 30 * time.Second

// getACPAgents returns the agents the server was given, else those the settings name.
func (s *agentServer) getACPAgents() []acpAgent {
	if s.acpAgents != nil {
		return s.acpAgents()
	}
	s.mu.RLock()
	commands := s.commands
	execPaths := slices.Clone(s.executablePaths)
	discover := s.discover
	s.mu.RUnlock()

	agents := getACPAgentsFromCommands(commands)
	if discover != nil {
		for _, candidate := range discover() {
			hasAgent := slices.ContainsFunc(agents, func(a acpAgent) bool {
				return a.Program == candidate.Program || (len(a.Command) > 0 && len(candidate.Command) > 0 && a.Command[0] == candidate.Command[0])
			})
			if !hasAgent {
				agents = append(agents, candidate)
			}
		}
	}
	for _, execPath := range execPaths {
		hasCustom := slices.ContainsFunc(agents, func(a acpAgent) bool {
			return len(a.Command) > 0 && a.Command[0] == execPath
		})
		if hasCustom {
			continue
		}
		agent := buildCustomACPAgent(execPath, len(execPaths) > 1)
		agents = append(agents, agent)
	}
	slices.SortFunc(agents, func(a, b acpAgent) int { return strings.Compare(a.Program, b.Program) })
	return agents
}

func buildCustomACPAgent(execPath string, isMulti bool) acpAgent {
	baseName := getExecutableProgramName(execPath)
	for _, candidate := range knownACPAgents {
		if candidate.binary == baseName {
			command := make([]string, 0, 1+len(candidate.args))
			command = append(command, execPath)
			command = append(command, candidate.args...)
			program := programACP
			if isMulti {
				program = candidate.program
			}
			return acpAgent{
				Program: program,
				Title:   candidate.title,
				Command: command,
			}
		}
	}
	program := programACP
	title := "ACP Agent"
	if isMulti {
		program = baseName
		title = baseName
	}
	return acpAgent{
		Program: program,
		Title:   title,
		Command: []string{execPath},
	}
}

// acpListing is what an agent reports for a session: its models and the one it answers with.
type acpListing struct {
	Models  []acpModel `json:"models"`
	Current string     `json:"current"`
	// Levels are the depths of reasoning the agent reports for the current model, and Level the one it
	// works at.
	Levels []reasoning.Level `json:"levels,omitempty"`
	Level  string            `json:"level,omitempty"`
}

func newACPModelCache(isAsync bool) *cache.Cache[acpListing] {
	return cache.New[acpListing](
		"models",
		cache.WithPluginID("agents-acp"),
		cache.WithTimeout(acpListLimit),
		cache.WithFailureBackoff(acpFailureBackoff),
		cache.WithAsync(isAsync),
	)
}

func (s *agentServer) getModelCache() *cache.Cache[acpListing] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.acpModels == nil {
		s.acpModels = newACPModelCache(false)
	}
	return s.acpModels
}

func (s *agentServer) getACPListing(ctx context.Context, agent acpAgent) acpListing {
	key := agent.Program + "\x00" + strings.Join(agent.Command, "\x00")
	listing, _, err := s.getModelCache().Get(ctx, key, func(ctx context.Context) (acpListing, error) {
		return readACPListing(ctx, agent)
	})
	if err != nil {
		// A failed load leaves the last listing of the cache, or none, and the agent is listed with it.
		return listing
	}
	return listing
}

// readACPListing opens a session with the agent and returns the models it reports for it.
func readACPListing(ctx context.Context, agent acpAgent) (listing acpListing, err error) {
	asking, stop := context.WithTimeout(ctx, acpListLimit)
	defer stop()
	session, err := openACPSession(asking, agent, toolEndpoint{})
	if err != nil {
		return acpListing{}, err
	}
	defer func() { err = errors.Join(err, session.close()) }()
	return acpListing{
		Models: session.models, Current: session.current, Levels: session.thoughtLevels, Level: session.thoughtCurrent,
	}, nil
}
