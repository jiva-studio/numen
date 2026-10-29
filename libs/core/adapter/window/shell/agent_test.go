package shell

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

// answering is an agent that returns one fixed response to each task and
// records the tasks and the ended conversations.
type answering struct {
	tasks chan agentport.Task
	ended chan string
}

func (a *answering) StartRun(_ context.Context, task agentport.Task) (agentport.Run, error) {
	a.tasks <- task
	steps := make(chan agentport.Step, 2)
	steps <- agentport.Step{Kind: domain.StepSay, Text: "an answer"}
	steps <- agentport.Step{Kind: domain.StepStop, Detail: "done"}
	close(steps)
	return answeringRun{steps: steps}, nil
}

func (a *answering) EndConversation(_ context.Context, conversation string) error {
	a.ended <- conversation
	return nil
}

type answeringRun struct{ steps chan agentport.Step }

func (r answeringRun) GetSteps() <-chan agentport.Step { return r.steps }
func (answeringRun) Stop() error                       { return nil }

// The window answers AgentService with its own agent even where a plugin serves the service, runs
// the task in the vault the question names, and ends the conversation there.
func TestTheWindowAnswersTheAgentWithItsOwn(t *testing.T) {
	agent := &answering{tasks: make(chan agentport.Task, 1), ended: make(chan string, 1)}
	api := newAPI(t)
	api.GetAgent = func() agentport.Agent { return agent }
	api.PluginServices = &forwarding{serves: map[string]bool{
		numenv1connect.AgentServiceAskAgentProcedure:           true,
		numenv1connect.AgentServiceFinishConversationProcedure: true,
		numenv1connect.AgentServiceGetAgentStateProcedure:      true,
	}}
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewAgentServiceClient(server.Client(), server.URL)

	state, err := client.GetAgentState(t.Context(), connect.NewRequest(&v1.GetAgentStateRequest{}))
	if err != nil || state.Msg.GetUnreachable() != "" {
		t.Fatalf("the state is %v, %v", state, err)
	}

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{
		Prompt: "Where does it live?", Focus: "samples/animals.md", Conversation: "c-1",
		Context: map[string]string{"org.example.sample/item": "k7"}, VaultId: "v-3",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var text, stopped string
	for stream.Receive() {
		text += stream.Msg().GetText()
		stopped += stream.Msg().GetStopped()
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if text != "an answer" || stopped != "done" {
		t.Errorf("the agent said %q and stopped with %q", text, stopped)
	}
	if task := <-agent.tasks; task.Question != "Where does it live?" || task.Focus != "samples/animals.md" ||
		task.Conversation != "c-1" || task.Agent != "" || task.Context["org.example.sample/item"] != "k7" ||
		task.Vault != "v-3" {
		t.Errorf("the agent was given %+v", task)
	}

	if _, err := client.FinishConversation(t.Context(),
		connect.NewRequest(&v1.FinishConversationRequest{Conversation: "c-1"})); err != nil {
		t.Fatal(err)
	}
	if ended := <-agent.ended; ended != "c-1" {
		t.Errorf("the agent ended %q", ended)
	}
}

// A window with no agent rejects a question as a precondition the installation
// does not meet.
func TestAskAgent_FailsPreconditionWithoutAgent(t *testing.T) {
	server := httptest.NewServer(newAPI(t).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewAgentServiceClient(server.Client(), server.URL)

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{Prompt: "q"}))
	if err != nil {
		t.Fatal(err)
	}
	for stream.Receive() {
	}
	if connect.CodeOf(stream.Err()) != connect.CodeFailedPrecondition {
		t.Errorf("the question failed with %v", stream.Err())
	}
}
