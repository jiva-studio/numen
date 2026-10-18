package completion

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	errNoConfig  = errors.New("a router needs the settings to read")
	errNoChoices = errors.New("a router needs the choices of the plugins")
	errNoAgents  = errors.New("a router needs the agents to open")
)

// Router answers a plugin's question with the agent the plugin's setting names,
// else with the default of the settings.
type Router struct {
	config  func() Config
	choices port.CompletionChoices
	agents  port.CompletionAgents
	limits  *limits
}

// NewRouter returns a router reading the settings and the plugin's choice at
// every question, so a choice edited in the settings applies to the next one.
func NewRouter(config func() Config, choices port.CompletionChoices, agents port.CompletionAgents) (*Router, error) {
	if config == nil {
		return nil, errNoConfig
	}
	if choices == nil {
		return nil, errNoChoices
	}
	if agents == nil {
		return nil, errNoAgents
	}
	return &Router{config: config, choices: choices, agents: agents, limits: newLimits()}, nil
}

// Complete answers the question of the plugin with the id given. A plugin whose
// setting names an agent is answered by that agent alone, with the model the
// setting names; one naming none is answered by the default agent and model.
// An agent that is not there fails with port.ErrNoRoute. The wait for a place
// under the agent's limit, and the answer, end with the context.
func (r *Router) Complete(ctx context.Context, pluginID string, question port.Question) (string, error) {
	defaults := r.config().Default
	route := getRoute(r.choices.GetCompletionChoice(pluginID), defaults)
	release, err := r.limits.acquire(ctx, route.Agent, defaults.GetLimit())
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

// getRoute is the agent and the model of the choice, or the default where the
// choice names no agent.
func getRoute(choice port.CompletionChoice, defaults Route) Route {
	if choice.Agent == "" {
		return Route{Agent: defaults.Agent, Model: defaults.Model}
	}
	return Route{Agent: choice.Agent, Model: choice.Model}
}
