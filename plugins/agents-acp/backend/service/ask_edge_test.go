package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

var errSend = errors.New("send failed")

func TestATaskStopsWhenTheFirstStepCannotBeSent(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "plain")
	calls := 0

	err := serveAgents(t, agent).Ask(t.Context(), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(*agentrun.Response) error {
		calls++
		return errSend
	})

	if !errors.Is(err, errSend) || calls != 1 {
		t.Errorf("Ask on initial send error = %v (calls %d)", err, calls)
	}
}

func TestATaskForAMissingAgentReturnsTheErrorOfTheStoppedStep(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "plain")
	calls := 0

	err := serveAgents(t, agent).Ask(t.Context(), &agentrun.Request{Agent: "acp:absent", Prompt: "why?"}, func(res *agentrun.Response) error {
		calls++
		if res.GetStopped() != "" {
			return errSend
		}
		return nil
	})

	if !errors.Is(err, errSend) || calls != 2 {
		t.Errorf("Ask on stopped send error = %v (calls %d)", err, calls)
	}
}

func TestAnAnswerThatCannotBeSentIsNotFollowedByAStoppedStep(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "plain")
	calls := 0

	err := serveAgents(t, agent).Ask(t.Context(), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(res *agentrun.Response) error {
		calls++
		if res.GetText() != "" {
			return errSend
		}
		return nil
	})

	if !errors.Is(err, errSend) || calls != 2 {
		t.Errorf("Ask on text send error = %v (calls %d)", err, calls)
	}
}

func TestACanceledTaskReturnsTheCancellationAndSendsNoReason(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "plain")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var stopReason string

	err := serveAgents(t, agent).Ask(ctx, &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(res *agentrun.Response) error {
		if s := res.GetStopped(); s != "" {
			stopReason = s
		}
		return nil
	})

	if !errors.Is(err, context.Canceled) || stopReason != "" {
		t.Errorf("Ask on canceled ctx = %v, stopReason = %q", err, stopReason)
	}
}

func TestATaskStopsWithTheReasonTheAgentRejectsItsDepth(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "fail-reasoning")

	_, stopReason := askAgent(t, serveAgents(t, agent), &agentrun.Request{Agent: "acp:fake", Reasoning: "high", Prompt: "why?"})

	if !strings.Contains(stopReason, "reasoning rejected") {
		t.Errorf("stopped with %q", stopReason)
	}
}

func TestATaskStopsWithTheReasonTheAgentRejectsTheEndOfTheSession(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "fail-close")

	_, stopReason := askAgent(t, serveAgents(t, agent), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"})

	if !strings.Contains(stopReason, "close rejected") {
		t.Errorf("stopped with %q", stopReason)
	}
}

func TestAnAgentThatCannotCloseItsSessionIsListedWithNoModels(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "fail-close")
	svc := serveAgents(t, agent)

	programs := listPrograms(t, svc)

	if len(programs) != 1 || programs[0].Name != "acp:fake" || len(programs[0].Models) != 0 {
		t.Errorf("listed %v", programs)
	}
}

func TestReadingTheModelsOfAnAgentFailsWhenItRejectsTheClose(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "fail-close")
	svc := serveAgents(t, agent)

	_, err := svc.lister.ReadListing(t.Context(), svc.registry.List()[0])

	if err == nil || !strings.Contains(err.Error(), "close rejected") {
		t.Errorf("ReadListing with a rejected close = %v", err)
	}
}
