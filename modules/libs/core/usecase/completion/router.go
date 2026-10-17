package completion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoRoutes = errors.New("a router needs the routes to read")
	errNoAgents = errors.New("a router needs the agents to open")
)

// Router answers a question with the agent its purpose is routed to.
type Router struct {
	routes func() Config
	agents port.CompletionAgents
	limits *limits
}

// NewRouter returns a router reading its routes from routes at every question,
// so a route edited in the settings applies to the next one.
func NewRouter(routes func() Config, agents port.CompletionAgents) (*Router, error) {
	if routes == nil {
		return nil, errNoRoutes
	}
	if agents == nil {
		return nil, errNoAgents
	}
	return &Router{routes: routes, agents: agents, limits: newLimits()}, nil
}

// Complete answers the question. A purpose without a route, and without a
// default route, fails with port.ErrNoRoute. The wait for a place under the
// route's limit, and the answer, end with the context.
func (r *Router) Complete(ctx context.Context, question port.Question) (string, error) {
	name, route, found := r.routes().GetRoute(question.Purpose)
	if !found {
		return "", fmt.Errorf("%w %q", port.ErrNoRoute, question.Purpose)
	}
	release, err := r.limits.acquire(ctx, name, route.GetLimit())
	if err != nil {
		return "", err
	}
	defer release()

	agent, err := r.agents.OpenCompletionAgent(ctx, route.Agent)
	if err != nil {
		return "", err
	}
	// Close lets the agent go and has nothing to say about an answer already read.
	defer func() { _ = agent.Close() }()
	return agent.Complete(ctx, route.Model, question)
}
