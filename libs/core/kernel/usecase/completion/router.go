package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/completionport"
)

var (
	errNoConfig   = errors.New("a router needs the settings to read")
	errNoPrograms = errors.New("a router needs the programs to ask")
	errNoMessage  = errors.New("the question holds no message to answer")
)

// Router answers a plugin's question with the default agent and model of the
// settings.
type Router struct {
	config   func() Config
	programs agentport.CompletionPrograms
	limits   *limits
}

// NewRouter returns a router reading the settings at every question, so a default
// edited in the settings applies to the next one.
func NewRouter(config func() Config, programs agentport.CompletionPrograms) (*Router, error) {
	if config == nil {
		return nil, errNoConfig
	}
	if programs == nil {
		return nil, errNoPrograms
	}
	return &Router{config: config, programs: programs, limits: newLimits()}, nil
}

// Complete answers the question of a plugin by the default agent and model. The
// question goes to the agent as a task that names no tool and no conversation. A model the program does not list is replaced by the one it answers with unasked,
// else its first; a failure then states the replacement. A program that is not there fails with
// port.ErrNoRoute. The wait for a place under the agent's limit, and the answer, end with the
// context.
func (r *Router) Complete(ctx context.Context, _ string, question completionport.Question) (string, error) {
	task, err := newTask(question)
	if err != nil {
		return "", err
	}
	defaults := r.config().Default
	route := Route{Agent: defaults.Agent, Model: defaults.Model}
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
	model, isReplaced := getReplacementModel(program, route.Model)
	if isReplaced {
		task.Model = model
	}
	text, err := agentport.Answer(ctx, r.programs, task)
	if err != nil && isReplaced {
		return "", fmt.Errorf("%w (the model %q is not available from %s; requested %q)", err, route.Model, program.Name, model)
	}
	return text, err
}

func getReplacementModel(program agentport.AgentProgram, selected string) (string, bool) {
	if selected == "" || len(program.Models) == 0 {
		return "", false
	}
	for _, one := range program.Models {
		if one.Name == selected {
			return "", false
		}
	}
	for _, one := range program.Models {
		if one.IsDefault {
			return one.Name, true
		}
	}
	return program.Models[0].Name, true
}

func newTask(question completionport.Question) (agentport.Task, error) {
	var system []string
	var turns []completionport.Message
	for _, message := range question.Messages {
		switch message.Role {
		case "system":
			system = append(system, message.Text)
		case "", "user", "assistant":
			turns = append(turns, message)
		default:
			return agentport.Task{}, fmt.Errorf("%w: unknown role %q", completionport.ErrInvalidQuestion, message.Role)
		}
	}
	if len(turns) == 0 {
		return agentport.Task{}, fmt.Errorf("%w: %w", completionport.ErrInvalidQuestion, errNoMessage)
	}
	return agentport.Task{Question: writePrompt(turns), Instructions: strings.Join(system, "\n\n")}, nil
}

func writePrompt(turns []completionport.Message) string {
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
