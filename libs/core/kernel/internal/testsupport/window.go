package testsupport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// SilentAgent accepts a task and produces no output, as an agent waiting on a
// model does.
type SilentAgent struct{}

func (SilentAgent) StartRun(context.Context, agent.Task) (agentport.Run, error) {
	return NoopRun{steps: make(chan agent.Step)}, nil
}

func (SilentAgent) EndConversation(context.Context, string) error { return nil }

// NoopRun is a run of a SilentAgent: a step never arrives, and stopping it
// costs nothing.
type NoopRun struct{ steps chan agent.Step }

func (w NoopRun) GetSteps() <-chan agent.Step { return w.steps }
func (w NoopRun) Stop() error                 { return nil }
