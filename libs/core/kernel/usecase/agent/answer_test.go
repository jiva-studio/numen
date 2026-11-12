package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
)

// run replays steps and records that it was stopped.
type run struct {
	steps   chan agentdomain.Step
	stopped int
}

func newRun(steps ...agentdomain.Step) *run {
	r := &run{steps: make(chan agentdomain.Step, len(steps))}
	for _, step := range steps {
		r.steps <- step
	}
	close(r.steps)
	return r
}

func (r *run) GetSteps() <-chan agentdomain.Step { return r.steps }
func (r *run) Stop() error                       { r.stopped++; return nil }

// starter starts the run it holds and records the task it was given.
type starter struct {
	run  *run
	err  error
	task agentdomain.Task
}

func (s *starter) StartRun(_ context.Context, task agentdomain.Task) (agentport.Run, error) {
	s.task = task
	if s.err != nil {
		return nil, s.err
	}
	return s.run, nil
}

func say(text string) agentdomain.Step { return agentdomain.Step{Kind: domain.StepSay, Text: text} }
func stop(why string) agentdomain.Step { return agentdomain.Step{Kind: domain.StepStop, Detail: why} }

// The pieces the agent emits are the answer, in arrival order, and the run is
// stopped once it has been read.
func TestAnswer_EqualsAgentOutput(t *testing.T) {
	target := &starter{run: newRun(
		agentdomain.Step{Kind: domain.StepModelStart}, say("The "), say("answer."), stop(""),
	)}

	got, err := agent.NewAnswer(target).Execute(t.Context(), agentdomain.Task{Question: "why?"})

	if err != nil || got != "The answer." {
		t.Fatalf("got %q, %v", got, err)
	}
	if target.run.stopped != 1 {
		t.Errorf("the run was stopped %d times, want 1", target.run.stopped)
	}
}

// An agent that stops with "done", or closes its steps without a stop, has answered.
func TestAnswer_CompletedAgentHasAnswer(t *testing.T) {
	for name, steps := range map[string][]agentdomain.Step{
		"done":    {say("yes"), stop("done")},
		"no stop": {say("yes")},
	} {
		got, err := agent.NewAnswer(&starter{run: newRun(steps...)}).Execute(t.Context(), agentdomain.Task{})
		if err != nil || got != "yes" {
			t.Errorf("%s: got %q, %v", name, got, err)
		}
	}
}

// The task is sent as a single answer: no tool, no endpoint, no conversation, and the rest as given.
func TestTheTaskIsSentWithNoToolAndNoConversation(t *testing.T) {
	target := &starter{run: newRun(say("yes"), stop(""))}
	task := agentdomain.Task{
		Question:         "why?",
		Instructions:     "be brief",
		Agent:            "claude",
		Model:            "m1",
		Conversation:     "c-1",
		Tools:            []string{"note_edit"},
		ToolDeclarations: []agentdomain.ToolDeclaration{{Name: "note_edit"}},
		Endpoint:         agentdomain.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "secret"},
	}

	if _, err := agent.NewAnswer(target).Execute(t.Context(), task); err != nil {
		t.Fatal(err)
	}

	want := agentdomain.Task{Question: "why?", Instructions: "be brief", Agent: "claude", Model: "m1"}
	if got := target.task; got.Question != want.Question || got.Instructions != want.Instructions ||
		got.Agent != want.Agent || got.Model != want.Model || got.Conversation != "" ||
		len(got.Tools) != 0 || len(got.ToolDeclarations) != 0 || got.Endpoint != (agentdomain.ToolEndpoint{}) {
		t.Errorf("the agent was given %+v", got)
	}
}

// What the agent emits about a tool is not part of the answer.
func TestOnlyWhatTheAgentReturnsIsTheAnswer(t *testing.T) {
	target := &starter{run: newRun(
		agentdomain.Step{Kind: domain.StepToolCall, Tool: "note_read", Text: "no"}, say("yes"), stop(""),
	)}

	if got, err := agent.NewAnswer(target).Execute(t.Context(), agentdomain.Task{}); err != nil || got != "yes" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestAnAgentThatCannotBeStartedIsUnavailable(t *testing.T) {
	_, err := agent.NewAnswer(&starter{err: errors.New("no plugin runs it")}).Execute(t.Context(), agentdomain.Task{})

	if !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("got %v, want port.ErrUnavailable", err)
	}
}

func TestAnAgentThatStopsForAReasonIsUnavailableWithThatReason(t *testing.T) {
	target := &starter{run: newRun(say("partial"), stop("not signed in"))}

	_, err := agent.NewAnswer(target).Execute(t.Context(), agentdomain.Task{})

	if !errors.Is(err, domain.ErrUnavailable) || !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("got %v, want port.ErrUnavailable naming the reason", err)
	}
}

func TestAnAgentThatReturnsNothingIsUnavailable(t *testing.T) {
	for name, steps := range map[string][]agentdomain.Step{
		"only a stop": {stop("")},
		"blank":       {say(" \n"), stop("")},
		"no steps":    {},
	} {
		_, err := agent.NewAnswer(&starter{run: newRun(steps...)}).Execute(t.Context(), agentdomain.Task{})
		if !errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("%s: got %v, want port.ErrUnavailable", name, err)
		}
	}
}

// The end of the caller's context is the caller's own, not an unavailability.
func TestTheEndOfTheContextIsNotAnUnavailability(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	for name, target := range map[string]*starter{
		"not started": {err: errors.New("canceled")},
		"no answer":   {run: newRun()},
		"stopped":     {run: newRun(stop("canceled"))},
	} {
		_, err := agent.NewAnswer(target).Execute(ctx, agentdomain.Task{})
		if !errors.Is(err, context.Canceled) || errors.Is(err, domain.ErrUnavailable) {
			t.Errorf("%s: got %v, want context.Canceled alone", name, err)
		}
	}
}

// A run that never closes its steps does not hold the answer past the end of the context.
func TestTheEndOfTheContextEndsTheWaitForSteps(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	target := &starter{run: &run{steps: make(chan agentdomain.Step)}}
	isDone := make(chan error, 1)
	go func() {
		_, err := agent.NewAnswer(target).Execute(ctx, agentdomain.Task{})
		isDone <- err
	}()

	cancel()

	if err := <-isDone; !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
	if target.run.stopped != 1 {
		t.Errorf("the run was stopped %d times, want 1", target.run.stopped)
	}
}
