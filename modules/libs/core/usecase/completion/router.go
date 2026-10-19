package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/agent"
)

var (
	errNoConfig   = errors.New("a router needs the settings to read")
	errNoChoices  = errors.New("a router needs the choices of the plugins")
	errNoPrograms = errors.New("a router needs the programs to ask")
	errNoMessage  = errors.New("the question holds no message to answer")
)

// Router answers a plugin's question with the agent the plugin's setting names,
// else with the default of the settings.
type Router struct {
	config   func() Config
	choices  port.CompletionChoices
	programs port.CompletionPrograms
	limits   *limits
}

// NewRouter returns a router reading the settings and the plugin's choice at
// every question, so a choice edited in the settings applies to the next one.
func NewRouter(config func() Config, choices port.CompletionChoices, programs port.CompletionPrograms) (*Router, error) {
	if config == nil {
		return nil, errNoConfig
	}
	if choices == nil {
		return nil, errNoChoices
	}
	if programs == nil {
		return nil, errNoPrograms
	}
	return &Router{config: config, choices: choices, programs: programs, limits: newLimits()}, nil
}

// Complete answers the question of the plugin with the id given. A plugin whose
// setting names an agent is answered by that agent alone, with the model the
// setting names; one naming none is answered by the default agent and model.
// The question goes to the agent as a task that names no tool and no
// conversation. A program that is not there fails with port.ErrNoRoute. The wait
// for a place under the agent's limit, and the answer, end with the context.
func (r *Router) Complete(ctx context.Context, pluginID string, question port.Question) (string, error) {
	task, err := newTask(question)
	if err != nil {
		return "", err
	}
	defaults := r.config().Default
	route := getRoute(r.choices.GetCompletionChoice(pluginID), defaults)
	release, err := r.limits.acquire(ctx, route.Agent, defaults.GetLimit())
	if err != nil {
		return "", err
	}
	defer release()

	program, err := r.programs.GetProgram(ctx, route.Agent)
	if err != nil {
		return "", err
	}
	task.Agent = program.Name
	task.Model = route.Model
	return agent.Answer(ctx, r.programs, task)
}

// getRoute is the agent and the model of the choice, or the default where the
// choice names no agent.
func getRoute(choice port.CompletionChoice, defaults Route) Route {
	if choice.Agent == "" {
		return Route{Agent: defaults.Agent, Model: defaults.Model}
	}
	return Route{Agent: choice.Agent, Model: choice.Model}
}

// newTask is the question as a task: the system messages are the instructions,
// and the other messages are the prompt, as written when there is one and with
// their roles when there are several.
func newTask(question port.Question) (port.Task, error) {
	var system []string
	var turns []port.Message
	for _, message := range question.Messages {
		switch message.Role {
		case "system":
			system = append(system, message.Text)
		case "", "user", "assistant":
			turns = append(turns, message)
		default:
			return port.Task{}, fmt.Errorf("%w: unknown role %q", port.ErrInvalidQuestion, message.Role)
		}
	}
	if len(turns) == 0 {
		return port.Task{}, fmt.Errorf("%w: %w", port.ErrInvalidQuestion, errNoMessage)
	}
	return port.Task{Question: writePrompt(turns), Instructions: strings.Join(system, "\n\n")}, nil
}

func writePrompt(turns []port.Message) string {
	if len(turns) == 1 {
		return turns[0].Text
	}
	parts := make([]string, 0, len(turns))
	for _, turn := range turns {
		role := "User"
		if turn.Role == "assistant" {
			role = "Assistant"
		}
		parts = append(parts, role+":\n"+turn.Text)
	}
	return strings.Join(parts, "\n\n")
}
