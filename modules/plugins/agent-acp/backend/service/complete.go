package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/completion"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// acpListLimit is how long an agent has to open the session its models are read from.
const acpListLimit = 15 * time.Second

// acpFailureBackoff is how long an agent that failed to list is left alone.
const acpFailureBackoff = 30 * time.Second

var (
	errNoCompletionAgent = errors.New("no ACP agent is installed")
	errAgentNotInstalled = errors.New("the ACP agent is not installed")
)

// Complete answers one question from an installed ACP agent, in a session of its own with no MCP
// servers and no tools. The model names the agent and its model as "<program>/<model>", or the
// agent alone for its own model; empty is the first installed agent with its own model.
func (s *agentServer) Complete(
	ctx context.Context,
	req *connect.Request[numenv1.CompleteRequest],
) (*connect.Response[numenv1.CompleteResponse], error) {
	agents, model := s.getACPAgents(), req.Msg.GetModel()
	return completion.Answer(ctx, req.Msg.GetQuestion(), func(ctx context.Context, request completion.Request) (string, error) {
		agent, modelID, err := findCompletionAgent(agents, model)
		if err != nil {
			return "", err
		}
		return s.answer(ctx, agent, modelID, request.System, request.Prompt)
	})
}

// ListCompletionModels lists, for each installed ACP agent, the models it reports as
// "<program>/<model>", or the agent alone where it reports none or cannot be asked. The default is
// the model the first agent answers with.
func (s *agentServer) ListCompletionModels(
	ctx context.Context,
	_ *connect.Request[numenv1.ListCompletionModelsRequest],
) (*connect.Response[numenv1.ListCompletionModelsResponse], error) {
	res := &numenv1.ListCompletionModelsResponse{}
	for i, agent := range s.getACPAgents() {
		listing := s.acpModels.get(ctx, agent)
		if len(listing.models) == 0 {
			res.Models = append(res.Models, &numenv1.CompletionModel{
				Name: agent.Program, Title: agent.Title + " (default model)", IsDefault: i == 0,
			})
			continue
		}
		for _, one := range listing.models {
			title := one.Name
			if title == "" {
				title = one.ID
			}
			res.Models = append(res.Models, &numenv1.CompletionModel{
				Name:      agent.Program + "/" + one.ID,
				Title:     agent.Title + " · " + title,
				IsDefault: i == 0 && one.ID == listing.current,
			})
		}
	}
	return connect.NewResponse(res), nil
}

// getACPAgents returns the agents the server was given, else those the settings name.
func (s *agentServer) getACPAgents() []acpAgent {
	if s.acpAgents != nil {
		return s.acpAgents()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return getACPAgentsFromCommands(s.commands)
}

// findCompletionAgent returns the agent the model names and the model id within it.
func findCompletionAgent(agents []acpAgent, model string) (acpAgent, string, error) {
	if len(agents) == 0 {
		return acpAgent{}, "", errNoCompletionAgent
	}
	if model == "" {
		return agents[0], "", nil
	}
	program, modelID, _ := strings.Cut(model, "/")
	for _, agent := range agents {
		if agent.Program == program {
			return agent, modelID, nil
		}
	}
	return acpAgent{}, "", fmt.Errorf("%w: %s", errAgentNotInstalled, program)
}

// acpListing is what an agent reports for a session: its models and the one it answers with.
type acpListing struct {
	models  []acpModel
	current string
}

// acpModelCache caches what each agent listed, for as long as the command that starts it stays the
// same. A failure caches nothing, and the agent that failed is not asked again for a while.
type acpModelCache struct {
	mu       sync.Mutex
	entries  map[string]acpListing
	failedAt map[string]time.Time
	// now is the clock the backoff uses. Nil is the system clock.
	now func() time.Time
}

// getNow returns the time by the clock of the cache.
func (c *acpModelCache) getNow() time.Time {
	if c.now == nil {
		return time.Now()
	}
	return c.now()
}

// get returns the agent's listing: the one cached, or the one it reports now. The agent is asked
// without the lock held.
func (c *acpModelCache) get(ctx context.Context, agent acpAgent) acpListing {
	key := agent.Program + "\x00" + strings.Join(agent.Command, "\x00")
	c.mu.Lock()
	listing, isCached := c.entries[key]
	failedAt, hasFailed := c.failedAt[key]
	isBackingOff := hasFailed && c.getNow().Sub(failedAt) < acpFailureBackoff
	c.mu.Unlock()
	if isCached || isBackingOff {
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
	session, err := openACPSession(asking, agent)
	if err != nil {
		return acpListing{}, err
	}
	defer session.close()
	return acpListing{models: session.models, current: session.current}, nil
}
