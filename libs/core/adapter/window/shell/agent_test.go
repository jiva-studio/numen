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

// answering is an agent that says one thing to each task and keeps the tasks and the conversations
// ended.
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

// The window answers AgentService with its own agent even where a plugin serves the service, and
// ends the conversation there.
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
		Prompt: "Where does it live?", Focus: "decks/animals.md", Conversation: "c-1",
		Context: map[string]string{"md.example.cards/card": "k7"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for stream.Receive() {
		text += stream.Msg().GetText()
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if text != "an answer" {
		t.Errorf("the agent said %q", text)
	}
	if task := <-agent.tasks; task.Question != "Where does it live?" || task.Focus != "decks/animals.md" ||
		task.Conversation != "c-1" || task.Agent != "" || task.Context["md.example.cards/card"] != "k7" {
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

// A window with no agent refuses a question as a precondition the installation does not meet.
func TestAWindowWithNoAgentRefusesAQuestion(t *testing.T) {
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
