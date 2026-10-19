//go:build unix

package agentrun_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// shell makes a spec that runs the script and hands each event to handle.
func shell(script string, handle func(r *agentrun.Reader, event map[string]any)) agentrun.Spec {
	return agentrun.Spec{
		Build: func(ctx context.Context) (*exec.Cmd, func(), error) {
			return exec.CommandContext(ctx, "sh", "-c", script), nil, nil
		},
		NewHandler: func(r *agentrun.Reader) func(map[string]any) {
			return func(event map[string]any) { handle(r, event) }
		},
	}
}

func drain(work *agentrun.Work) []*numenv1.AskAgentResponse {
	var got []*numenv1.AskAgentResponse
	for resp := range work.Output {
		got = append(got, resp)
	}
	return got
}

func said(text string) *numenv1.AskAgentResponse {
	return &numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_Said{Said: text}}
}

func TestStartHandsJSONLinesToTheHandlerAndSaysTheOthers(t *testing.T) {
	spec := shell(`echo '{"say":"hello"}'; echo plain text; echo; echo 42`,
		func(r *agentrun.Reader, event map[string]any) { r.Send(said(event["say"].(string))) })
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	want := []string{"hello", "plain text\n", "42\n"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, text := range want {
		if got[i].GetSaid() != text {
			t.Errorf("response %d is %v, want %q", i, got[i], text)
		}
	}
}

func TestStartBuffersSixtyFourResponses(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell("exit 0", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = work.Stop() }()

	if got := cap(work.Output); got != 64 {
		t.Errorf("the output holds %d responses, want 64", got)
	}
}

func TestStartRunsTheChildInTheRootWithTheToken(t *testing.T) {
	root := t.TempDir()
	spec := shell(`echo "{\"dir\":\"$(pwd)\",\"token\":\"$NUMEN_TOKEN\"}"`,
		func(r *agentrun.Reader, event map[string]any) {
			r.Send(said(event["dir"].(string) + " " + event["token"].(string)))
		})
	spec.Root, spec.Token = root, "secret-1"
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	if len(got) != 1 || got[0].GetSaid() == "" {
		t.Fatalf("got %v", got)
	}
	if said := got[0].GetSaid(); !strings.HasSuffix(said, " secret-1") || !strings.Contains(said, "/") {
		t.Errorf("the child said %q", said)
	}
}

func TestStartStopsTheChildAfterTheStepLimit(t *testing.T) {
	spec := shell(`while true; do echo '{"finished":true}'; done`,
		func(r *agentrun.Reader, _ map[string]any) {
			r.Send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_ToolFinished{ToolFinished: &numenv1.ToolFinished{}}})
		})
	spec.MaxSteps = 2
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	if len(got) != 3 || got[2].GetStopped() != "reached maximum limit of 2 steps" {
		t.Errorf("got %v", got)
	}
}

func TestStartKeepsTheSessionOfTheConversation(t *testing.T) {
	spec := shell(`echo '{"session":"s-1"}'`,
		func(r *agentrun.Reader, event map[string]any) { r.SetSession(event["session"].(string)) })
	spec.Conversation = "c-1"
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	drain(work)

	if got := reg.GetSession("c-1"); got != "s-1" {
		t.Errorf("the session is %q, want s-1", got)
	}
}

func TestStartKeepsNoSessionWithoutAConversation(t *testing.T) {
	spec := shell(`echo '{"session":"s-1"}'`,
		func(r *agentrun.Reader, event map[string]any) { r.SetSession(event["session"].(string)) })
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	drain(work)

	if got := reg.GetSession(""); got != "" {
		t.Errorf("the session is %q, want none", got)
	}
}

func TestStartCleansUpWhenTheRunIsOver(t *testing.T) {
	cleaned := make(chan struct{})
	spec := shell("exit 0", nil)
	build := spec.Build
	spec.Build = func(ctx context.Context) (*exec.Cmd, func(), error) {
		cmd, _, _ := build(ctx)
		return cmd, func() { close(cleaned) }, nil
	}
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	drain(work)

	<-cleaned
}

func TestStartCleansUpAndFailsWhenTheProgramIsMissing(t *testing.T) {
	cleaned := false
	spec := agentrun.Spec{
		Build: func(ctx context.Context) (*exec.Cmd, func(), error) {
			return exec.CommandContext(ctx, "/nonexistent/program"), func() { cleaned = true }, nil
		},
	}
	var reg agentrun.Registry

	if _, err := agentrun.Start(t.Context(), &reg, spec); err == nil {
		t.Fatal("expected an error")
	}
	if !cleaned {
		t.Error("the run was not cleaned up")
	}
}

func TestStartRefusesWorkOnceTheRegistryIsClosed(t *testing.T) {
	var reg agentrun.Registry
	_ = reg.Close()

	if _, err := agentrun.Start(t.Context(), &reg, shell("exit 0", nil)); err == nil {
		t.Fatal("expected an error")
	}
}

type recordingStream struct {
	got []*numenv1.AskAgentResponse
	err error
}

func (s *recordingStream) Send(resp *numenv1.AskAgentResponse) error {
	s.got = append(s.got, resp)
	return s.err
}

func TestRelaySendsEveryResponseUntilTheWorkIsDone(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell(`echo one; echo two`, nil))
	if err != nil {
		t.Fatal(err)
	}
	stream := &recordingStream{}

	if err := agentrun.Relay(t.Context(), work, stream); err != nil {
		t.Fatal(err)
	}
	if len(stream.got) != 2 || stream.got[0].GetSaid() != "one\n" || stream.got[1].GetSaid() != "two\n" {
		t.Errorf("got %v", stream.got)
	}
}

func TestRelayStopsTheWorkWhenTheStreamFails(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell(`echo one; sleep 30`, nil))
	if err != nil {
		t.Fatal(err)
	}
	stream := &recordingStream{err: context.Canceled}

	if err := agentrun.Relay(t.Context(), work, stream); err != context.Canceled {
		t.Errorf("got %v", err)
	}
}

func TestRelayStopsTheWorkWhenTheCallEnds(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell(`sleep 30`, nil))
	if err != nil {
		t.Fatal(err)
	}
	ended, end := context.WithCancel(t.Context())
	end()

	if err := agentrun.Relay(ended, work, &recordingStream{}); err != context.Canceled {
		t.Errorf("got %v", err)
	}
}

func TestStartKillsTheChildThatIgnoresTheClosedOutputAtTheStepLimit(t *testing.T) {
	spec := shell(`trap '' PIPE; echo '{"finished":true}'; sleep 30`,
		func(r *agentrun.Reader, _ map[string]any) {
			r.Send(&numenv1.AskAgentResponse{Step: &numenv1.AskAgentResponse_ToolFinished{ToolFinished: &numenv1.ToolFinished{}}})
		})
	spec.MaxSteps = 1
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	if len(got) != 2 || got[1].GetStopped() == "" {
		t.Errorf("got %v", got)
	}
}

func TestStartReportsTheErrorOfTheBuild(t *testing.T) {
	spec := agentrun.Spec{
		Build: func(context.Context) (*exec.Cmd, func(), error) { return nil, nil, errors.New("no home") },
	}
	var reg agentrun.Registry

	if _, err := agentrun.Start(t.Context(), &reg, spec); err == nil || err.Error() != "no home" {
		t.Errorf("got %v", err)
	}
}

func TestStartReportsAFailedRunAsTheReasonItStopped(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell(`echo one; echo "not signed in" >&2; exit 3`, nil))
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	if len(got) != 2 || got[1].GetStopped() != "not signed in" {
		t.Errorf("got %v", got)
	}
}

func TestStartReportsALineTooLongAsTheReasonItStopped(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell(`head -c 5000000 /dev/zero | tr '\0' x; echo`, nil))
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	if len(got) != 1 || got[0].GetStopped() == "" {
		t.Errorf("got %v", got)
	}
}

func TestStartReportsNothingForARunThatEndedWell(t *testing.T) {
	var reg agentrun.Registry
	work, err := agentrun.Start(t.Context(), &reg, shell(`echo one`, nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := drain(work); len(got) != 1 {
		t.Errorf("got %v", got)
	}
}

func TestStartStopsAtTheStepLimitOfCountedSteps(t *testing.T) {
	spec := shell(`while true; do echo '{}'; done`,
		func(r *agentrun.Reader, _ map[string]any) { r.CountStep() })
	spec.MaxSteps = 3
	var reg agentrun.Registry

	work, err := agentrun.Start(t.Context(), &reg, spec)
	if err != nil {
		t.Fatal(err)
	}
	got := drain(work)

	if len(got) != 1 || got[0].GetStopped() != "reached maximum limit of 3 steps" {
		t.Errorf("got %v", got)
	}
}
