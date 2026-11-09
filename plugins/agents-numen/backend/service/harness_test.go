package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
)

const vaultToken = "vault-token"

// turn is what the fake model answers to one request: events in order, then the error the stream ends with.
type turn struct {
	events []connection.ChatEvent
	err    error
}

func newTextTurn(text string) turn {
	return turn{events: []connection.ChatEvent{
		{TextDelta: text}, {Finish: &connection.Finish{Reason: connection.FinishStop}},
	}}
}

func newCallTurn(id, tool, args string) turn {
	return turn{events: []connection.ChatEvent{
		{TextDelta: "Let me look. "},
		{ToolCall: &connection.ToolCall{ID: id, Name: tool, Arguments: args}},
		{Finish: &connection.Finish{Reason: connection.FinishToolCalls}},
	}}
}

// fakeModel answers each request with the next turn it holds, and the last turn again after that. It
// keeps the requests it received.
type fakeModel struct {
	mu       sync.Mutex
	turns    []turn
	requests []connection.ChatRequest
	failOpen error
}

func (f *fakeModel) chat(_ context.Context, request connection.ChatRequest) (chatStream, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, request)
	if f.failOpen != nil {
		return nil, f.failOpen
	}
	index := min(len(f.requests), len(f.turns)) - 1
	return &fakeStream{turn: f.turns[index]}, nil
}

func (f *fakeModel) getRequests() []connection.ChatRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]connection.ChatRequest(nil), f.requests...)
}

type fakeStream struct {
	turn   turn
	next   int
	closed atomic.Bool
}

func (s *fakeStream) Next() (connection.ChatEvent, bool) {
	if s.next >= len(s.turn.events) {
		return connection.ChatEvent{}, false
	}
	s.next++
	return s.turn.events[s.next-1], true
}

func (s *fakeStream) Err() error {
	if s.next < len(s.turn.events) {
		return nil
	}
	return s.turn.err
}

func (s *fakeStream) Close() error {
	s.closed.Store(true)
	return nil
}

// fakeVault is a tool server with two tools, behind the token, that keeps the arguments of every
// call.
type fakeVault struct {
	mu    sync.Mutex
	calls []string
	metas []sdk.Meta
}

func (v *fakeVault) serve(t *testing.T) string {
	t.Helper()
	server := sdk.NewServer(&sdk.Implementation{Name: "numen", Version: "1"}, nil)
	schema := map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}}
	server.AddTool(&sdk.Tool{Name: "note_read", Description: "Reads a note", InputSchema: schema},
		func(_ context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			v.mu.Lock()
			v.calls = append(v.calls, string(req.Params.Arguments))
			v.metas = append(v.metas, req.Params.GetMeta())
			v.mu.Unlock()
			return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "Plan: water the garden."}}}, nil
		})
	server.AddTool(&sdk.Tool{Name: "note_delete", Description: "Deletes a note", InputSchema: schema},
		func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			t.Error("a tool the task did not name was called")
			return &sdk.CallToolResult{}, nil
		})
	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+vaultToken {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func (v *fakeVault) getCalls() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]string(nil), v.calls...)
}

func newTask(url, prompt string) *agentrun.Request {
	return &agentrun.Request{
		Prompt:           prompt,
		Instructions:     "Answer from the vault.",
		Tools:            []string{"note_read"},
		ToolDeclarations: []toolcall.ToolDeclaration{{Name: "note_read", Title: "Read a note", Kind: "read", About: "path"}},
		ToolsURL:         url,
		ToolsToken:       vaultToken,
	}
}

func ask(t *testing.T, s *Server, task *agentrun.Request) []*agentrun.Response {
	t.Helper()
	var steps []*agentrun.Response
	if err := s.Ask(t.Context(), task, func(r *agentrun.Response) error {
		steps = append(steps, r)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return steps
}

func describeSteps(steps []*agentrun.Response) string {
	words := make([]string, 0, len(steps))
	for _, one := range steps {
		switch {
		case one.IsModelStarted():
			words = append(words, "model")
		case one.GetToolCall() != nil:
			words = append(words, "call")
		case one.IsToolFinished():
			words = append(words, "finished")
		case one.GetText() != "":
			if len(words) == 0 || words[len(words)-1] != "said" {
				words = append(words, "said")
			}
		default:
			words = append(words, "stopped")
		}
	}
	return strings.Join(words, " ")
}

func getSpoken(steps []*agentrun.Response) []string {
	var response []string
	for _, one := range steps {
		if text := one.GetText(); text != "" {
			response = append(response, text)
		}
	}
	return response
}

func newServer(model *fakeModel) *Server {
	s := &Server{chat: model.chat}
	s.Configure(map[string]string{SettingConnections: rowsJSON(connection.Row{Connection: "c00000001", Model: "m1", Reasoning: "low"})})
	return s
}

func rowsJSON(rows ...connection.Row) string {
	items := make([]string, len(rows))
	for i, one := range rows {
		items[i] = fmt.Sprintf(`{"connection":%q,"model":%q,"reasoning":%q}`, one.Connection, one.Model, one.Reasoning)
	}
	return "[" + strings.Join(items, ",") + "]"
}

func assertToolSteps(t *testing.T, steps []*agentrun.Response) {
	t.Helper()
	if got := describeSteps(steps); got != "model said call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	call := steps[2].GetToolCall()
	if call.Tool != "Read a note" || call.Kind != "read" || call.Call != "call-1" || call.Path != "Plan.md" {
		t.Errorf("the call is %+v", call)
	}
	if response, last := strings.Join(getSpoken(steps), ""), steps[len(steps)-1]; response != "Let me look. Water the garden." || last.GetStopped() != reasonDone {
		t.Errorf("the answer is %q, stopped %q", response, last.GetStopped())
	}
}

func assertToolRequests(t *testing.T, requests []connection.ChatRequest) {
	t.Helper()
	if len(requests) != 2 {
		t.Fatalf("the model was asked %d times", len(requests))
	}
	first := requests[0]
	if len(first.Tools) != 1 || first.Tools[0].Name != "note_read" || !strings.Contains(string(first.Tools[0].Parameters), `"path"`) {
		t.Errorf("the model was offered %+v", first.Tools)
	}
	if first.Messages[0].Role != "system" || first.Messages[1].Role != "user" || first.Messages[1].Text != "What is the plan?" {
		t.Errorf("the first request is %+v", first)
	}
	if first.Priority != connection.PriorityInteractive {
		t.Errorf("the priority is %v, want interactive: a person waits for the answer", first.Priority)
	}
	messages := requests[1].Messages
	asked, fedBack := messages[len(messages)-2], messages[len(messages)-1]
	if asked.Role != "assistant" || len(asked.ToolCalls) != 1 || asked.ToolCalls[0].ID != "call-1" || asked.Text != "Let me look. " {
		t.Errorf("the model was reminded of %+v", asked)
	}
	if fedBack.Role != "tool" || fedBack.ToolCallID != "call-1" || fedBack.Text != "Plan: water the garden." {
		t.Errorf("the model received %+v", fedBack)
	}
}

func TestAToolTheModelCallsRunsAndItsResultIsFedBack(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("call-1", "note_read", `{"path":"Plan.md"}`), newTextTurn("Water the garden.")}}
	vault := &fakeVault{}
	s := newServer(model)

	steps := ask(t, s, newTask(vault.serve(t), "What is the plan?"))
	assertToolSteps(t, steps)

	if calls := vault.getCalls(); len(calls) != 1 || calls[0] != `{"path":"Plan.md"}` {
		t.Errorf("the vault was called with %v", calls)
	}

	assertToolRequests(t, model.getRequests())
}

func TestAModelThatKeepsCallingToolsStopsAtTheTurnLimit(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(model)
	task := newTask(vault.serve(t), "Read forever.")
	task.MaxSteps = 2

	steps := ask(t, s, task)

	if got := describeSteps(steps); got != "model said call finished model said call finished stopped" {
		t.Fatalf("steps are %q", got)
	}
	if steps[len(steps)-1].GetStopped() != reasonTurns {
		t.Errorf("stopped %q", steps[len(steps)-1].GetStopped())
	}
	if len(model.getRequests()) != 2 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}

func TestEveryRowFailingBeforeAnyTextStopsTheRunWithEachConnectionAndReason(t *testing.T) {
	model := &fakeModel{failOpen: &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
	}}}
	s := newServer(model)

	steps := ask(t, s, newTask("", "Hello?"))

	if got := describeSteps(steps); got != "model stopped" {
		t.Fatalf("steps are %q", got)
	}
	if reason := steps[1].GetStopped(); reason != "OpenRouter: payment required; Ollama: unreachable" {
		t.Errorf("stopped %q", reason)
	}
}

func TestAFailureInTheMiddleOfTheAnswerKeepsTheTextAndNamesTheConnection(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "OpenRouter", Reason: "unreachable"}}}
	model := &fakeModel{turns: []turn{{
		events: []connection.ChatEvent{
			{TextDelta: "The plan is "},
			{ToolCall: &connection.ToolCall{ID: "c", Name: "note_read", Arguments: `{"path":"A.md"}`}},
		},
		err: failure,
	}}}
	vault := &fakeVault{}
	s := newServer(model)
	task := newTask(vault.serve(t), "Plan?")
	task.Conversation = "talk-1"

	steps := ask(t, s, task)

	if got := describeSteps(steps); got != "model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	if keptText := strings.Join(getSpoken(steps), ""); keptText != "The plan is " {
		t.Errorf("the text kept is %q", keptText)
	}
	if reason := steps[len(steps)-1].GetStopped(); reason != "OpenRouter: unreachable" {
		t.Errorf("stopped %q", reason)
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("a tool call of a failed turn was run: %v", calls)
	}
	if got := len(model.getRequests()); got != 1 {
		t.Errorf("the model was asked %d times, want once: nothing is repeated by itself", got)
	}
	if history := s.getConversation("talk-1"); len(history) != 0 {
		t.Errorf("a failed turn went into the conversation: %+v", history)
	}
}

func TestARejectionNamesTheConnectionAndTheReason(t *testing.T) {
	model := &fakeModel{failOpen: &connection.RejectedError{ConnectionName: "OpenRouter", Reason: "bad request"}}
	s := newServer(model)

	steps := ask(t, s, newTask("", "Hello?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != "OpenRouter: bad request" {
		t.Errorf("stopped %q", reason)
	}
}

func TestAnAgentWithNoRowStopsBeforeAnyRequestAndNamesWhereToChooseOne(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	s := &Server{chat: model.chat}

	steps := ask(t, s, newTask("", "Hello?"))

	if len(steps) != 1 || steps[0].GetStopped() != reasonNoConnection {
		t.Fatalf("steps are %v", steps)
	}
	if len(model.getRequests()) != 0 {
		t.Error("the model was asked")
	}
}

func TestACoreWithNoConnectionForTheAgentStopsTheRunWithTheSameWords(t *testing.T) {
	model := &fakeModel{failOpen: connection.ErrNoConnection}
	s := newServer(model)

	steps := ask(t, s, newTask("", "Hello?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != reasonNoConnection {
		t.Errorf("stopped %q", reason)
	}
}

func TestUnreachableVaultStopsRunBeforeModelCall(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	s := newServer(model)
	nowhere := httptest.NewServer(http.NotFoundHandler())
	defer nowhere.Close()
	task := newTask(nowhere.URL, "Hello?")

	steps := ask(t, s, task)

	if len(steps) != 1 || !strings.HasPrefix(steps[0].GetStopped(), "the agent could not reach this vault") {
		t.Fatalf("steps are %v", steps)
	}
	if len(model.getRequests()) != 0 {
		t.Error("the model was asked")
	}
}

func TestTheOptionsOfTheSettingsGoWithEveryRequest(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	s := newServer(model)
	s.Configure(map[string]string{SettingTemperature: "0.5", SettingMaxTokens: "2048"})

	ask(t, s, newTask("", "Hello?"))

	request := model.getRequests()[0]
	if request.Temperature == nil || *request.Temperature != 0.5 || request.MaxTokens != 2048 {
		t.Errorf("temperature = %v, max tokens = %d", request.Temperature, request.MaxTokens)
	}
}

func TestAnAnswerTruncatedAtItsLengthLimitIsReportedSo(t *testing.T) {
	model := &fakeModel{turns: []turn{{events: []connection.ChatEvent{
		{TextDelta: "The plan is to"}, {Finish: &connection.Finish{Reason: connection.FinishLength}},
	}}}}
	s := newServer(model)

	steps := ask(t, s, newTask("", "Plan?"))

	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != reasonCut {
		t.Errorf("steps are %q, stopped %q", got, steps[len(steps)-1].GetStopped())
	}
}

func TestATruncatedAnswerCallsNoTool(t *testing.T) {
	model := &fakeModel{turns: []turn{{events: []connection.ChatEvent{
		{ToolCall: &connection.ToolCall{ID: "c", Name: "note_read", Arguments: "{}"}}, {Finish: &connection.Finish{Reason: connection.FinishLength}},
	}}}}
	vault := &fakeVault{}
	s := newServer(model)

	steps := ask(t, s, newTask(vault.serve(t), "Read."))

	if got := describeSteps(steps); got != "model stopped" || steps[1].GetStopped() != reasonCut {
		t.Errorf("steps are %q", got)
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestConversationContinuesUntilFinished(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("Noted.")}}
	s := newServer(model)
	task := newTask("", "Remember the garden.")
	task.Conversation = "talk-1"

	ask(t, s, task)
	task.Prompt = "What did I ask?"
	ask(t, s, task)
	if err := s.FinishConversation(t.Context(), "talk-1"); err != nil {
		t.Fatal(err)
	}
	ask(t, s, task)

	requests := model.getRequests()
	second := requests[1].Messages
	if len(second) != 4 || second[1].Text != "Remember the garden." || second[2].Text != "Noted." || second[3].Text != "What did I ask?" {
		t.Errorf("the second request was %+v", second)
	}
	if len(requests[2].Messages) != 2 {
		t.Errorf("a finished conversation went on: %+v", requests[2].Messages)
	}
}

func TestTheAgentNamesNeitherAModelNorAConnectionToTheCore(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("the answer")}}
	s := newServer(model)
	task := newTask("", "why?")
	task.Tools, task.ToolDeclarations, task.Model, task.Reasoning = nil, nil, "routed", "high"

	ask(t, s, task)

	request, err := json.Marshal(model.getRequests()[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"routed", "high", "m1", "c00000001", "Model", "Connection"} {
		if strings.Contains(string(request), name) {
			t.Errorf("the request %s names %q: the row of the settings decides", request, name)
		}
	}
}

func TestAskAgentStreamsTheStepsOverConnect(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("call-1", "note_read", `{"path":"Plan.md"}`), newTextTurn("Done.")}}
	vault := &fakeVault{}
	s := newServer(model)
	mux := http.NewServeMux()
	mux.Handle(agentrun.NewHandler(s))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stream, err := agentrun.NewClient(srv.Client(), srv.URL).Ask(t.Context(), newTask(vault.serve(t), "Plan?"))
	if err != nil {
		t.Fatal(err)
	}
	var steps []*agentrun.Response
	for stream.Receive() {
		steps = append(steps, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if got := describeSteps(steps); got != "model said call finished model said stopped" {
		t.Errorf("steps are %q", got)
	}
}

func TestACallWithNoArgumentsIsReportedBackWithAnEmptyObject(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("c", "note_read", ""), newTextTurn("ok")}}
	vault := &fakeVault{}
	s := newServer(model)

	ask(t, s, newTask(vault.serve(t), "Read."))

	messages := model.getRequests()[1].Messages
	if args := messages[len(messages)-2].ToolCalls[0].Arguments; args != "{}" {
		t.Errorf("call returned with %q", args)
	}
	if calls := vault.getCalls(); len(calls) != 1 || calls[0] != "{}" {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestTheVaultTokenIsNotSentOverPlainHTTPToAnotherMachine(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	s := newServer(model)

	steps := ask(t, s, newTask("http://192.0.2.1/mcp", "Hello?"))

	if len(steps) != 1 || !strings.Contains(steps[0].GetStopped(), errPlainToken.Error()) {
		t.Errorf("steps are %v", steps)
	}
}

func TestTheVaultTokenIsNotFollowedToPlainHTTP(t *testing.T) {
	redirecting := httptest.NewServer(http.RedirectHandler("http://192.0.2.1/mcp", http.StatusTemporaryRedirect))
	defer redirecting.Close()
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	s := newServer(model)

	steps := ask(t, s, newTask(redirecting.URL, "Hello?"))

	if len(steps) != 1 || !strings.Contains(steps[0].GetStopped(), errDowngrade.Error()) {
		t.Errorf("steps are %v", steps)
	}
}

func TestAStepThatCannotBeSentEndsTheRunWithItsError(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("c", "note_read", `{"path":"A.md"}`), newTextTurn("ok")}}
	vault := &fakeVault{}
	s := newServer(model)
	removed := errors.New("the window closed")
	dispatched := 0

	err := s.Ask(t.Context(), newTask(vault.serve(t), "Read."), func(*agentrun.Response) error {
		dispatched++
		if dispatched == 3 {
			return removed
		}
		return nil
	})

	if !errors.Is(err, removed) {
		t.Errorf("the run ended with %v", err)
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("the vault was called after the call could not be told: %v", calls)
	}
}

// A task that names no tool is one answer: the model is offered no tool, and the vault is not
// connected to, whatever endpoint the task holds.
func TestNoToolsWhenTaskNamesNone(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("the answer")}}
	var connected atomic.Bool
	vault := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { connected.Store(true) }))
	t.Cleanup(vault.Close)
	s := newServer(model)
	task := newTask(vault.URL, "why?")
	task.Tools, task.ToolDeclarations = nil, nil

	steps := ask(t, s, task)

	if got := strings.Join(getSpoken(steps), ""); got != "the answer" {
		t.Errorf("the answer is %q", got)
	}
	if connected.Load() {
		t.Error("the vault was connected to")
	}
	requests := model.getRequests()
	if len(requests) != 1 || len(requests[0].Tools) != 0 {
		t.Errorf("the model was asked %+v", requests)
	}
}

func TestAnyMCPToolWithoutPrefixCanBeCalled(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("call-custom", "custom_remote_tool", `{"query":"test"}`), newTextTurn("Result received.")}}
	mcpServer := sdk.NewServer(&sdk.Implementation{Name: "custom", Version: "1"}, nil)
	var calledWith string
	mcpServer.AddTool(&sdk.Tool{Name: "custom_remote_tool", Description: "A custom MCP tool", InputSchema: map[string]any{"type": "object"}},
		func(_ context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			calledWith = string(req.Params.Arguments)
			return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "Tool output"}}}, nil
		})
	handler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return mcpServer }, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+vaultToken {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	s := newServer(model)
	task := &agentrun.Request{
		Prompt:       "Run custom tool",
		Instructions: "Follow instructions",
		Tools:        []string{"custom_remote_tool"},
		ToolsURL:     srv.URL,
		ToolsToken:   vaultToken,
	}
	steps := ask(t, s, task)
	if calledWith != `{"query":"test"}` {
		t.Errorf("custom tool was called with %q, want %q", calledWith, `{"query":"test"}`)
	}
	if response := strings.Join(getSpoken(steps), ""); response != "Let me look. Result received." {
		t.Errorf("the answer is %q", response)
	}
}

// The context of the task is passed to every tool the model calls as the `_meta` of
// the call.
func TestTheContextOfATaskIsTheMetaOfEveryCall(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("call-1", "note_read", `{"path":"Plan.md"}`), newTextTurn("ok")}}
	vault := &fakeVault{}
	s := newServer(model)
	task := newTask(vault.serve(t), "What is on this card?")
	task.Context = map[string]string{"md.example.cards/card": "k7"}

	ask(t, s, task)

	vault.mu.Lock()
	defer vault.mu.Unlock()
	if len(vault.metas) != 1 || vault.metas[0]["md.example.cards/card"] != "k7" {
		t.Errorf("the calls carried %v", vault.metas)
	}
}

func TestTheMaxStepsSettingLimitsATaskThatNamesNoLimit(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(model)
	s.Configure(map[string]string{SettingMaxSteps: "3"})

	steps := ask(t, s, newTask(vault.serve(t), "Read forever."))

	if steps[len(steps)-1].GetStopped() != reasonTurns {
		t.Errorf("stopped %q", steps[len(steps)-1].GetStopped())
	}
	if len(model.getRequests()) != 3 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}

func TestATaskLimitOverridesTheMaxStepsSetting(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(model)
	s.Configure(map[string]string{SettingMaxSteps: "5"})
	task := newTask(vault.serve(t), "Read forever.")
	task.MaxSteps = 2

	ask(t, s, task)

	if len(model.getRequests()) != 2 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}

// blockingStream delivers one event and then waits for the context of its request.
type blockingStream struct {
	ctx    context.Context
	isSent bool
	ended  chan struct{}
}

func (s *blockingStream) Next() (connection.ChatEvent, bool) {
	if !s.isSent {
		s.isSent = true
		return connection.ChatEvent{TextDelta: "partial"}, true
	}
	<-s.ctx.Done()
	close(s.ended)
	return connection.ChatEvent{}, false
}

func (s *blockingStream) Err() error { return s.ctx.Err() }

func (s *blockingStream) Close() error { return nil }

func TestACancelledTaskEndsTheStreamOfTheModel(t *testing.T) {
	ended := make(chan struct{})
	s := &Server{chat: func(ctx context.Context, _ connection.ChatRequest) (chatStream, error) {
		return &blockingStream{ctx: ctx, ended: ended}, nil
	}}
	s.Configure(map[string]string{SettingConnections: rowsJSON(connection.Row{Connection: "c1", Model: "m"})})
	ctx, cancel := context.WithCancel(t.Context())
	finish := make(chan error, 1)
	firstText := make(chan struct{}, 1)
	go func() {
		finish <- s.Ask(ctx, newTask("", "Hello?"), func(r *agentrun.Response) error {
			if r.GetText() != "" {
				firstText <- struct{}{}
			}
			return nil
		})
	}()
	<-firstText

	cancel()

	if err := <-finish; err != nil {
		t.Errorf("a canceled task returned %v", err)
	}
	<-ended
}

func TestEveryStreamOfTheModelIsClosed(t *testing.T) {
	stream := &fakeStream{turn: newTextTurn("hi")}
	s := &Server{chat: func(context.Context, connection.ChatRequest) (chatStream, error) { return stream, nil }}
	s.Configure(map[string]string{SettingConnections: rowsJSON(connection.Row{Connection: "c1", Model: "m"})})

	ask(t, s, newTask("", "Hello?"))

	if !stream.closed.Load() {
		t.Error("the stream of the model was left open")
	}
}

func TestTheProgramIsInstalledWhileARowNamesAConnectionAndOffersItsModel(t *testing.T) {
	s := newServer(&fakeModel{})

	programs, err := s.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	program := programs[0]
	if program.Name != ProgramName || !program.IsInstalled || !program.LimitsTools || len(program.Models) != 1 {
		t.Fatalf("the program is %+v", program)
	}
	if model := program.Models[0]; model.Name != "m1" || !model.IsDefault || len(model.ReasoningLevels) != 0 {
		t.Errorf("the model is %+v", model)
	}
	state, err := s.GetState(t.Context())
	if err != nil || state.Unreachable != "" {
		t.Errorf("state %v, %v", state, err)
	}
}

func TestTheProgramIsNotInstalledWhileNoRowNamesAConnection(t *testing.T) {
	s := &Server{}

	programs, err := s.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if programs[0].IsInstalled || len(programs[0].Models) != 0 {
		t.Errorf("the program is %+v", programs[0])
	}
	state, err := s.GetState(t.Context())
	if err != nil || state.Unreachable != reasonNoConnection {
		t.Errorf("state %v, %v, want unreachable for %q", state, err, reasonNoConnection)
	}
}

func TestOnlyTheFirstRowIsUsed(t *testing.T) {
	s := &Server{}
	s.Configure(map[string]string{SettingConnections: rowsJSON(
		connection.Row{Connection: "c1", Model: "first"}, connection.Row{Connection: "c2", Model: "second"},
	)})

	programs, err := s.GetPrograms(t.Context(), nil)

	if err != nil || len(programs[0].Models) != 1 || programs[0].Models[0].Name != "first" {
		t.Errorf("programs = %+v, %v", programs, err)
	}
}

func TestARowsSettingThatIsNoListKeepsTheRowsAsTheyWere(t *testing.T) {
	s := newServer(&fakeModel{})

	s.Configure(map[string]string{SettingConnections: "not a list"})

	programs, err := s.GetPrograms(t.Context(), nil)
	if err != nil || !programs[0].IsInstalled {
		t.Errorf("programs = %+v, %v", programs, err)
	}
}
