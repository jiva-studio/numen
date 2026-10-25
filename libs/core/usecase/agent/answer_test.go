package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/agent"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
)

// run replays steps and records that it was stopped.
type run struct {
	steps   chan agentport.Step
	stopped int
}

func newRun(steps ...agentport.Step) *run {
	r := &run{steps: make(chan agentport.Step, len(steps))}
	for _, step := range steps {
		r.steps <- step
	}
	close(r.steps)
	return r
}

func (r *run) GetSteps() <-chan agentport.Step { return r.steps }
func (r *run) Stop() error                     { r.stopped++; return nil }

// starter starts the run it holds and records the task it was given.
type starter struct {
	run  *run
	err  error
	task agentport.Task
}

func (s *starter) StartRun(_ context.Context, task agentport.Task) (agentport.Run, error) {
	s.task = task
	if s.err != nil {
		return nil, s.err
	}
	return s.run, nil
}

func say(text string) agentport.Step { return agentport.Step{Kind: domain.StepSay, Text: text} }
func stop(why string) agentport.Step { return agentport.Step{Kind: domain.StepStop, Detail: why} }

// The pieces the agent says are the answer, in the order they arrive, and the run is stopped once
// it has been read.
func TestTheAnswerIsWhatTheAgentSaid(t *testing.T) {
	target := &starter{run: newRun(
		agentport.Step{Kind: domain.StepModelStart}, say("The "), say("answer."), stop(""),
	)}

	got, err := agent.Answer(t.Context(), target, agentport.Task{Question: "why?"})

	if err != nil || got != "The answer." {
		t.Fatalf("got %q, %v", got, err)
	}
	if target.run.stopped != 1 {
		t.Errorf("the run was stopped %d times, want 1", target.run.stopped)
	}
}

// An agent that stops with "done", or closes its steps without a stop, has answered.
func TestAnAgentThatIsCompletedHasAnswered(t *testing.T) {
	for name, steps := range map[string][]agentport.Step{
		"done":    {say("yes"), stop("done")},
		"no stop": {say("yes")},
	} {
		got, err := agent.Answer(t.Context(), &starter{run: newRun(steps...)}, agentport.Task{})
		if err != nil || got != "yes" {
			t.Errorf("%s: got %q, %v", name, got, err)
		}
	}
}

// The task is sent as a single answer: no tool, no endpoint, no conversation, and the rest as given.
func TestTheTaskIsSentWithNoToolAndNoConversation(t *testing.T) {
	target := &starter{run: newRun(say("yes"), stop(""))}
	task := agentport.Task{
		Question:         "why?",
		Instructions:     "be brief",
		Agent:            "claude",
		Model:            "m1",
		Conversation:     "c-1",
		Tools:            []string{"note_edit"},
		ToolDeclarations: []agentport.ToolDeclaration{{Name: "note_edit"}},
		Endpoint:         agentport.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "secret"},
	}

	if _, err := agent.Answer(t.Context(), target, task); err != nil {
		t.Fatal(err)
	}

	want := agentport.Task{Question: "why?", Instructions: "be brief", Agent: "claude", Model: "m1"}
	if got := target.task; got.Question != want.Question || got.Instructions != want.Instructions ||
		got.Agent != want.Agent || got.Model != want.Model || got.Conversation != "" ||
		len(got.Tools) != 0 || len(got.ToolDeclarations) != 0 || got.Endpoint != (agentport.ToolEndpoint{}) {
		t.Errorf("the agent was given %+v", got)
	}
}

// What the agent says about a tool is not part of the answer.
func TestOnlyWhatTheAgentReturnsIsTheAnswer(t *testing.T) {
	target := &starter{run: newRun(
		agentport.Step{Kind: domain.StepToolCall, Tool: "note_read", Text: "no"}, say("yes"), stop(""),
	)}

	if got, err := agent.Answer(t.Context(), target, agentport.Task{}); err != nil || got != "yes" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestAnAgentThatCannotBeStartedIsUnavailable(t *testing.T) {
	_, err := agent.Answer(t.Context(), &starter{err: errors.New("no plugin runs it")}, agentport.Task{})

	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("got %v, want port.ErrUnavailable", err)
	}
}

func TestAnAgentThatStopsForAReasonIsUnavailableWithThatReason(t *testing.T) {
	target := &starter{run: newRun(say("partial"), stop("not signed in"))}

	_, err := agent.Answer(t.Context(), target, agentport.Task{})

	if !errors.Is(err, completionport.ErrUnavailable) || !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("got %v, want port.ErrUnavailable naming the reason", err)
	}
}

func TestAnAgentThatReturnsNothingIsUnavailable(t *testing.T) {
	for name, steps := range map[string][]agentport.Step{
		"only a stop": {stop("")},
		"blank":       {say(" \n"), stop("")},
		"no steps":    {},
	} {
		_, err := agent.Answer(t.Context(), &starter{run: newRun(steps...)}, agentport.Task{})
		if !errors.Is(err, completionport.ErrUnavailable) {
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
		_, err := agent.Answer(ctx, target, agentport.Task{})
		if !errors.Is(err, context.Canceled) || errors.Is(err, completionport.ErrUnavailable) {
			t.Errorf("%s: got %v, want context.Canceled alone", name, err)
		}
	}
}

// A run that never closes its steps does not hold the answer past the end of the context.
func TestTheEndOfTheContextEndsTheWaitForSteps(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	target := &starter{run: &run{steps: make(chan agentport.Step)}}
	isDone := make(chan error, 1)
	go func() {
		_, err := agent.Answer(ctx, target, agentport.Task{})
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
