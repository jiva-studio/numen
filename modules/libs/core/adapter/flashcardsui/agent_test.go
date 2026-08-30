package flashcardsui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// asking keeps the task it was given and takes the steps a test told it to
// take. What is asked of it is the hop: what the page sends arrives as a task,
// and what the agent does arrives back as steps.
type asking struct {
	took  chan port.Task
	takes []port.Step
	over  chan string
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
	return nil
}

// answering is work that has already been done.
type answering struct{ steps chan port.Step }

func (a answering) Steps() <-chan port.Step { return a.steps }
func (a answering) Stop() error             { return nil }

// panelled is this window's agent and a client talking to it the way the page
// does.
func panelled(t *testing.T, api *API) numenv1connect.AgentServiceClient {
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

// A question carries the card it is about, and every part of it reaches the
// agent: what was written, the deck in front of the person, and which
// conversation they asked it in.
func TestWhatIsAskedAboutACardReachesTheAgent(t *testing.T) {
	agent := &asking{took: make(chan port.Task, 1)}
	api := &API{}
	api.Answers(agent)

	stream, err := panelled(t, api).Ask(t.Context(), connect.NewRequest(&v1.AskRequest{
		Asked:        "why is it called that",
		Focus:        "decks/Words.md",
		Conversation: "3f4g5h6j7k",
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	for stream.Receive() {
	}

	task := <-agent.took
	if task.Asked != "why is it called that" {
		t.Errorf("asked %q", task.Asked)
	}
	if task.Focus != "decks/Words.md" {
		t.Errorf("focused on %q", task.Focus)
	}
	if task.Conversation != "3f4g5h6j7k" {
		t.Errorf("in conversation %q", task.Conversation)
	}
}

// Every kind of step the agent takes reaches the page as itself, in the order
// it was taken.
func TestEveryStepReachesThePageAsItself(t *testing.T) {
	agent := &asking{took: make(chan port.Task, 1), takes: []port.Step{
		{Kind: port.StepThinking},
		{Kind: port.StepSearch, Tool: "note_search", About: "leaf mould", Written: 11,
			Place: domain.Place{Path: "decks/Words.md", Start: 4, Length: 9}},
		{Kind: port.StepAnswered},
		{Kind: port.StepSaying, Text: "Because the leaves make it."},
		{Kind: port.StepStopped, Failed: "the agent went away"},
	}}
	api := &API{}
	api.Answers(agent)

	stream, err := panelled(t, api).Ask(t.Context(), connect.NewRequest(&v1.AskRequest{Asked: "why"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	var said []string
	for stream.Receive() {
		switch step := stream.Msg().GetStep().(type) {
		case *v1.AskResponse_Thinking:
			said = append(said, "thinking")
		case *v1.AskResponse_Doing:
			said = append(said, "doing "+step.Doing.GetTool()+" "+step.Doing.GetPath())
		case *v1.AskResponse_Answered:
			said = append(said, "answered")
		case *v1.AskResponse_Said:
			said = append(said, "said "+step.Said)
		case *v1.AskResponse_Stopped:
			said = append(said, "stopped "+step.Stopped)
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"thinking",
		"doing note_search decks/Words.md",
		"answered",
		"said Because the leaves make it.",
		"stopped the agent went away",
	}
	if strings.Join(said, "|") != strings.Join(want, "|") {
		t.Errorf("the page heard %q", said)
	}
}

// A window that can reach no agent says so, and answers nothing about a card.
func TestAWindowWithNoAgentAnswersNothingAboutACard(t *testing.T) {
	client := panelled(t, &API{})

	stream, err := client.Ask(t.Context(), connect.NewRequest(&v1.AskRequest{Asked: "why"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	for stream.Receive() {
	}
	if code := connect.CodeOf(stream.Err()); code != connect.CodeUnimplemented {
		t.Errorf("asking answered %v", code)
	}

	_, err = client.Finish(t.Context(), connect.NewRequest(&v1.FinishRequest{Conversation: "one"}))
	if code := connect.CodeOf(err); code != connect.CodeUnimplemented {
		t.Errorf("finishing answered %v", code)
	}
}

// A conversation said to be over reaches the agent under the name its questions
// carried.
func TestAConversationSaidToBeOverReachesTheAgent(t *testing.T) {
	agent := &asking{over: make(chan string, 1)}
	api := &API{}
	api.Answers(agent)

	if _, err := panelled(t, api).Finish(t.Context(),
		connect.NewRequest(&v1.FinishRequest{Conversation: "3f4g5h6j7k"})); err != nil {
		t.Fatal(err)
	}
	if over := <-agent.over; over != "3f4g5h6j7k" {
		t.Errorf("finished %q", over)
	}
}

// Whether a card can be asked about at all.
func TestWhetherACardCanBeAskedAboutIsSaid(t *testing.T) {
	nothing := &API{}
	nothing.Unreachable.Store("no agent is named in the settings")
	said, err := nothing.Asking(t.Context(), connect.NewRequest(&v1.AskingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if said.Msg.GetUnreachable() == "" {
		t.Error("a window with no agent says it can be asked")
	}

	reachable := &API{}
	reachable.Unreachable.Store("")
	said, err = reachable.Asking(t.Context(), connect.NewRequest(&v1.AskingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if why := said.Msg.GetUnreachable(); why != "" {
		t.Errorf("a window with an agent says %q", why)
	}
}

// The page asks once, as it opens, and no sitting is open then. An agent that
// is started when a person sits down to a vault has not started yet, and the
// page is not told this window can ask nothing.
func TestAWindowAskedBeforeASittingIsNotSaidToHaveNoAgent(t *testing.T) {
	api := &API{}
	api.Unreachable.Store("")

	said, err := api.Asking(t.Context(), connect.NewRequest(&v1.AskingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if why := said.Msg.GetUnreachable(); why != "" {
		t.Errorf("before anybody sat down the page was told %q", why)
	}
}

// Why the agent could not be served is what the page is told, in the words
// whoever tried to serve it used.
func TestWhyTheAgentCouldNotBeServedReachesThePage(t *testing.T) {
	api := &API{}
	api.Unreachable.Store("claude is not on this machine")

	said, err := api.Asking(t.Context(), connect.NewRequest(&v1.AskingRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if why := said.Msg.GetUnreachable(); why != "claude is not on this machine" {
		t.Errorf("the page was told %q", why)
	}
}

// The window serves the agent beside the cards. A path belonging to neither is
// still the page's own file, so the routes stand in front of the files and not
// over them.
func TestTheAgentIsServedBesideTheCards(t *testing.T) {
	handler := (&API{}).Serving(http.NotFoundHandler())
	asked := func(route string) int {
		r := httptest.NewRequest(http.MethodPost, route, strings.NewReader("{}"))
		r.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		handler.ServeHTTP(out, r)
		return out.Code
	}

	for _, route := range []string{"/numen.v1.AgentService/Ask", "/numen.v1.AgentService/Finish"} {
		if code := asked(route); code == http.StatusNotFound {
			t.Errorf("%s is not served", route)
		}
	}
	if code := asked("/built/index.css"); code != http.StatusNotFound {
		t.Errorf("a file of the page answered %d", code)
	}
}

// The agent works the vault the person is sitting to, and it is told which when
// the sitting opens.
func TestTheAgentIsToldWhichVaultTheSittingIsOn(t *testing.T) {
	api, vaults := windowed(t, deck, other)
	var told []string
	api.Sat = func(_ context.Context, v domain.Vault) { told = append(told, v.ID) }

	var opened []string
	for _, v := range vaults {
		if _, err := api.Start(t.Context(),
			connect.NewRequest(&v1.StartRequest{VaultId: v.ID})); err != nil {
			t.Fatal(err)
		}
		opened = append(opened, v.ID)
	}

	if strings.Join(told, "|") != strings.Join(opened, "|") {
		t.Errorf("sittings opened on %v and the agent was told %v", opened, told)
	}
}
