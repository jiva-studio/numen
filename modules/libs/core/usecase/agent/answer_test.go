package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/agent"
)

// run replays steps and records that it was stopped.
type run struct {
	steps   chan port.Step
	stopped int
}

func newRun(steps ...port.Step) *run {
	r := &run{steps: make(chan port.Step, len(steps))}
	for _, step := range steps {
		r.steps <- step
	}
	close(r.steps)
	return r
}

func (r *run) GetSteps() <-chan port.Step { return r.steps }
func (r *run) Stop() error                { r.stopped++; return nil }

// starter starts the run it holds and records the task it was given.
type starter struct {
	run  *run
	err  error
	task port.Task
}

func (s *starter) StartRun(_ context.Context, task port.Task) (port.Run, error) {
	s.task = task
	if s.err != nil {
		return nil, s.err
	}
	return s.run, nil
}

func say(text string) port.Step { return port.Step{Kind: domain.StepSay, Text: text} }
func stop(why string) port.Step { return port.Step{Kind: domain.StepStop, Detail: why} }

// The pieces the agent says are the answer, in the order they arrive, and the run is stopped once
// it has been read.
func TestTheAnswerIsWhatTheAgentSaid(t *testing.T) {
	held := &starter{run: newRun(
		port.Step{Kind: domain.StepModelStart}, say("The "), say("answer."), stop(""),
	)}

	got, err := agent.Answer(t.Context(), held, port.Task{Question: "why?"})

	if err != nil || got != "The answer." {
		t.Fatalf("got %q, %v", got, err)
	}
	if held.run.stopped != 1 {
		t.Errorf("the run was stopped %d times, want 1", held.run.stopped)
	}
}

// An agent that stops with "done", or closes its steps without a stop, has answered.
func TestAnAgentThatIsDoneHasAnswered(t *testing.T) {
	for name, steps := range map[string][]port.Step{
		"done":    {say("yes"), stop("done")},
		"no stop": {say("yes")},
	} {
		got, err := agent.Answer(t.Context(), &starter{run: newRun(steps...)}, port.Task{})
		if err != nil || got != "yes" {
			t.Errorf("%s: got %q, %v", name, got, err)
		}
	}
}

// The task is sent as a single answer: no tool, no endpoint, no conversation, and the rest as given.
func TestTheTaskIsSentWithNoToolAndNoConversation(t *testing.T) {
	held := &starter{run: newRun(say("yes"), stop(""))}
	task := port.Task{
		Question:         "why?",
		Instructions:     "be brief",
		Agent:            "claude",
		Model:            "m1",
		Conversation:     "c-1",
		Tools:            []string{"note_edit"},
		ToolDeclarations: []port.ToolDeclaration{{Name: "note_edit"}},
		Endpoint:         port.ToolEndpoint{URL: "http://127.0.0.1:1/mcp", Token: "secret"},
	}

	if _, err := agent.Answer(t.Context(), held, task); err != nil {
		t.Fatal(err)
	}

	want := port.Task{Question: "why?", Instructions: "be brief", Agent: "claude", Model: "m1"}
	if got := held.task; got.Question != want.Question || got.Instructions != want.Instructions ||
		got.Agent != want.Agent || got.Model != want.Model || got.Conversation != "" ||
		len(got.Tools) != 0 || len(got.ToolDeclarations) != 0 || got.Endpoint != (port.ToolEndpoint{}) {
		t.Errorf("the agent was given %+v", got)
	}
}

// What the agent says about a tool is not part of the answer.
func TestOnlyWhatTheAgentSaysIsTheAnswer(t *testing.T) {
	held := &starter{run: newRun(
		port.Step{Kind: domain.StepToolCall, Tool: "note_read", Text: "no"}, say("yes"), stop(""),
	)}

	if got, err := agent.Answer(t.Context(), held, port.Task{}); err != nil || got != "yes" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestAnAgentThatCannotBeStartedIsUnavailable(t *testing.T) {
	_, err := agent.Answer(t.Context(), &starter{err: errors.New("no plugin runs it")}, port.Task{})

	if !errors.Is(err, port.ErrUnavailable) {
		t.Errorf("got %v, want port.ErrUnavailable", err)
	}
}

func TestAnAgentThatStopsForAReasonIsUnavailableWithThatReason(t *testing.T) {
	held := &starter{run: newRun(say("partial"), stop("not signed in"))}

	_, err := agent.Answer(t.Context(), held, port.Task{})

	if !errors.Is(err, port.ErrUnavailable) || !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("got %v, want port.ErrUnavailable naming the reason", err)
	}
}

func TestAnAgentThatSaysNothingIsUnavailable(t *testing.T) {
	for name, steps := range map[string][]port.Step{
		"only a stop": {stop("")},
		"blank":       {say(" \n"), stop("")},
		"no steps":    {},
	} {
		_, err := agent.Answer(t.Context(), &starter{run: newRun(steps...)}, port.Task{})
		if !errors.Is(err, port.ErrUnavailable) {
			t.Errorf("%s: got %v, want port.ErrUnavailable", name, err)
		}
	}
}

// The end of the caller's context is the caller's own, not an unavailability.
func TestTheEndOfTheContextIsNotAnUnavailability(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	for name, held := range map[string]*starter{
		"not started": {err: errors.New("canceled")},
		"no answer":   {run: newRun()},
		"stopped":     {run: newRun(stop("canceled"))},
	} {
		_, err := agent.Answer(ctx, held, port.Task{})
		if !errors.Is(err, context.Canceled) || errors.Is(err, port.ErrUnavailable) {
			t.Errorf("%s: got %v, want context.Canceled alone", name, err)
		}
	}
}
