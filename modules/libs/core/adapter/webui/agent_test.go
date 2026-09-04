package webui_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// asking is an agent that keeps the task it was given and takes the steps a
// test told it to take. What is asked of it is the hop: what a client sends
// arrives as a task, and what the agent does arrives back as steps.
type asking struct {
	took  chan port.Task
	takes []port.Step
	// over is the conversations this agent was told are finished.
	over chan string
	// refuses is what finishing a conversation answers with.
	refuses error
}

func (a *asking) Take(_ context.Context, task port.Task) (port.Work, error) {
	a.took <- task
	steps := make(chan port.Step, len(a.takes))
	for _, step := range a.takes {
		steps <- step
	}
	close(steps)
	return answering{steps}, nil
}

func (a *asking) Finish(_ context.Context, conversation string) error {
	a.over <- conversation
	return a.refuses
}

// answering is work that has already been done.
type answering struct{ steps chan port.Step }

func (a answering) Steps() <-chan port.Step { return a.steps }
func (a answering) Stop() error             { return nil }

// panelling is a vault whose panel one agent answers.
func panelling(taking port.Agent) *webui.API {
	api := &webui.API{}
	api.Answers(taking)
	return api
}

// panelled is one vault's panel and a client talking to it the way the window
// does.
func panelled(t *testing.T, api *webui.API) numenv1connect.AgentServiceClient {
	t.Helper()

	route, handler := numenv1connect.NewAgentServiceHandler(api)
	mux := http.NewServeMux()
	mux.Handle(route, handler)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	return numenv1connect.NewAgentServiceClient(server.Client(), server.URL)
}

// heard is every step a client is sent, in order.
func heard(t *testing.T, client numenv1connect.AgentServiceClient, ask *v1.AskAgentRequest) []*v1.AskAgentResponse {
	t.Helper()

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(ask))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	var steps []*v1.AskAgentResponse
	for stream.Receive() {
		steps = append(steps, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return steps
}

// TestWhatTheClientAsksReachesTheAgent. Three things are sent and three arrive:
// the question, the note in front of the person, and which conversation they
// asked it in.
func TestWhatTheClientAsksReachesTheAgent(t *testing.T) {
	taking := &asking{took: make(chan port.Task, 1), takes: []port.Step{{Kind: port.StepStopped}}}
	client := panelled(t, panelling(taking))

	heard(t, client, &v1.AskAgentRequest{
		Asked:        "rewrite this note",
		Focus:        "notes/Fugue.md",
		Conversation: "8f2c1e",
	})

	task := <-taking.took
	if task.Question != "rewrite this note" {
		t.Errorf("the agent was asked %q, want %q", task.Question, "rewrite this note")
	}
	if task.Focus != "notes/Fugue.md" {
		t.Errorf("the note in focus is %q, want %q", task.Focus, "notes/Fugue.md")
	}
	if task.Conversation != "8f2c1e" {
		t.Errorf("the conversation is %q, want %q", task.Conversation, "8f2c1e")
	}
}

// TestEveryStepTheAgentTakesReachesTheClient, in the words the schema carries
// and in the order they were taken.
func TestEveryStepTheAgentTakesReachesTheClient(t *testing.T) {
	taking := &asking{took: make(chan port.Task, 1), takes: []port.Step{
		{Kind: port.StepThinking},
		{Kind: port.StepToolCall, Tool: "note_rewrite", About: "notes/Fugue.md", Count: 240},
		{Kind: port.StepAnswered},
		{Kind: port.StepSaying, Text: "Rewritten."},
		{Kind: port.StepStopped, Detail: "out of turns"},
	}}
	client := panelled(t, panelling(taking))

	steps := heard(t, client, &v1.AskAgentRequest{Asked: "rewrite this note"})
	if len(steps) != 5 {
		t.Fatalf("got %d steps, want 5: %+v", len(steps), steps)
	}
	if steps[0].GetThinking() == nil {
		t.Errorf("the first step is %+v, want a wait beginning", steps[0])
	}
	doing := steps[1].GetToolCall()
	if doing.GetTool() != "note_rewrite" || doing.GetAbout() != "notes/Fugue.md" || doing.GetWritten() != 240 {
		t.Errorf("the tool in hand is %+v, want note_rewrite over notes/Fugue.md at 240", doing)
	}
	if steps[2].GetAnswered() == nil {
		t.Errorf("the third step is %+v, want the tool answering", steps[2])
	}
	if said := steps[3].GetSaid(); said != "Rewritten." {
		t.Errorf("the agent said %q, want %q", said, "Rewritten.")
	}
	if why := steps[4].GetStopped(); why != "out of turns" {
		t.Errorf("it stopped saying %q, want %q", why, "out of turns")
	}
}

// A call says where in the vault it was working, so that an answer can name the
// place it stands on. A call the vault serves is drawn as a tool in hand
// whatever that tool does to the vault.
func TestACallSaysWhereItIsWorking(t *testing.T) {
	taking := &asking{took: make(chan port.Task, 1), takes: []port.Step{
		{
			Kind: port.StepRead, Tool: "Show the person a passage of a document",
			About: "library/A Book.epub",
			Place: domain.Place{Path: "library/A Book.epub", Start: 1200, Length: 80},
		},
		{Kind: port.StepStopped},
	}}
	client := panelled(t, panelling(taking))

	steps := heard(t, client, &v1.AskAgentRequest{Asked: "show me where that is"})
	doing := steps[0].GetToolCall()
	if doing.GetPath() != "library/A Book.epub" || doing.GetStart() != 1200 || doing.GetLength() != 80 {
		t.Errorf("the call is working at %+v", doing)
	}
}

// TestAConversationSaidToBeOverReachesTheAgent, under the name its questions
// carried. A tab closes, and what was kept for it is let go of.
func TestAConversationSaidToBeOverReachesTheAgent(t *testing.T) {
	taking := &asking{over: make(chan string, 1)}
	client := panelled(t, panelling(taking))

	if _, err := client.FinishConversation(t.Context(),
		connect.NewRequest(&v1.FinishConversationRequest{Conversation: "8f2c1e"})); err != nil {
		t.Fatal(err)
	}

	if over := <-taking.over; over != "8f2c1e" {
		t.Errorf("the conversation said to be over is %q, want %q", over, "8f2c1e")
	}
}

// TestAConversationThatCouldNotBeFinishedSaysSo. The client hears it and the
// agent is the one that could not let go.
func TestAConversationThatCouldNotBeFinishedSaysSo(t *testing.T) {
	taking := &asking{over: make(chan string, 1), refuses: errors.New("the child would not go")}
	client := panelled(t, panelling(taking))

	_, err := client.FinishConversation(t.Context(),
		connect.NewRequest(&v1.FinishConversationRequest{Conversation: "8f2c1e"}))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("a conversation that could not be finished answered %v, want %v",
			got, connect.CodeInternal)
	}
}

// TestAVaultWithNoAgentHasNoConversationToFinish. Nothing is kept for one, and
// the panel is told there is nobody to ask.
func TestAVaultWithNoAgentHasNoConversationToFinish(t *testing.T) {
	client := panelled(t, &webui.API{})

	_, err := client.FinishConversation(t.Context(),
		connect.NewRequest(&v1.FinishConversationRequest{Conversation: "8f2c1e"}))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("a vault with no agent answered %v, want %v", got, connect.CodeFailedPrecondition)
	}
}

// TestAVaultWithNoAgentSaysSo. The rest of the window works as it did, and the
// panel is told there is nobody to ask.
func TestAVaultWithNoAgentSaysSo(t *testing.T) {
	client := panelled(t, &webui.API{})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{Asked: "anyone?"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	for stream.Receive() {
	}
	if got := connect.CodeOf(stream.Err()); got != connect.CodeFailedPrecondition {
		t.Errorf("a vault with no agent answered %v, want %v", got, connect.CodeFailedPrecondition)
	}
}
