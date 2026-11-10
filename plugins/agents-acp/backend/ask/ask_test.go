package ask_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/agents"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/ask"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/internal/adapter/process"
	"github.com/jiva-studio/numen/plugins/agents-acp/backend/testsupport/fakeacp"
)

var errSend = errors.New("send failed")

func TestMain(m *testing.M) { fakeacp.Main(m) }

func createAsker(t *testing.T, behavior string) *ask.Asker {
	t.Helper()
	command, _ := fakeacp.Start(t, behavior)
	registry := agents.NewRegistry(nil)
	registry.SetCommands(map[string][]string{"acp:fake": command})
	asker, err := ask.New(registry, process.Launcher{})
	if err != nil {
		t.Fatal(err)
	}
	return asker
}

func TestAskerNeedsAFinderAndALauncher(t *testing.T) {
	if _, err := ask.New(nil, process.Launcher{}); err == nil {
		t.Error("no finder was refused")
	}
	if _, err := ask.New(agents.NewRegistry(nil), nil); err == nil {
		t.Error("no launcher was refused")
	}
}

func TestATaskStopsWhenTheFirstStepCannotBeSent(t *testing.T) {
	asker := createAsker(t, "plain")
	calls := 0

	err := asker.Execute(t.Context(), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(*agentrun.Response) error {
		calls++
		return errSend
	})

	if !errors.Is(err, errSend) || calls != 1 {
		t.Errorf("Execute on initial send error = %v (calls %d)", err, calls)
	}
}

func TestATaskForAMissingAgentReturnsTheErrorOfTheStoppedStep(t *testing.T) {
	asker := createAsker(t, "plain")
	calls := 0

	err := asker.Execute(t.Context(), &agentrun.Request{Agent: "acp:absent", Prompt: "why?"}, func(res *agentrun.Response) error {
		calls++
		if res.GetStopped() != "" {
			return errSend
		}
		return nil
	})

	if !errors.Is(err, errSend) || calls != 2 {
		t.Errorf("Execute on stopped send error = %v (calls %d)", err, calls)
	}
}

func TestAnAnswerThatCannotBeSentIsNotFollowedByAStoppedStep(t *testing.T) {
	asker := createAsker(t, "plain")
	calls := 0

	err := asker.Execute(t.Context(), &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(res *agentrun.Response) error {
		calls++
		if res.GetText() != "" {
			return errSend
		}
		return nil
	})

	if !errors.Is(err, errSend) || calls != 2 {
		t.Errorf("Execute on text send error = %v (calls %d)", err, calls)
	}
}

func TestACanceledTaskReturnsTheCancellationAndSendsNoReason(t *testing.T) {
	asker := createAsker(t, "plain")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var stopReason string

	err := asker.Execute(ctx, &agentrun.Request{Agent: "acp:fake", Prompt: "why?"}, func(res *agentrun.Response) error {
		if s := res.GetStopped(); s != "" {
			stopReason = s
		}
		return nil
	})

	if !errors.Is(err, context.Canceled) || stopReason != "" {
		t.Errorf("Execute on canceled ctx = %v, stopReason = %q", err, stopReason)
	}
}

func TestATaskStopsWithTheReasonTheAgentRejectsItsDepth(t *testing.T) {
	asker := createAsker(t, "fail-reasoning")

	stopReason := execute(t, asker, &agentrun.Request{Agent: "acp:fake", Reasoning: "high", Prompt: "why?"})

	if !strings.Contains(stopReason, "reasoning rejected") {
		t.Errorf("stopped with %q", stopReason)
	}
}

func TestATaskStopsWithTheReasonTheAgentRejectsTheEndOfTheSession(t *testing.T) {
	asker := createAsker(t, "fail-close")

	stopReason := execute(t, asker, &agentrun.Request{Agent: "acp:fake", Prompt: "why?"})

	if !strings.Contains(stopReason, "close rejected") {
		t.Errorf("stopped with %q", stopReason)
	}
}

func execute(t *testing.T, asker *ask.Asker, task *agentrun.Request) (stopReason string) {
	t.Helper()
	err := asker.Execute(t.Context(), task, func(res *agentrun.Response) error {
		if s := res.GetStopped(); s != "" {
			stopReason = s
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return stopReason
}
