package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// stoppedDone is the reason an agent gives when it stopped because it was done.
const stoppedDone = "done"

var errNoAnswer = errors.New("agent returned empty response")

// Starter starts the tasks an agent is given.
type Starter interface {
	StartRun(ctx context.Context, task agentdomain.Task) (agentport.Run, error)
}

// Answer sends the agent one question and returns its reply.
type Answer struct {
	agent Starter
}

func NewAnswer(agent Starter) Answer {
	return Answer{agent: agent}
}

// Execute sends the agent the task and returns its reply. The task is sent with no tool, no tool
// endpoint and no conversation, so the agent answers on its own and has no access to the vault. An
// agent that cannot be started, stops for its own reason or returns nothing fails with
// domain.ErrUnavailable. A context that ends stops the run and fails with the context's error.
func (u Answer) Execute(ctx context.Context, task agentdomain.Task) (_ string, err error) {
	task.Tools = nil
	task.ToolDeclarations = nil
	task.Endpoint = agentdomain.ToolEndpoint{}
	task.Conversation = ""

	run, err := u.agent.StartRun(ctx, task)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf("%w: %w", domain.ErrUnavailable, err)
	}
	defer func() { err = errors.Join(err, run.Stop()) }()
	return readAnswer(ctx, run.GetSteps())
}

func readAnswer(ctx context.Context, steps <-chan agentdomain.Step) (string, error) {
	var response strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case step, isOpen := <-steps:
			if !isOpen {
				return finishAnswer(ctx, response.String())
			}
			if step.Kind == domain.StepSay {
				response.WriteString(step.Text)
			}
			if step.Kind == domain.StepStop && step.Detail != "" && step.Detail != stoppedDone {
				return "", failAnswer(ctx, fmt.Errorf("the agent stopped: %s", step.Detail))
			}
		}
	}
}

func finishAnswer(ctx context.Context, response string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(response) == "" {
		return "", fmt.Errorf("%w: %w", domain.ErrUnavailable, errNoAnswer)
	}
	return response, nil
}

func failAnswer(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %w", domain.ErrUnavailable, err)
}
