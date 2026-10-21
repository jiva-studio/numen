package editor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// askedAgent is an agent that keeps the task it was given and takes the steps a
// test told it to take. What is asked of it is the hop: what a client sends
// arrives as a task, and what the agent does arrives back as steps.
type askedAgent struct {
	took  chan port.Task
	takes []port.Step
	// over is the conversations this agent was told are finished.
	over chan string
	// refuses is what finishing a conversation answers with.
	refuses error
}

func (a *askedAgent) StartRun(_ context.Context, task port.Task) (port.Run, error) {
	a.took <- task
	steps := make(chan port.Step, len(a.takes))
	for _, step := range a.takes {
		steps <- step
	}
	close(steps)
	return answering{steps}, nil
}

func (a *askedAgent) EndConversation(_ context.Context, conversation string) error {
	a.over <- conversation
	return a.refuses
}

// answering is work that has already been done.
type answering struct{ steps chan port.Step }

func (a answering) GetSteps() <-chan port.Step { return a.steps }
func (a answering) Stop() error                { return nil }

// newAgentAPI is a vault whose panel one agent answers.
func newAgentAPI(taking port.Agent) *agentHandler {
	state := &State{}
	state.SetAgent(taking)
	return &agentHandler{state: state}
}

// newAgentClient is one vault's panel and a client talking to it the way the window
// does.
func newAgentClient(t *testing.T, api *agentHandler) numenv1connect.AgentServiceClient {
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

// getSteps is every step a client is sent, in order.
func getSteps(t *testing.T, client numenv1connect.AgentServiceClient, ask *v1.AskAgentRequest) []*v1.AskAgentResponse {
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
	taking := &askedAgent{took: make(chan port.Task, 1), takes: []port.Step{{Kind: domain.StepStop}}}
	client := newAgentClient(t, newAgentAPI(taking))

	getSteps(t, client, &v1.AskAgentRequest{
		Prompt:       "rewrite this note",
		Focus:        "notes/Fugue.md",
		Conversation: "8f2c1e",
		Reasoning:    "high",
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
	if task.Reasoning != "high" {
		t.Errorf("the depth of reasoning is %q, want %q", task.Reasoning, "high")
	}
}

// TestEveryStepTheAgentTakesReachesTheClient, in the words the schema carries
// and in the order they were taken.
func TestEveryStepTheAgentTakesReachesTheClient(t *testing.T) {
	taking := &askedAgent{took: make(chan port.Task, 1), takes: []port.Step{
		{Kind: domain.StepModelStart},
		{Kind: domain.StepToolCall, Tool: "note_rewrite", About: "notes/Fugue.md", Count: 240},
		{Kind: domain.StepToolDone},
		{Kind: domain.StepSay, Text: "Rewritten."},
		{Kind: domain.StepStop, Detail: "out of turns"},
	}}
	client := newAgentClient(t, newAgentAPI(taking))

	steps := getSteps(t, client, &v1.AskAgentRequest{Prompt: "rewrite this note"})
	if len(steps) != 5 {
		t.Fatalf("got %d steps, want 5: %+v", len(steps), steps)
	}
	if steps[0].GetModelStarted() == nil {
		t.Errorf("the first step is %+v, want a wait beginning", steps[0])
	}
	doing := steps[1].GetToolCall()
	if doing.GetTool() != "note_rewrite" || doing.GetAbout() != "notes/Fugue.md" || doing.GetLength() != 240 {
		t.Errorf("the tool in hand is %+v, want note_rewrite over notes/Fugue.md at 240", doing)
	}
	if steps[2].GetToolFinished() == nil {
		t.Errorf("the third step is %+v, want the tool answering", steps[2])
	}
	if said := steps[3].GetText(); said != "Rewritten." {
		t.Errorf("the agent said %q, want %q", said, "Rewritten.")
	}
	if why := steps[4].GetStopped(); why != "out of turns" {
		t.Errorf("it stopped saying %q, want %q", why, "out of turns")
	}
}

// A call says where in the vault it was working, so that an answer can name the
// place it relies on. A call the vault serves is drawn as a tool in hand
// whatever that tool does to the vault.
func TestACallSaysWhereItIsWorking(t *testing.T) {
	taking := &askedAgent{took: make(chan port.Task, 1), takes: []port.Step{
		{
			Kind: domain.StepRead, Tool: "Show the person a passage of a document",
			About: "library/A Book.epub",
			Place: domain.Place{Path: "library/A Book.epub", Spans: []domain.ByteSpan{{From: 1200, To: 1280}}},
		},
		{Kind: domain.StepStop},
	}}
	client := newAgentClient(t, newAgentAPI(taking))

	steps := getSteps(t, client, &v1.AskAgentRequest{Prompt: "show me where that is"})
	doing := steps[0].GetToolCall()
	if doing.GetPath() != "library/A Book.epub" ||
		doing.GetSpan().GetFrom() != 1200 || doing.GetSpan().GetTo() != 1280 {
		t.Errorf("the call is working at %+v", doing)
	}
}

// TestAConversationSaidToBeOverReachesTheAgent, under the name its questions
// carried. A tab closes, and what was kept for it is release.
func TestAConversationSaidToBeOverReachesTheAgent(t *testing.T) {
	taking := &askedAgent{over: make(chan string, 1)}
	client := newAgentClient(t, newAgentAPI(taking))

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
	taking := &askedAgent{over: make(chan string, 1), refuses: errors.New("the child would not go")}
	client := newAgentClient(t, newAgentAPI(taking))

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
	client := newAgentClient(t, &agentHandler{state: &State{}})

	_, err := client.FinishConversation(t.Context(),
		connect.NewRequest(&v1.FinishConversationRequest{Conversation: "8f2c1e"}))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("a vault with no agent answered %v, want %v", got, connect.CodeFailedPrecondition)
	}
}

// TestAVaultWithNoAgentSaysSo. The rest of the window works as it did, and the
// panel is told there is nobody to ask.
func TestAVaultWithNoAgentSaysSo(t *testing.T) {
	client := newAgentClient(t, &agentHandler{state: &State{}})

	stream, err := client.AskAgent(t.Context(), connect.NewRequest(&v1.AskAgentRequest{Prompt: "anyone?"}))
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
