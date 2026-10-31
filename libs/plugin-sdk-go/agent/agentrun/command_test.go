package agentrun_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

func TestExecutablePathKeepsTheTrimmedValueOfTheConfiguration(t *testing.T) {
	var path agentrun.ExecutablePath

	if got := path.GetExecutablePath(); got != "" {
		t.Errorf("zero value = %q, want empty", got)
	}
	path.ApplyConfig(map[string]string{"executable_path": "  /usr/bin/tool \n"})
	if got := path.GetExecutablePath(); got != "/usr/bin/tool" {
		t.Errorf("path = %q, want /usr/bin/tool", got)
	}
	path.ApplyConfig(map[string]string{"other": "x"})
	if got := path.GetExecutablePath(); got != "/usr/bin/tool" {
		t.Errorf("path after an unrelated configuration = %q, want it kept", got)
	}
	path.ApplyConfig(map[string]string{"executable_path": ""})
	if got := path.GetExecutablePath(); got != "" {
		t.Errorf("path after an empty value = %q, want empty", got)
	}
}

func TestNewInstalledStateIsReachableOnlyWhereInstalled(t *testing.T) {
	if got := agentrun.NewInstalledState(true, "missing").Msg.GetUnreachable(); got != "" {
		t.Errorf("installed = %q, want reachable", got)
	}
	if got := agentrun.NewInstalledState(false, "missing").Msg.GetUnreachable(); got != "missing" {
		t.Errorf("not installed = %q, want the message", got)
	}
}

type streamHandler struct {
	agentrun.Service
	start func() (*agentrun.Work, error)
}

func (h *streamHandler) AskAgent(
	ctx context.Context,
	_ *connect.Request[agentrun.AskAgentRequest],
	stream *connect.ServerStream[agentrun.AskAgentResponse],
) error {
	return agentrun.StreamWork(ctx, stream, h.start)
}

func receiveAll(t *testing.T, handler agentrun.AgentServiceHandler) []*agentrun.AskAgentResponse {
	t.Helper()
	client := agenttest.Serve(t, handler)
	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&agentrun.AskAgentRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var got []*agentrun.AskAgentResponse
	for stream.Receive() {
		got = append(got, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestStreamWorkReportsAFailedStartAsAStop(t *testing.T) {
	got := receiveAll(t, &streamHandler{start: func() (*agentrun.Work, error) {
		return nil, errors.New("cannot start")
	}})

	if len(got) != 2 || got[0].GetModelStarted() == nil || got[1].GetStopped() != "cannot start" {
		t.Fatalf("responses = %v", got)
	}
}

func TestStreamWorkForwardsTheResponsesOfTheWork(t *testing.T) {
	got := receiveAll(t, &streamHandler{start: func() (*agentrun.Work, error) {
		work := agentrun.NewWork("", 1, func() error { return nil })
		work.Output <- &agentrun.AskAgentResponse{Step: &agentrun.AskAgentResponseText{Text: "hi"}}
		close(work.Output)
		return work, nil
	}})

	if len(got) != 2 || got[0].GetModelStarted() == nil || got[1].GetText() != "hi" {
		t.Fatalf("responses = %v", got)
	}
}
