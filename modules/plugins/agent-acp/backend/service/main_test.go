package service

import (
	"context"
	"slices"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestLifecycleServer(t *testing.T) {
	ch := make(chan struct{})
	srv := server.NewLifecycleServer(ch)

	ctx := context.Background()
	pingResp, err := srv.Ping(ctx, connect.NewRequest(&numenv1.PingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if pingResp.Msg.GetStatus() != "ok" {
		t.Errorf("expected ok, got %s", pingResp.Msg.GetStatus())
	}

	shutResp, err := srv.Shutdown(ctx, connect.NewRequest(&numenv1.ShutdownRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !shutResp.Msg.GetSuccess() {
		t.Error("expected shutdown success")
	}
}

// The program list is empty, and the command lines of the settings name the agents completions are
// answered by.
func TestTheCommandLinesOfTheSettingsNameTheCompletionAgents(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "config")
	svc := &agentServer{}

	got, err := svc.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{
		Commands: []*numenv1.AgentProgramCommand{
			{Program: "acp:fake", Command: agent.Command},
			{Program: "other", Command: []string{"/bin/other"}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if programs := got.Msg.GetPrograms(); len(programs) != 0 {
		t.Errorf("the programs are %v", programs)
	}
	if models := listCompletionModels(t, svc); !slices.ContainsFunc(models, func(m string) bool { return strings.HasPrefix(m, "fake/m1=") }) {
		t.Errorf("the completion models are %v", models)
	}
}
