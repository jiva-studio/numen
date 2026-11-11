package agent

import (
	"context"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
)

// Run is a task being worked. Steps are the steps in the order the agent takes them, and the
// channel closes when there are no more. Stop ends the run and returns once nothing of it is still
// running.
type Run struct {
	Steps <-chan agentdomain.Step
	Stop  func() error
}

// StartTask gives the agent a task and returns the run it has begun.
type StartTask struct {
	agent Starter
}

// NewStartTask returns the use case over the agent.
func NewStartTask(agent Starter) *StartTask {
	return &StartTask{agent: agent}
}

// Execute starts the task on the agent. The caller stops the run it gets.
func (u *StartTask) Execute(ctx context.Context, task agentdomain.Task) (Run, error) {
	run, err := u.agent.StartRun(ctx, task)
	if err != nil {
		return Run{}, err
	}
	return Run{Steps: run.GetSteps(), Stop: run.Stop}, nil
}

// Closer ends the conversations of an agent.
type Closer interface {
	// EndConversation ends a conversation. The agent releases its state for the conversation and
	// stops and waits for any work still running in it.
	EndConversation(ctx context.Context, conversation string) error
}

type EndConversation struct {
	agent Closer
}

// NewEndConversation returns the use case over the agent.
func NewEndConversation(agent Closer) *EndConversation {
	return &EndConversation{agent: agent}
}

// Execute ends the conversation. An empty conversation is a no-op.
func (u *EndConversation) Execute(ctx context.Context, conversation string) error {
	if conversation == "" {
		return nil
	}
	return u.agent.EndConversation(ctx, conversation)
}
