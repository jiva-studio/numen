package wire

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

type closingAgent struct{}

func (closingAgent) StartRun(_ context.Context, _ agentport.Task) (agentport.Run, error) {
	steps := make(chan agentport.Step, 1)
	steps <- agentport.Step{Kind: domain.StepSay, Text: "Hey! How can I help you today?"}
	close(steps)
	return closingRun{steps: steps}, nil
}

func (closingAgent) EndConversation(_ context.Context, _ string) error { return nil }

type closingRun struct{ steps <-chan agentport.Step }

func (r closingRun) GetSteps() <-chan agentport.Step { return r.steps }
func (closingRun) Stop() error                       { return nil }

// A run that closes its steps channel without an explicit stop still writes a stopped frame and a
// trailing frame to flush it to the window.
func TestStreamAgentTaskEndsWithStoppedAndTrailingFrame(t *testing.T) {
	route, handler := numenv1connect.NewAgentServiceHandler(&AgentHandler{
		GetAgent: func() agentport.Agent { return closingAgent{} },
	})
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := numenv1connect.NewAgentServiceClient(server.Client(), server.URL)
	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{Prompt: "hey"}))
	if err != nil {
		t.Fatal(err)
	}
	var frames []*v1.AskAgentResponse
	for stream.Receive() {
		frames = append(frames, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 {
		t.Fatalf("got %d frames, want 3: %v", len(frames), frames)
	}
	if frames[0].GetText() != "Hey! How can I help you today?" {
		t.Errorf("frame 0 text = %q", frames[0].GetText())
	}
	if frames[1].GetStopped() != "done" {
		t.Errorf("frame 1 stopped = %q, want done", frames[1].GetStopped())
	}
	if frames[2].GetStep() != nil {
		t.Errorf("frame 2 step = %v, want nil trailing flush frame", frames[2].GetStep())
	}
}
