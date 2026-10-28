package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/clock"
)

// acpListLimit is how long an agent has to open the session its models are read from.
const acpListLimit = 3 * time.Second

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
			program := "acp"
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
	program := "acp"
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
	inFlight map[string]bool
	isAsync  bool
	// now is the clock the backoff is kept by. Nil is the system clock.
	now func() time.Time
}

// getNow returns the time by the clock of the cache.
func (c *acpModelCache) getNow() time.Time {
	if c.now == nil {
		return clock.Now()
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
	if ok || isBackingOff {
		c.mu.Unlock()
		return listing
	}
	if c.isAsync {
		if !c.inFlight[key] {
			if c.inFlight == nil {
				c.inFlight = make(map[string]bool)
			}
			c.inFlight[key] = true
			go c.fetchInBackground(agent, key)
		}
		if cached, found := readACPCache(key); found {
			if c.entries == nil {
				c.entries = make(map[string]acpListing)
			}
			c.entries[key] = cached
			c.mu.Unlock()
			return cached
		}
		c.mu.Unlock()
		return acpListing{}
	}
	c.mu.Unlock()
	return c.fetchAndStore(ctx, agent, key)
}

func (c *acpModelCache) fetchInBackground(agent acpAgent, key string) {
	defer func() {
		c.mu.Lock()
		delete(c.inFlight, key)
		c.mu.Unlock()
	}()
	c.fetchAndStore(context.Background(), agent, key)
}

func (c *acpModelCache) fetchAndStore(ctx context.Context, agent acpAgent, key string) acpListing {
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
	if c.isAsync {
		writeACPCache(key, listing)
	}
	return listing
}

type persistedACPListing struct {
	Models  []acpModel        `json:"models"`
	Current string            `json:"current"`
	Levels  []reasoning.Level `json:"levels,omitempty"`
	Level   string            `json:"level,omitempty"`
}

func acpCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", err
		}
		dir = filepath.Join(home, ".cache")
	}
	return filepath.Join(dir, "numen", "models-acp.json"), nil
}

func readACPCache(key string) (acpListing, bool) {
	path, err := acpCachePath()
	if err != nil {
		return acpListing{}, false
	}
	data, err := agentrun.ReadFile(path)
	if err != nil {
		return acpListing{}, false
	}
	var all map[string]persistedACPListing
	if json.Unmarshal(data, &all) != nil {
		return acpListing{}, false
	}
	one, found := all[key]
	if !found || len(one.Models) == 0 {
		return acpListing{}, false
	}
	return acpListing{
		models:  one.Models,
		current: one.Current,
		levels:  one.Levels,
		level:   one.Level,
	}, true
}

func writeACPCache(key string, listing acpListing) {
	if len(listing.models) == 0 {
		return
	}
	path, err := acpCachePath()
	if err != nil {
		return
	}
	all := make(map[string]persistedACPListing)
	if data, err := agentrun.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &all)
	}
	all[key] = persistedACPListing{
		Models:  listing.models,
		Current: listing.current,
		Levels:  listing.levels,
		Level:   listing.level,
	}
	data, err := json.Marshal(all)
	if err != nil {
		return
	}
	_ = agentrun.CreateDir(filepath.Dir(path), 0o700)
	_ = agentrun.WriteFile(path, data, 0o600)
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
