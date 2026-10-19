package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// stoppedDone is the reason an agent gives when it stopped because it was done.
const stoppedDone = "done"

var errNoAnswer = errors.New("the agent said nothing")

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

	var said strings.Builder
	for step := range run.GetSteps() {
		switch step.Kind {
		case domain.StepSay:
			said.WriteString(step.Text)
		case domain.StepStop:
			if step.Detail != "" && step.Detail != stoppedDone {
				return "", failAnswer(ctx, fmt.Errorf("the agent stopped: %s", step.Detail))
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(said.String()) == "" {
		return "", fmt.Errorf("%w: %w", port.ErrUnavailable, errNoAnswer)
	}
	return said.String(), nil
}

// failAnswer is the failure of an answer: the end of the caller's context is the caller's own, and
// any other is an unavailability.
func failAnswer(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %w", port.ErrUnavailable, err)
}
