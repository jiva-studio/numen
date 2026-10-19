package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func askCompletion(text string) *connect.Request[numenv1.CompleteRequest] {
	return connect.NewRequest(&numenv1.CompleteRequest{Question: &numenv1.CompletionQuestion{
		Messages: []*numenv1.CompletionMessage{
			{Role: "system", Text: "be brief"},
			{Role: "user", Text: text},
		},
	}})
}

func serveAgents(agents ...acpAgent) *agentServer {
	return &agentServer{acpAgents: func() []acpAgent { return agents }}
}

func TestCompleteCollectsTheAgentsChunksIntoOneAnswer(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	res, err := serveAgents(agent).Complete(t.Context(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Msg.GetAnswer().GetText(); got != "the answer" {
		t.Fatalf("got answer %q, want %q", got, "the answer")
	}
	seen := readRecord(t, record)
	prompts := seen["session/prompt"]
	if len(prompts) != 1 {
		t.Fatalf("the agent was prompted %d times, want 1", len(prompts))
	}
	var texts []string
	for _, block := range prompts[0]["params"].(map[string]any)["prompt"].([]any) {
		texts = append(texts, block.(map[string]any)["text"].(string))
	}
	if strings.Join(texts, "|") != "be brief|why?" {
		t.Errorf("the prompt held %q, want the system text and then the question", texts)
	}
	if len(seen["session/close"]) != 1 {
		t.Errorf("the session was closed %d times, want 1", len(seen["session/close"]))
	}
}

// The session has no MCP servers, the client offers no file system or terminal, and every request
// of the agent is refused.
func TestCompleteOffersTheAgentNoTools(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "plain")

	if _, err := serveAgents(agent).Complete(t.Context(), askCompletion("why?")); err != nil {
		t.Fatal(err)
	}
	seen := readRecord(t, record)
	session := seen["session/new"][0]["params"].(map[string]any)
	if servers := session["mcpServers"].([]any); len(servers) != 0 {
		t.Errorf("the session was given MCP servers %v", servers)
	}
	if cwd, _ := session["cwd"].(string); cwd == "" || !filepath.IsAbs(cwd) {
		t.Errorf("the session was opened in %q", cwd)
	}
	capabilities := seen["initialize"][0]["params"].(map[string]any)["clientCapabilities"].(map[string]any)
	if capabilities["terminal"] != false {
		t.Errorf("the client offered a terminal: %v", capabilities)
	}
	for name, offered := range capabilities["fs"].(map[string]any) {
		if offered != false {
			t.Errorf("the client offered %s", name)
		}
	}
	replies := seen["reply"]
	if len(replies) != 2 {
		t.Fatalf("the agent had %d replies, want 2", len(replies))
	}
	outcome := replies[0]["result"].(map[string]any)["outcome"].(map[string]any)["outcome"]
	if outcome != "cancelled" {
		t.Errorf("the permission was answered %v, want cancelled", outcome)
	}
	if replies[1]["error"] == nil || replies[1]["result"] != nil {
		t.Errorf("the file read was answered %v, want an error", replies[1])
	}
}

func TestCompleteIsUnavailableWhenNoACPAgentIsInstalled(t *testing.T) {
	svc := &agentServer{}

	res, err := svc.Complete(t.Context(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE || res.Msg.GetAnswer() != nil {
		t.Errorf("got %+v, want UNAVAILABLE", res.Msg)
	}
	if _, _, err := findCompletionAgent(getACPAgentsFromCommands(nil), ""); !errors.Is(err, errNoCompletionAgent) {
		t.Errorf("got %v, want %v", err, errNoCompletionAgent)
	}
}

func TestCompleteIsUnavailableForAnAgentThatIsNotInstalled(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "plain")
	request := askCompletion("why?")
	request.Msg.Model = "other/m1"

	res, err := serveAgents(agent).Complete(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
		t.Errorf("got %+v, want UNAVAILABLE", res.Msg)
	}
	if _, _, err := findCompletionAgent([]acpAgent{agent}, "other/m1"); !errors.Is(err, errAgentNotInstalled) {
		t.Errorf("got %v, want %v", err, errAgentNotInstalled)
	}
}

// Without a model the first installed agent answers.
func TestCompleteTakesTheFirstAgentWithoutAModel(t *testing.T) {
	first, firstRecord := fakeACPAgent(t, "first", "plain")
	second, secondRecord := fakeACPAgent(t, "second", "plain")

	if _, err := serveAgents(first, second).Complete(t.Context(), askCompletion("why?")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(firstRecord); err != nil {
		t.Errorf("the first agent was not asked: %v", err)
	}
	if _, err := os.Stat(secondRecord); err == nil {
		t.Error("the second agent was asked")
	}
}

func TestCompleteChoosesTheModelTheHostNames(t *testing.T) {
	for _, tc := range []struct {
		behaviour, method, field string
	}{
		{"config", "session/set_config_option", "value"},
		{"legacy", "session/set_model", "modelId"},
	} {
		t.Run(tc.behaviour, func(t *testing.T) {
			agent, record := fakeACPAgent(t, "fake", tc.behaviour)
			request := askCompletion("why?")
			request.Msg.Model = "fake/m2"

			res, err := serveAgents(agent).Complete(t.Context(), request)
			if err != nil || res.Msg.GetAnswer() == nil {
				t.Fatalf("got %+v, %v", res.Msg, err)
			}
			set := readRecord(t, record)[tc.method]
			if len(set) != 1 || set[0]["params"].(map[string]any)[tc.field] != "m2" {
				t.Errorf("the agent was sent %v, want %s with m2", set, tc.method)
			}
		})
	}
}

func TestCompleteRefusesAModelTheAgentDoesNotOffer(t *testing.T) {
	for _, model := range []string{"fake/m9", "plain/m1"} {
		program := strings.Split(model, "/")[0]
		behaviour := map[string]string{"fake": "config", "plain": "plain"}[program]
		agent, record := fakeACPAgent(t, program, behaviour)
		request := askCompletion("why?")
		request.Msg.Model = model

		res, err := serveAgents(agent).Complete(t.Context(), request)
		if err != nil {
			t.Fatal(err)
		}
		if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
			t.Errorf("%s: got %+v, want UNAVAILABLE", model, res.Msg)
		}
		if prompts := readRecord(t, record)["session/prompt"]; len(prompts) != 0 {
			t.Errorf("%s: the agent was prompted", model)
		}
	}
}

func TestCompleteIsUnavailableWhenTheAgentFails(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "fail")

	res, err := serveAgents(agent).Complete(t.Context(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE {
		t.Errorf("got %+v, want UNAVAILABLE", res.Msg)
	}
	_, err = openACPSession(t.Context(), agent)
	if err == nil || !strings.Contains(err.Error(), "not signed in") {
		t.Errorf("got %v, want the agent's own message", err)
	}
}

func TestCompleteStopsTheAgentWhenCancelled(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "hang")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	_, err := serveAgents(agent).Complete(ctx, askCompletion("why?"))
	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("got %v, want cancelled", err)
	}
}

func listCompletionModels(t *testing.T, svc *agentServer) []string {
	t.Helper()
	res, err := svc.ListCompletionModels(t.Context(), connect.NewRequest(&numenv1.ListCompletionModelsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, one := range res.Msg.GetModels() {
		entry := one.GetName() + "=" + one.GetTitle()
		if one.GetIsDefault() {
			entry += " (default)"
		}
		listed = append(listed, entry)
	}
	return listed
}

// Each agent lists the models its session reports, the first agent's current one the default; an
// agent that reports none, or cannot be asked, is listed alone for its own model.
func TestTheCompletionModelsAreThoseTheACPAgentsReport(t *testing.T) {
	config, _ := fakeACPAgent(t, "config", "config")
	legacy, _ := fakeACPAgent(t, "legacy", "legacy")
	plain, _ := fakeACPAgent(t, "plain", "plain")
	failing, _ := fakeACPAgent(t, "failing", "fail")

	got := strings.Join(listCompletionModels(t, serveAgents(config, legacy, plain, failing)), "\n")
	want := strings.Join([]string{
		"config/m1=Fake config · Model One (default)",
		"config/m2=Fake config · Model Two",
		"legacy/m1=Fake legacy · Model One",
		"legacy/m2=Fake legacy · Model Two",
		"plain=Fake plain (default model)",
		"failing=Fake failing (default model)",
	}, "\n")
	if got != want {
		t.Errorf("listed\n%s\nwant\n%s", got, want)
	}
}

func TestAnAgentWithoutModelsIsTheDefaultWhenFirst(t *testing.T) {
	plain, _ := fakeACPAgent(t, "plain", "plain")

	if got := listCompletionModels(t, serveAgents(plain)); len(got) != 1 || got[0] != "plain=Fake plain (default model) (default)" {
		t.Errorf("listed %v", got)
	}
}

func TestTheCompletionModelsAreKept(t *testing.T) {
	agent, record := fakeACPAgent(t, "fake", "config")
	svc := serveAgents(agent)

	first := listCompletionModels(t, svc)
	second := listCompletionModels(t, svc)
	if strings.Join(first, ",") != strings.Join(second, ",") {
		t.Errorf("listed %v, then %v", first, second)
	}
	if started := readRecord(t, record)["initialize"]; len(started) != 1 {
		t.Errorf("the agent was started %d times, want 1", len(started))
	}
}

func TestNoCompletionModelIsListedWithoutAnACPAgent(t *testing.T) {

	if got := listCompletionModels(t, &agentServer{}); len(got) != 0 {
		t.Errorf("listed %v", got)
	}
}

// A turn the agent ends refusing is no answer, whatever it said before.
func TestCompleteIsUnavailableWhenTheAgentRefuses(t *testing.T) {
	agent, _ := fakeACPAgent(t, "fake", "refuse")

	res, err := serveAgents(agent).Complete(t.Context(), askCompletion("why?"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetError() != numenv1.CompletionErrorCode_COMPLETION_ERROR_CODE_UNAVAILABLE || res.Msg.GetAnswer() != nil {
		t.Errorf("got %+v, want UNAVAILABLE", res.Msg)
	}
}

// An agent that failed to list is not asked again until the backoff has passed.
func TestAnAgentThatFailedToListIsLeftAlone(t *testing.T) {
	agent, record := fakeACPAgent(t, "failing", "fail")
	now := time.Unix(1000, 0)
	svc := serveAgents(agent)
	svc.acpModels.now = func() time.Time { return now }

	listCompletionModels(t, svc)
	listCompletionModels(t, svc)
	if started := readRecord(t, record)["started"]; len(started) != 1 {
		t.Fatalf("the agent was started %d times within the backoff, want 1", len(started))
	}
	now = now.Add(acpFailureBackoff)
	listCompletionModels(t, svc)
	if started := readRecord(t, record)["started"]; len(started) != 2 {
		t.Errorf("the agent was started %d times after the backoff, want 2", len(started))
	}
}
