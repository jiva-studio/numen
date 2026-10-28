package service

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
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
		program := "acp"
		title := "ACP Agent"
		if len(execPaths) > 1 {
			program = getExecutableProgramName(execPath)
			title = program
		}
		agents = append(agents, acpAgent{
			Program: program,
			Title:   title,
			Command: []string{execPath},
		})
	}
	slices.SortFunc(agents, func(a, b acpAgent) int { return strings.Compare(a.Program, b.Program) })
	return agents
}

// acpListing is what an agent reports for a session: its models and the one it answers with.
type acpListing struct {
	models  []acpModel
	current string
	// levels are the depths of reasoning the agent reports for the current model, and level the one it
	// works at.
	levels []reasoning.Level
	level  string
}

// acpModelCache keeps what each agent listed, for as long as the command that starts it stays the
// same. A failure keeps nothing, and the agent that failed is left alone for a while.
type acpModelCache struct {
	mu       sync.Mutex
	entries  map[string]acpListing
	failedAt map[string]time.Time
	// now is the clock the backoff is kept by. Nil is the system clock.
	now func() time.Time
}

// getNow returns the time by the clock of the cache.
func (c *acpModelCache) getNow() time.Time {
	if c.now == nil {
		return time.Now()
	}
	return c.now()
}

// get returns the agent's listing: the one kept, or the one it reports now. The agent is asked
// without the lock held.
func (c *acpModelCache) get(ctx context.Context, agent acpAgent) acpListing {
	key := agent.Program + "\x00" + strings.Join(agent.Command, "\x00")
	c.mu.Lock()
	listing, ok := c.entries[key]
	failedAt, hasFailed := c.failedAt[key]
	isBackingOff := hasFailed && c.getNow().Sub(failedAt) < acpFailureBackoff
	c.mu.Unlock()
	if ok || isBackingOff {
		return listing
	}
	listing, err := readACPListing(ctx, agent)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if ctx.Err() == nil {
			if c.failedAt == nil {
				c.failedAt = make(map[string]time.Time)
			}
			c.failedAt[key] = c.getNow()
		}
		return acpListing{}
	}
	if c.entries == nil {
		c.entries = make(map[string]acpListing)
	}
	c.entries[key] = listing
	delete(c.failedAt, key)
	return listing
}

// readACPListing opens a session with the agent and returns the models it reports for it.
func readACPListing(ctx context.Context, agent acpAgent) (acpListing, error) {
	asking, stop := context.WithTimeout(ctx, acpListLimit)
	defer stop()
	session, err := openACPSession(asking, agent, toolEndpoint{})
	if err != nil {
		return acpListing{}, err
	}
	defer session.close()
	return acpListing{
		models: session.models, current: session.current, levels: session.thoughtLevels, level: session.thoughtCurrent,
	}, nil
}
