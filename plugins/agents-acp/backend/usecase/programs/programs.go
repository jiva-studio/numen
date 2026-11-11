// Package programs lists the ACP agents as programs with the models each reports.
package programs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/reasoning"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/acp"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/domain/agents"
)

// listLimit is how long an agent has to open the session its models are read from.
const listLimit = 15 * time.Second

// failureBackoff is how long an agent that failed to list is left alone.
const failureBackoff = 30 * time.Second

// Registry holds the agents to list and the command lines the settings name for them.
type Registry interface {
	SetCommands(commands map[string][]string)
	List() []agents.Agent
}

// Lister lists the programs of the agents, reading the models of an agent from a session and
// keeping them.
type Lister struct {
	registry Registry
	launcher acp.Launcher
	models   *cache.Cache[acp.Listing]
}

// New returns a Lister that starts agents with launcher. When isAsync is set, an agent whose models
// are not kept yet is listed with none and read in the background.
func New(registry Registry, launcher acp.Launcher, source cache.Clock, isAsync bool) (*Lister, error) {
	if registry == nil || launcher == nil || source == nil {
		return nil, errors.New("programs: a registry, a launcher and a clock are required")
	}
	models := cache.New[acp.Listing](
		"models",
		cache.WithTimeout(listLimit),
		cache.WithFailureBackoff(failureBackoff),
		cache.WithAsync(isAsync),
		cache.WithClock(source),
	)
	return &Lister{registry: registry, launcher: launcher, models: models}, nil
}

// Execute keeps the command lines of the requested programs and lists, for each agent, its program
// name and the models it reports.
func (l *Lister) Execute(ctx context.Context, requested []agentrun.ProgramCommand) ([]*agentrun.Program, error) {
	commands := make(map[string][]string, len(requested))
	for _, one := range requested {
		commands[one.Program] = one.Command
	}
	l.registry.SetCommands(commands)

	found := l.registry.List()
	listings := make([]acp.Listing, len(found))
	var group sync.WaitGroup
	for idx, agent := range found {
		group.Add(1)
		go func(i int, target agents.Agent) {
			defer group.Done()
			listings[i] = l.getListing(ctx, target)
		}(idx, agent)
	}
	group.Wait()

	var programs []*agentrun.Program
	for idx, agent := range found {
		programs = append(programs, createProgram(agent, listings[idx]))
	}
	return programs, nil
}

func (l *Lister) getListing(ctx context.Context, agent agents.Agent) acp.Listing {
	key := agent.Program + "\x00" + strings.Join(agent.Command, "\x00")
	// A failed load leaves the last listing of the cache, or none, and the agent is listed with it.
	listing, _, err := l.models.Get(ctx, key, func(ctx context.Context) (acp.Listing, error) {
		return l.ReadListing(ctx, agent)
	})
	if err != nil {
		return listing
	}
	return listing
}

// ReadListing opens a session with the agent and reads the models it reports. A session that fails to close
// fails the read.
func (l *Lister) ReadListing(ctx context.Context, agent agents.Agent) (listing acp.Listing, err error) {
	asking, stop := context.WithTimeout(ctx, listLimit)
	defer stop()
	session, err := acp.Open(asking, l.launcher, agent.Program, agent.Command, acp.Tools{URL: "", Token: "", Names: nil})
	if err != nil {
		return acp.Listing{}, err
	}
	defer func() { err = errors.Join(err, session.Close()) }()
	return session.GetListing(), nil
}

func createProgram(agent agents.Agent, listing acp.Listing) *agentrun.Program {
	program := &agentrun.Program{
		Name:        agents.ProgramPrefix + agent.Program,
		Title:       agent.Title,
		IsInstalled: true,
		LimitsTools: true,
		IsDefault:   false,
		Models:      nil,
	}
	for _, one := range listing.Models {
		title := one.Name
		if title == "" {
			title = one.ID
		}
		model := &reasoning.Model{Name: one.ID, Title: title, IsDefault: one.ID == listing.Current, ReasoningLevels: nil, DefaultReasoning: ""}
		if one.ID == listing.Current {
			model.ReasoningLevels = listing.Levels
			model.DefaultReasoning = listing.Level
		}
		program.Models = append(program.Models, model)
	}
	return program
}
