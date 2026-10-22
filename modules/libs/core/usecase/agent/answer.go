package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// stoppedDone is the reason an agent gives when it stopped because it was done.
const stoppedDone = "done"

var errNoAnswer = errors.New("agent returned empty response")

// Starter starts the tasks an agent is given.
type Starter interface {
	StartRun(ctx context.Context, task port.Task) (port.Run, error)
}

// Answer gives the agent one question and returns what it said. The task is sent with no tool, no
// tool endpoint and no conversation, so the agent answers on its own and is given nothing of the
// vault. An agent that cannot be started, stops for a reason of its own or says nothing fails with
// port.ErrUnavailable. The end of the context stops the run and fails with the context's error.
func Answer(ctx context.Context, agent Starter, task port.Task) (string, error) {
	task.Tools = nil
	task.ToolDeclarations = nil
	task.Endpoint = port.ToolEndpoint{}
	task.Conversation = ""

	run, err := agent.StartRun(ctx, task)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", fmt.Errorf("%w: %w", port.ErrUnavailable, err)
	}
	defer func() { _ = run.Stop() }()
	return readAnswer(ctx, run.GetSteps())
}

// readAnswer is what the agent said in the steps it takes, until they are over or the context ends.
func readAnswer(ctx context.Context, steps <-chan port.Step) (string, error) {
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

// finishAnswer is the text the agent said, once its steps are over. The end of the context makes it
// no answer of the agent's.
func finishAnswer(ctx context.Context, response string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(response) == "" {
		return "", fmt.Errorf("%w: %w", port.ErrUnavailable, errNoAnswer)
	}
	return response, nil
}

// failAnswer is the failure of an answer: the end of the caller's context is the caller's own, and
// any other is an unavailability.
func failAnswer(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
}
