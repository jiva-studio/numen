package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/toolcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/endpoint"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/chatapi"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/vaulttools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/answer"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/usecase/catalog"
)

const vaultToken = "vault-token"

// sentRequest is what the fake chat API was sent.
type sentRequest struct {
	Auth            string
	Model           string                      `json:"model"`
	Messages        []conversation.Message      `json:"messages"`
	Tools           []conversation.FunctionTool `json:"tools"`
	Stream          bool                        `json:"stream"`
	ReasoningEffort string                      `json:"reasoning_effort"`
	Reasoning       *struct {
		Effort string `json:"effort"`
	} `json:"reasoning"`
}

// fakeChat is an OpenAI-compatible chat API that answers each request with the next reply it holds,
// with the reply in pieces: its text word by word, and each tool call in three fragments. script,
// where it is set, is the data of the events that answer the first request. rejectsStream answers
// a streaming request 400 Bad Request, and ignoresStream answers it with the whole reply.
type fakeChat struct {
	mu            sync.Mutex
	replies       []conversation.Message
	script        []string
	status        int
	finish        string
	models        []string
	parameters    map[string][]string
	rejectsStream bool
	ignoresStream bool
	requests      []sentRequest
	replyCount    int
	t             testing.TB
}

func (f *fakeChat) serveModels(w http.ResponseWriter) {
	var listing struct {
		Data []map[string]any `json:"data"`
	}
	for _, one := range f.models {
		entry := map[string]any{"id": one, "name": strings.ToUpper(one)}
		if supported, found := f.parameters[one]; found {
			entry["supported_parameters"] = supported
		}
		listing.Data = append(listing.Data, entry)
	}
	encodeJSON(f.t, w, listing)
}

func (f *fakeChat) serveChat(w http.ResponseWriter, r *http.Request) {
	var got sentRequest
	decodeJSON(f.t, r.Body, &got)
	got.Auth = r.Header.Get("Authorization")
	f.requests = append(f.requests, got)
	if got.Stream && f.rejectsStream {
		http.Error(w, `{"error":{"message":"stream is not supported"}}`, http.StatusBadRequest)
		return
	}
	f.replyCount++
	if got.Stream && f.script != nil && f.replyCount == 1 {
		writeEvents(f.t, w, f.script)
		return
	}
	reply := f.replies[min(f.replyCount, len(f.replies))-1]
	if got.Stream && !f.ignoresStream {
		writeEvents(f.t, w, f.getPieces(reply))
		return
	}
	encodeJSON(f.t, w, map[string]any{"choices": []map[string]any{{"message": reply, "finish_reason": f.finish}}})
}

func (f *fakeChat) serve(t *testing.T) *httptest.Server {
	t.Helper()
	f.t = t
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.status != 0 {
			http.Error(w, "invalid key", f.status)
			return
		}
		if r.URL.Path == "/models" {
			f.serveModels(w)
			return
		}
		f.serveChat(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (f *fakeChat) getPieces(reply conversation.Message) []string {
	var events []string
	addDelta := func(delta map[string]any, finish any) {
		event, err := json.Marshal(map[string]any{"choices": []map[string]any{{"delta": delta, "finish_reason": finish}}})
		if err != nil {
			f.t.Error(err)
			return
		}
		events = append(events, string(event))
	}
	addDelta(map[string]any{"role": "assistant"}, nil)
	if reply.Content != nil {
		for _, word := range strings.SplitAfter(*reply.Content, " ") {
			addDelta(map[string]any{"content": word}, nil)
		}
	}
	for i, call := range reply.ToolCalls {
		half := len(call.Function.Arguments) / 2
		addDelta(map[string]any{"tool_calls": []map[string]any{{
			"index": i, "id": call.ID, "type": "function", "function": map[string]any{"name": call.Function.Name, "arguments": ""},
		}}}, nil)
		for _, part := range []string{call.Function.Arguments[:half], call.Function.Arguments[half:]} {
			addDelta(map[string]any{"tool_calls": []map[string]any{{"index": i, "function": map[string]any{"arguments": part}}}}, nil)
		}
	}
	finish := f.finish
	if finish == "" {
		finish = "stop"
	}
	addDelta(map[string]any{}, finish)
	return append(events, "[DONE]")
}

func writeEvents(tb testing.TB, w http.ResponseWriter, data []string) {
	tb.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, ok := w.(http.Flusher)
	if !ok {
		tb.Error("the response writer does not flush")
		return
	}
	for _, one := range data {
		if _, err := fmt.Fprintf(w, "data: %s\n\n", one); err != nil {
			tb.Error(err)
			return
		}
		flusher.Flush()
	}
}

func encodeJSON(tb testing.TB, w io.Writer, value any) {
	tb.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		tb.Errorf("encode the response: %v", err)
	}
}

func decodeJSON(tb testing.TB, r io.Reader, target any) {
	tb.Helper()
	if err := json.NewDecoder(r).Decode(target); err != nil {
		tb.Errorf("decode the request: %v", err)
	}
}

func (f *fakeChat) getRequests() []sentRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentRequest(nil), f.requests...)
}

func newCallReply(id, tool, args string) conversation.Message {
	call := conversation.ToolCall{ID: id, Type: "function"}
	call.Function.Name, call.Function.Arguments = tool, args
	return conversation.Message{Role: "assistant", ToolCalls: []conversation.ToolCall{call}}
}

func newTextReply(text string) conversation.Message {
	return conversation.Message{Role: "assistant", Content: new(text)}
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

func ask(t *testing.T, s *agent, task *agentrun.Request) []*agentrun.Response {
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

func getSpokenTexts(steps []*agentrun.Response) []string {
	var response []string
	for _, one := range steps {
		if text := one.GetText(); text != "" {
			response = append(response, text)
		}
	}
	return response
}

func keySecrets(value string) server.Secrets {
	return server.Secrets{settings.KeyAPIKey: {{ID: "s00000001", Name: "Work", Group: "openrouter", Value: value}}}
}

func newTestAgent(t *testing.T, opts ...chatapi.Option) *agent {
	t.Helper()
	a, err := newAgent(opts...)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newServer(t *testing.T, chatURL string) *agent {
	t.Helper()
	s := newTestAgent(t)
	s.configure(map[string]string{settings.KeyBaseURL: chatURL, settings.KeyModel: "m1"})
	s.applySecrets(keySecrets("host-key"))
	return s
}

func assertToolSteps(t *testing.T, steps []*agentrun.Response) {
	t.Helper()
	if got := describeSteps(steps); got != "model call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	call := steps[1].GetToolCall()
	if call.Tool != "Read a note" || call.Kind != "read" || call.Call != "call-1" || call.Path != "Plan.md" {
		t.Errorf("the call is %+v", call)
	}
	if response, last := strings.Join(getSpokenTexts(steps), ""), steps[len(steps)-1]; response != "Water the garden." || last.GetStopped() != answer.ReasonDone {
		t.Errorf("the answer is %q, stopped %q", response, last.GetStopped())
	}
}

func assertToolRequests(t *testing.T, requests []sentRequest) {
	t.Helper()
	if len(requests) != 2 {
		t.Fatalf("the model was asked %d times", len(requests))
	}
	first := requests[0]
	if len(first.Tools) != 1 || first.Tools[0].Function.Name != "note_read" || !strings.Contains(string(first.Tools[0].Function.Parameters), `"path"`) {
		t.Errorf("the model was offered %+v", first.Tools)
	}
	if first.Model != "m1" || first.Messages[0].Role != "system" || *first.Messages[1].Content != "What is the plan?" {
		t.Errorf("the first request is %+v", first)
	}
	fedBack := requests[1].Messages[len(requests[1].Messages)-1]
	if fedBack.Role != "tool" || fedBack.ToolCallID != "call-1" || *fedBack.Content != "Plan: water the garden." {
		t.Errorf("the model received %+v", fedBack)
	}
}

func TestAToolTheModelCallsRunsAndItsResultIsFedBack(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("call-1", "note_read", `{"path":"Plan.md"}`), newTextReply("Water the garden.")}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "What is the plan?"))
	assertToolSteps(t, steps)

	if calls := vault.getCalls(); len(calls) != 1 || calls[0] != `{"path":"Plan.md"}` {
		t.Errorf("the vault was called with %v", calls)
	}

	assertToolRequests(t, model.getRequests())
}

func TestAModelThatPreservesCallingToolsStopsAtTheTurnLimit(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)
	task := newTask(vault.serve(t), "Read forever.")
	task.MaxSteps = 2

	steps := ask(t, s, task)

	if got := describeSteps(steps); got != "model call finished model call finished stopped" {
		t.Fatalf("steps are %q", got)
	}
	if steps[len(steps)-1].GetStopped() != answer.ReasonTurns {
		t.Errorf("stopped %q", steps[len(steps)-1].GetStopped())
	}
	if len(model.getRequests()) != 2 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}

func TestAFailureOfTheChatAPIStopsTheRunWithItsReason(t *testing.T) {
	model := &fakeChat{status: http.StatusUnauthorized}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Hello?"))

	if got := describeSteps(steps); got != "model stopped" {
		t.Fatalf("steps are %q", got)
	}
	if reason := steps[1].GetStopped(); !strings.Contains(reason, "401") || !strings.Contains(reason, "invalid key") {
		t.Errorf("stopped %q", reason)
	}
}

func TestHarness_UnreachableVaultStopsRunBeforeModelCall(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("hi")}}
	s := newServer(t, model.serve(t).URL)
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

func TestTheKeyIsTheOneConfigurePassesOverAndNoneIsSentWithoutOne(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	model := &fakeChat{replies: []conversation.Message{newTextReply("hi")}}
	s := newTestAgent(t)
	s.configure(map[string]string{settings.KeyBaseURL: model.serve(t).URL})

	ask(t, s, newTask("", "Hello?"))
	s.applySecrets(keySecrets("host-key"))
	ask(t, s, newTask("", "Hello?"))
	s.applySecrets(server.Secrets{settings.KeyAPIKey: {}})
	ask(t, s, newTask("", "Hello?"))

	requests := model.getRequests()
	if requests[0].Auth != "" || requests[1].Auth != "Bearer host-key" || requests[2].Auth != "" {
		t.Errorf("authorization was %q, %q, %q", requests[0].Auth, requests[1].Auth, requests[2].Auth)
	}
	if requests[0].Model != settings.DefaultModel {
		t.Errorf("the model asked was %q", requests[0].Model)
	}
}

func TestHarness_ConversationContinuesUntilFinished(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("Noted.")}}
	s := newServer(t, model.serve(t).URL)
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
	if len(second) != 4 || *second[1].Content != "Remember the garden." || *second[2].Content != "Noted." || *second[3].Content != "What did I ask?" {
		t.Errorf("the second request was %+v", second)
	}
	if len(requests[2].Messages) != 2 {
		t.Errorf("a finished conversation went on: %+v", requests[2].Messages)
	}
}

func TestTheProgramIsInstalledWhileTheChatAPIListsItsModels(t *testing.T) {
	model := &fakeChat{models: []string{"m0", "m1"}}
	s := newServer(t, model.serve(t).URL)

	res, err := s.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	program := res[0]
	if program.Name != catalog.ProgramName || !program.IsInstalled || !program.LimitsTools || len(program.Models) != 2 {
		t.Fatalf("the program is %+v", program)
	}
	if program.Models[0].IsDefault || !program.Models[1].IsDefault || program.Models[1].Title != "M1" {
		t.Errorf("the models are %+v", program.Models)
	}
	state, err := s.GetState(t.Context())
	if err != nil || state.Unreachable != "" {
		t.Errorf("state %v, %v", state, err)
	}
}

func TestTheProgramIsNotInstalledWhileTheChatAPIRejects(t *testing.T) {
	model := &fakeChat{status: http.StatusUnauthorized}
	s := newServer(t, model.serve(t).URL)

	res, err := s.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].IsInstalled {
		t.Error("a refusing chat API was installed")
	}
	state, err := s.GetState(t.Context())
	if err != nil || !strings.Contains(state.Unreachable, "401") {
		t.Errorf("state %v, %v", state, err)
	}
}

func TestAskAgentStreamsTheStepsOverConnect(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("call-1", "note_read", `{"path":"Plan.md"}`), newTextReply("Done.")}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)
	mux := http.NewServeMux()
	mux.Handle(agentrun.NewHandler(s, s))
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
	if got := describeSteps(steps); got != "model call finished model said stopped" {
		t.Errorf("steps are %q", got)
	}
}

func TestAnAnswerSplitOffAtItsLengthLimitIsReportedSo(t *testing.T) {
	reply := newTextReply("The plan is to")
	model := &fakeChat{replies: []conversation.Message{reply}, finish: "length"}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != answer.ReasonCut {
		t.Errorf("steps are %q, stopped %q", got, steps[len(steps)-1].GetStopped())
	}
}

func TestACallWithNoArgumentsIsReportedBackWithAnEmptyObject(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("c", "note_read", ""), newTextReply("ok")}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)

	ask(t, s, newTask(vault.serve(t), "Read."))

	messages := model.getRequests()[1].Messages
	if args := messages[len(messages)-2].ToolCalls[0].Function.Arguments; args != "{}" {
		t.Errorf("call returned with %q", args)
	}
	if calls := vault.getCalls(); len(calls) != 1 || calls[0] != "{}" {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestAKeyIsNotSentOverPlainHTTPToAnotherMachine(t *testing.T) {
	s := newTestAgent(t)
	s.configure(map[string]string{settings.KeyBaseURL: "http://192.0.2.1/v1"})
	s.applySecrets(keySecrets("host-key"))

	steps := ask(t, s, newTask("", "Hello?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != chatapi.ErrPlainKey.Error() {
		t.Errorf("stopped %q", reason)
	}
}

func TestTheVaultTokenIsNotSentOverPlainHTTPToAnotherMachine(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("hi")}}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("http://192.0.2.1/mcp", "Hello?"))

	if len(steps) != 1 || !strings.Contains(steps[0].GetStopped(), vaulttools.ErrPlainToken.Error()) {
		t.Errorf("steps are %v", steps)
	}
}

func TestHarness_FailedListingIsRequeried(t *testing.T) {
	model := &fakeChat{status: http.StatusServiceUnavailable, models: []string{"m1"}}
	s := newServer(t, model.serve(t).URL)
	if _, err := s.catalog.ListModels(t.Context()); err == nil {
		t.Fatal("a refusing chat API listed models")
	}
	model.mu.Lock()
	model.status = 0
	model.mu.Unlock()

	if models, err := s.catalog.ListModels(t.Context()); err != nil || len(models) != 1 {
		t.Errorf("the second listing is %v, %v", models, err)
	}
}

// A task names the model it is answered by, else the configured one answers.
func TestATaskNamesTheModelThatAnswersIt(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("the answer")}}
	s := newServer(t, model.serve(t).URL)

	task := newTask("", "why?")
	task.Tools, task.ToolDeclarations, task.Model = nil, nil, "routed"
	ask(t, s, task)
	task.Model = ""
	ask(t, s, task)

	requests := model.getRequests()
	if len(requests) != 2 || requests[0].Model != "routed" || requests[1].Model != "m1" {
		t.Errorf("the models asked were %+v", requests)
	}
	if requests[0].Auth != "Bearer host-key" || len(requests[0].Messages) != 2 || requests[0].Messages[0].Role != "system" {
		t.Errorf("the first request was %+v", requests[0])
	}
}

func TestASplitOffReplyCallsNoTool(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("c", "note_read", "")}, finish: "length"}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Read."))

	if got := describeSteps(steps); got != "model stopped" || steps[1].GetStopped() != answer.ReasonCut {
		t.Errorf("steps are %q", got)
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestTheVaultTokenIsNotFollowedToPlainHTTP(t *testing.T) {
	redirecting := httptest.NewServer(http.RedirectHandler("http://192.0.2.1/mcp", http.StatusTemporaryRedirect))
	defer redirecting.Close()
	model := &fakeChat{replies: []conversation.Message{newTextReply("hi")}}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask(redirecting.URL, "Hello?"))

	if len(steps) != 1 || !strings.Contains(steps[0].GetStopped(), endpoint.ErrDowngrade.Error()) {
		t.Errorf("steps are %v", steps)
	}
}

func TestAStepThatCannotBeSentEndsTheRunWithItsError(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("c", "note_read", `{"path":"A.md"}`), newTextReply("ok")}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)
	removed := errors.New("the window closed")
	dispatched := 0

	err := s.Ask(t.Context(), newTask(vault.serve(t), "Read."), func(*agentrun.Response) error {
		dispatched++
		if dispatched == 2 {
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
func TestRunTask_NoToolsWhenTaskNamesNone(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("the answer")}}
	var connected atomic.Bool
	vault := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { connected.Store(true) }))
	t.Cleanup(vault.Close)
	s := newServer(t, model.serve(t).URL)
	task := newTask(vault.URL, "why?")
	task.Tools, task.ToolDeclarations = nil, nil

	steps := ask(t, s, task)

	if got := strings.Join(getSpokenTexts(steps), ""); got != "the answer" {
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

// refusingTransport fails the test when a request is sent.
type refusingTransport struct{ t *testing.T }

func (r refusingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.t.Errorf("a request was sent to %s", req.URL)
	return nil, errors.New("no network in this test")
}

func TestARemoteChatAPIWithNoKeyStopsTheRunBeforeAnyRequest(t *testing.T) {
	for _, base := range []string{"", "https://api.example.com/v1"} {
		s := newTestAgent(t, chatapi.WithHTTPClient(&http.Client{Transport: refusingTransport{t}}))
		s.configure(map[string]string{settings.KeyBaseURL: base})

		steps := ask(t, s, newTask("", "Hello?"))

		if reason := steps[len(steps)-1].GetStopped(); reason != "API key is not set: add it in Settings → Keys" {
			t.Errorf("base %q stopped %q", base, reason)
		}
	}
}

func TestHarness_LocalChatAPIWithoutKeyIsCalled(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newTextReply("hi")}}
	s := newTestAgent(t)
	s.configure(map[string]string{settings.KeyBaseURL: model.serve(t).URL})

	steps := ask(t, s, newTask("", "Hello?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != answer.ReasonDone {
		t.Errorf("stopped %q", reason)
	}
}

func TestARefusedKeyIsReportedWithWhereToSetIt(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		model := &fakeChat{status: status}
		s := newServer(t, model.serve(t).URL)

		steps := ask(t, s, newTask("", "Hello?"))

		reason := steps[len(steps)-1].GetStopped()
		if !strings.Contains(reason, strconv.Itoa(status)) || !strings.HasSuffix(reason, "(check the API key in Settings → Keys)") {
			t.Errorf("status %d stopped %q", status, reason)
		}
	}
}

func TestAnAnswerOtherThanARefusedKeyAddsNoHint(t *testing.T) {
	model := &fakeChat{status: http.StatusInternalServerError}
	s := newServer(t, model.serve(t).URL)

	steps := ask(t, s, newTask("", "Hello?"))

	if reason := steps[len(steps)-1].GetStopped(); strings.Contains(reason, "Settings") {
		t.Errorf("stopped %q", reason)
	}
}

func TestGetModels_ReasoningParameterYieldsLevels(t *testing.T) {
	model := &fakeChat{
		models:     []string{"thinker", "router", "plain"},
		parameters: map[string][]string{"thinker": {"temperature", "reasoning_effort"}, "router": {"reasoning"}},
	}
	s := newServer(t, model.serve(t).URL)

	res, err := s.GetPrograms(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var proposed []int
	for _, one := range res[0].Models {
		proposed = append(proposed, len(one.ReasoningLevels))
	}
	if !slices.Equal(proposed, []int{3, 3, 0, 0}) {
		t.Errorf("the levels offered are %v", proposed)
	}
}

func TestTheLevelOfATaskGoesInTheFieldTheModelListsOrNowhere(t *testing.T) {
	cases := []struct {
		model, level string
		effort       string
		reasoning    string
	}{
		{"thinker", "high", "high", ""},
		{"router", "low", "", "low"},
		{"plain", "high", "", ""},
		{"thinker", "extreme", "", ""},
		{"thinker", "", "", ""},
		{"unlisted", "high", "", ""},
	}
	for _, one := range cases {
		model := &fakeChat{
			replies:    []conversation.Message{newTextReply("hi")},
			models:     []string{"thinker", "router", "plain"},
			parameters: map[string][]string{"thinker": {"reasoning_effort"}, "router": {"reasoning"}},
		}
		s := newServer(t, model.serve(t).URL)
		task := newTask("", "Hello?")
		task.Model, task.Reasoning = one.model, one.level
		task.Tools, task.ToolsURL = nil, ""

		ask(t, s, task)

		got := model.getRequests()[0]
		gotReasoning := ""
		if got.Reasoning != nil {
			gotReasoning = got.Reasoning.Effort
		}
		if got.ReasoningEffort != one.effort || gotReasoning != one.reasoning {
			t.Errorf("model %q at %q was sent %q and %q, want %q and %q",
				one.model, one.level, got.ReasoningEffort, gotReasoning, one.effort, one.reasoning)
		}
	}
}

// Any MCP tool from any connected MCP server can be called without requiring or stripping prefixes.
func TestAnyMCPToolWithoutPrefixCanBeCalled(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("call-custom", "custom_remote_tool", `{"query":"test"}`), newTextReply("Result received.")}}
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

	s := newServer(t, model.serve(t).URL)
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
	if response := strings.Join(getSpokenTexts(steps), ""); response != "Result received." {
		t.Errorf("the answer is %q, want %q", response, "Result received.")
	}
}

// The context of the task is passed to every tool the model calls as the `_meta` of
// the call.
func TestTheContextOfATaskIsTheMetaOfEveryCall(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("call-1", "note_read", `{"path":"Plan.md"}`), newTextReply("ok")}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)
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
	model := &fakeChat{replies: []conversation.Message{newCallReply("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)
	s.configure(map[string]string{settings.KeyMaxSteps: "3"})

	steps := ask(t, s, newTask(vault.serve(t), "Read forever."))

	if steps[len(steps)-1].GetStopped() != answer.ReasonTurns {
		t.Errorf("stopped %q", steps[len(steps)-1].GetStopped())
	}
	if len(model.getRequests()) != 3 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}

func TestATaskLimitOverridesTheMaxStepsSetting(t *testing.T) {
	model := &fakeChat{replies: []conversation.Message{newCallReply("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(t, model.serve(t).URL)
	s.configure(map[string]string{settings.KeyMaxSteps: "5"})
	task := newTask(vault.serve(t), "Read forever.")
	task.MaxSteps = 2

	ask(t, s, task)

	if len(model.getRequests()) != 2 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}
