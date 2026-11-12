package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	agentuse "github.com/jiva-studio/numen/libs/core/kernel/usecase/agent"
)

// programAgent is the agent program: it answers every task with one line and records the tasks and
// the ended conversations.
type programAgent struct {
	mu      sync.Mutex
	tasks   []agentdomain.Task
	ended   []string
	stopped int
}

func (a *programAgent) StartRun(_ context.Context, task agentdomain.Task) (agentport.Run, error) {
	a.mu.Lock()
	a.tasks = append(a.tasks, task)
	a.mu.Unlock()
	steps := make(chan agentdomain.Step, 1)
	steps <- agentdomain.Step{Kind: domain.StepSay, Text: "Hey! How can I help you today?"}
	close(steps)
	return programRun{agent: a, steps: steps}, nil
}

func (a *programAgent) EndConversation(_ context.Context, conversation string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ended = append(a.ended, conversation)
	return nil
}

type programRun struct {
	agent *programAgent
	steps <-chan agentdomain.Step
}

func (r programRun) GetSteps() <-chan agentdomain.Step { return r.steps }

func (r programRun) Stop() error {
	r.agent.mu.Lock()
	defer r.agent.mu.Unlock()
	r.agent.stopped++
	return nil
}

func newHandler(program *programAgent) *Handler {
	return &Handler{Start: agentuse.NewStartTask(program), End: agentuse.NewEndConversation(program)}
}

func newClient(t *testing.T, handler *Handler) numenv1connect.AgentServiceClient {
	t.Helper()
	route, served := numenv1connect.NewAgentServiceHandler(handler)
	mux := http.NewServeMux()
	mux.Handle(route, served)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewAgentServiceClient(server.Client(), server.URL)
}

func readFrames(t *testing.T, client numenv1connect.AgentServiceClient, request *v1.AskAgentRequest) []*v1.AskAgentResponse {
	t.Helper()
	stream, err := client.AskAgent(t.Context(), connect.NewRequest(request))
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
	return frames
}

// A run that closes its steps channel without an explicit stop still writes a stopped frame and a
// trailing frame to flush it to the window.
func TestAskAgentEndsWithStoppedAndTrailingFrame(t *testing.T) {
	program := &programAgent{}
	client := newClient(t, newHandler(program))

	frames := readFrames(t, client, &v1.AskAgentRequest{Prompt: "hey"})

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
	if program.stopped != 1 {
		t.Errorf("the run was stopped %d times, want once", program.stopped)
	}
}

func TestATaskThatNamesNoFolderOrVaultIsStartedInTheWindowsVault(t *testing.T) {
	program := &programAgent{}
	handler := newHandler(program)
	handler.GetRoot = func() string { return "/vaults/Garden" }
	handler.GetVault = func() domain.VaultID { return "v-garden" }
	client := newClient(t, handler)

	readFrames(t, client, &v1.AskAgentRequest{Prompt: "test question", Agent: "claude"})
	readFrames(t, client, &v1.AskAgentRequest{Prompt: "elsewhere", Root: "/vaults/Other", VaultId: "v-other"})

	if len(program.tasks) != 2 {
		t.Fatalf("the agent received %d tasks, want 2", len(program.tasks))
	}
	first, second := program.tasks[0], program.tasks[1]
	if first.Agent != "claude" || first.Question != "test question" || first.Vault != "v-garden" || first.Root != "/vaults/Garden" {
		t.Errorf("unexpected task: %+v", first)
	}
	if second.Vault != "v-other" || second.Root != "/vaults/Other" {
		t.Errorf("a task that names a folder and a vault was started in %q of %q", second.Root, second.Vault)
	}
}

func TestAskAgentWithNoAgentIsAFailedPrecondition(t *testing.T) {
	client := newClient(t, &Handler{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{Prompt: "hey"}))
	if err == nil {
		for stream.Receive() {
		}
		err = stream.Err()
	}

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("got %v", err)
	}
}

func TestFinishConversationEndsItOnTheAgent(t *testing.T) {
	program := &programAgent{}
	handler := newHandler(program)

	_, err := handler.FinishConversation(t.Context(), connect.NewRequest(&v1.FinishConversationRequest{Conversation: "c-1"}))
	_, none := handler.FinishConversation(t.Context(), connect.NewRequest(&v1.FinishConversationRequest{}))

	if err != nil || none != nil {
		t.Fatalf("answered %v and %v", err, none)
	}
	if len(program.ended) != 1 || program.ended[0] != "c-1" {
		t.Errorf("the agent ended %v, want only c-1", program.ended)
	}
}

func TestAgentStateReportsWhyNoAgentIsReachable(t *testing.T) {
	handler := &Handler{GetUnreachable: func() string { return "no program is installed" }}

	state, err := handler.GetAgentState(t.Context(), connect.NewRequest(&v1.GetAgentStateRequest{}))

	if err != nil || state.Msg.GetUnreachable() != "no program is installed" {
		t.Fatalf("answered %v (%v)", state, err)
	}
}
