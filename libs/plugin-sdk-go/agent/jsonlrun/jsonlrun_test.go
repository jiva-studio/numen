package jsonlrun

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

func echoHandler(r *Reader) func(map[string]any) {
	return func(event map[string]any) {
		switch event["kind"] {
		case "text":
			text, ok := event["text"].(string)
			if !ok {
				return
			}
			r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseText{Text: text}})
		case "tool":
			r.Send(&agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseToolFinished{ToolFinished: &agentrun.ToolFinished{}}})
		case "end":
			r.Finish()
		}
	}
}

func specFor(script string, release func() error) Spec {
	return Spec{
		Conversation: "", MaxSteps: 0, Root: "", Token: "",
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, script), release, nil
		},
		NewHandler: echoHandler,
	}
}

func drain(t *testing.T, w *agentrun.Work) []*agentrun.AskAgentResponse {
	t.Helper()
	var got []*agentrun.AskAgentResponse
	for resp := range w.Output {
		got = append(got, resp)
	}
	if err := w.Stop(); err != nil {
		t.Error(err)
	}
	return got
}

func TestStartReportsEventsAndPlainLines(t *testing.T) {
	script := agenttest.WriteScript(t, "prog", `echo '{"kind":"text","text":"hello"}'`+"\necho plain\n")
	var reg agentrun.Registry

	w, err := Start(t.Context(), &reg, specFor(script, nil))
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, w)

	if len(got) != 2 || got[0].GetText() != "hello" || got[1].GetText() != "plain\n" {
		t.Fatalf("responses = %v", got)
	}
}

func TestFinishEndsTheRunAtTheEventThatCalledIt(t *testing.T) {
	script := agenttest.WriteScript(t, "prog",
		`echo '{"kind":"text","text":"a"}'`+"\n"+`echo '{"kind":"end"}'`+"\n"+`echo '{"kind":"text","text":"after"}'`+"\n")
	var reg agentrun.Registry

	w, err := Start(t.Context(), &reg, specFor(script, nil))
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, w)

	if len(got) != 1 || got[0].GetText() != "a" {
		t.Fatalf("responses = %v, want the text before the end alone", got)
	}
}

func TestTheStepLimitStopsTheRun(t *testing.T) {
	script := agenttest.WriteScript(t, "prog",
		`echo '{"kind":"tool"}'`+"\n"+`echo '{"kind":"tool"}'`+"\n"+`echo '{"kind":"tool"}'`+"\n")
	var reg agentrun.Registry
	spec := specFor(script, nil)
	spec.MaxSteps = 2

	w, err := Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, w)

	last := got[len(got)-1]
	if !strings.Contains(last.GetStopped(), "maximum limit of 2 steps") {
		t.Errorf("last response = %v, want the step limit", last)
	}
}

func TestAFailedExitIsReportedWithItsStderr(t *testing.T) {
	script := agenttest.WriteScript(t, "prog", "echo broken >&2\nexit 3\n")
	var reg agentrun.Registry

	w, err := Start(t.Context(), &reg, specFor(script, nil))
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, w)

	if len(got) != 1 || !strings.Contains(got[0].GetStopped(), "broken") {
		t.Fatalf("responses = %v, want a stop that names stderr", got)
	}
}

func TestTheReleaseRunsOnceAndItsErrorIsReported(t *testing.T) {
	script := agenttest.WriteScript(t, "prog", `echo '{"kind":"text","text":"x"}'`+"\n")
	var reg agentrun.Registry
	calls := 0

	w, err := Start(t.Context(), &reg, specFor(script, func() error {
		calls++
		return errors.New("folder is busy")
	}))
	if err != nil {
		t.Fatal(err)
	}
	got := drain(t, w)

	if calls != 1 {
		t.Errorf("release ran %d times, want 1", calls)
	}
	last := got[len(got)-1]
	if !strings.Contains(last.GetStopped(), "folder is busy") {
		t.Errorf("last response = %v, want the release error", last)
	}
}

func TestAFailedStartReturnsTheReleaseError(t *testing.T) {
	var reg agentrun.Registry
	spec := Spec{
		Conversation: "", MaxSteps: 0, Root: "", Token: "",
		Build: func(ctx context.Context) (*exec.Cmd, func() error, error) {
			return exec.CommandContext(ctx, "/nonexistent/program"), func() error { return errors.New("release failed") }, nil
		},
		NewHandler: echoHandler,
	}

	_, err := Start(t.Context(), &reg, spec)

	if err == nil || !strings.Contains(err.Error(), "release failed") || !strings.Contains(err.Error(), "start") {
		t.Fatalf("err = %v, want the start failure joined with the release failure", err)
	}
}

func TestABuildFailureIsReturned(t *testing.T) {
	var reg agentrun.Registry
	failure := errors.New("no command")
	spec := Spec{
		Conversation: "", MaxSteps: 0, Root: "", Token: "",
		Build: func(context.Context) (*exec.Cmd, func() error, error) {
			return nil, nil, failure
		},
		NewHandler: echoHandler,
	}

	if _, err := Start(t.Context(), &reg, spec); !errors.Is(err, failure) {
		t.Fatalf("err = %v, want %v", err, failure)
	}
}
