package completion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	completiondomain "github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

var (
	errNoMessage = errors.New("the question holds no message to answer")
)

// Answerer sends the agent one question and returns its reply.
type Answerer interface {
	Execute(ctx context.Context, task agent.Task) (string, error)
}

// ProgramSource finds the program a route names and starts the tasks that
// carry the questions to it.
type ProgramSource interface {
	// GetProgram returns the program named, or, when the name is empty, the one
	// the settings name, else the default of the plugins. A program that is not
	// installed or that no plugin lists fails with completion.ErrNoRoute, and plugins that
	// cannot be reached with ErrUnavailable.
	GetProgram(ctx context.Context, name string) (agent.Program, error)
	// StartRun starts the task on the program it names, under that program's
	// section of the settings.
	StartRun(ctx context.Context, task agent.Task) (agentport.Run, error)
}

// Router answers a plugin's question with the default agent and model of the
// settings.
type Router struct {
	config   func() Config
	programs ProgramSource
	answerer Answerer
	limits   *limits
}

// NewRouter returns a router reading the settings at every question, so a default
// edited in the settings applies to the next one.
func NewRouter(config func() Config, programs ProgramSource, answerer Answerer) *Router {
	return &Router{config: config, programs: programs, answerer: answerer, limits: newLimits()}
}

// Complete answers the question of a plugin by the default agent and model. The
// question goes to the agent as a task that names no tool and no conversation. A model the program does not list is replaced by the one it answers with unasked,
// else its first; a failure then states the replacement. A program that is not there fails with
// port.ErrNoRoute. The wait for a place under the agent's limit, and the answer, end with the
// context.
func (r *Router) Complete(ctx context.Context, _ string, question completiondomain.Question) (string, error) {
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
	text, err := r.answerer.Execute(ctx, task)
	if err != nil && isReplaced {
		return "", fmt.Errorf("%w (the model %q is not available from %s; requested %q)", err, route.Model, program.Name, model)
	}
	return text, err
}

func getReplacementModel(program agent.Program, selected string) (string, bool) {
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

func newTask(question completiondomain.Question) (agent.Task, error) {
	var system []string
	var turns []completiondomain.Message
	for _, message := range question.Messages {
		switch message.Role {
		case "system":
			system = append(system, message.Text)
		case "", "user", "assistant":
			turns = append(turns, message)
		default:
			return agent.Task{}, fmt.Errorf("%w: unknown role %q", completiondomain.ErrInvalidQuestion, message.Role)
		}
	}
	if len(turns) == 0 {
		return agent.Task{}, fmt.Errorf("%w: %w", completiondomain.ErrInvalidQuestion, errNoMessage)
	}
	return agent.Task{Question: writePrompt(turns), Instructions: strings.Join(system, "\n\n")}, nil
}

func writePrompt(turns []completiondomain.Message) string {
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
