package agent_test

import (
	"context"
	"errors"
	"testing"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
)

func TestStartTaskReturnsTheRunOfTheAgent(t *testing.T) {
	source := &starter{run: newRun(say("Hello."))}

	work, err := agent.NewStartTask(source).Execute(t.Context(), agentdomain.Task{Question: "hi", Root: "/vaults/Garden"})

	if err != nil {
		t.Fatal(err)
	}
	if source.task.Question != "hi" || source.task.Root != "/vaults/Garden" {
		t.Errorf("the agent received %+v", source.task)
	}
	if step := <-work.Steps; step.Text != "Hello." {
		t.Errorf("the first step is %+v", step)
	}
	if err := work.Stop(); err != nil || source.run.stopped != 1 {
		t.Errorf("stopping answered %v and stopped the run %d times", err, source.run.stopped)
	}
}

func TestStartTaskReturnsTheFailureOfTheAgent(t *testing.T) {
	failure := errors.New("the program is gone")

	_, err := agent.NewStartTask(&starter{err: failure}).Execute(t.Context(), agentdomain.Task{})

	if !errors.Is(err, failure) {
		t.Fatalf("got %v", err)
	}
}

// closer records the conversations that were ended.
type closer struct{ ended []string }

func (c *closer) EndConversation(_ context.Context, conversation string) error {
	c.ended = append(c.ended, conversation)
	return nil
}

func TestEndConversationEndsOnlyAConversationThatIsNamed(t *testing.T) {
	ending := &closer{}
	end := agent.NewEndConversation(ending)

	if err := end.Execute(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	if err := end.Execute(t.Context(), ""); err != nil {
		t.Fatal(err)
	}

	if len(ending.ended) != 1 || ending.ended[0] != "c-1" {
		t.Errorf("the agent ended %v, want only c-1", ending.ended)
	}
}
