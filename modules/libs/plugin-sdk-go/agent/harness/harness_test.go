package harness

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

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

const testToken = "vault-token"

// sentRequest is what the fake chat API was sent.
type sentRequest struct {
	Auth     string
	Model    string         `json:"model"`
	Messages []message      `json:"messages"`
	Tools    []functionTool `json:"tools"`
	Stream   bool           `json:"stream"`
}

// fakeChat is an OpenAI-compatible chat API that answers each request with the next reply it holds,
// repeating the last, and keeps every request, refused ones too. A streaming request is answered
// with the reply in pieces: its text word by word, and each tool call in three fragments. script,
// where it is set, is the data of the events that answer the first request. refusesStream answers
// a streaming request 400 Bad Request, and ignoresStream answers it with the whole reply.
type fakeChat struct {
	mu            sync.Mutex
	replies       []message
	script        []string
	status        int
	finish        string
	models        []string
	refusesStream bool
	ignoresStream bool
	requests      []sentRequest
	answered      int
}

func (f *fakeChat) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.status != 0 {
			http.Error(w, "invalid key", f.status)
			return
		}
		if r.URL.Path == "/models" {
			var listing struct {
				Data []map[string]string `json:"data"`
			}
			for _, one := range f.models {
				listing.Data = append(listing.Data, map[string]string{"id": one, "name": strings.ToUpper(one)})
			}
			_ = json.NewEncoder(w).Encode(listing)
			return
		}
		var got sentRequest
		_ = json.NewDecoder(r.Body).Decode(&got)
		got.Auth = r.Header.Get("Authorization")
		f.requests = append(f.requests, got)
		if got.Stream && f.refusesStream {
			http.Error(w, `{"error":{"message":"stream is not supported"}}`, http.StatusBadRequest)
			return
		}
		f.answered++
		if got.Stream && f.script != nil && f.answered == 1 {
			writeEvents(w, f.script)
			return
		}
		reply := f.replies[min(f.answered, len(f.replies))-1]
		if got.Stream && !f.ignoresStream {
			writeEvents(w, f.getPieces(reply))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []map[string]any{{"message": reply, "finish_reason": f.finish}}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// getPieces is the data of the events that stream the reply, ending with [DONE].
func (f *fakeChat) getPieces(reply message) []string {
	var events []string
	addDelta := func(delta map[string]any, finish any) {
		event, _ := json.Marshal(map[string]any{"choices": []map[string]any{{"delta": delta, "finish_reason": finish}}})
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

// writeEvents writes each data as one server-sent event, flushing after each.
func writeEvents(w http.ResponseWriter, data []string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, one := range data {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", one)
		w.(http.Flusher).Flush()
	}
}

func (f *fakeChat) getRequests() []sentRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentRequest(nil), f.requests...)
}

// newCallReply is a reply of the model that calls the tool with the arguments.
func newCallReply(id, tool, args string) message {
	call := toolCall{ID: id, Type: "function"}
	call.Function.Name, call.Function.Arguments = tool, args
	return message{Role: "assistant", ToolCalls: []toolCall{call}}
}

// newTextReply is a reply of the model that says the text.
func newTextReply(text string) message {
	return message{Role: "assistant", Content: newText(text)}
}

// fakeVault is a tool server with two tools, behind the token, that keeps the arguments of every
// call.
type fakeVault struct {
	mu    sync.Mutex
	calls []string
}

func (v *fakeVault) serve(t *testing.T) string {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "numen", Version: "1"}, nil)
	schema := map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}}
	server.AddTool(&mcp.Tool{Name: "note_read", Description: "Reads a note", InputSchema: schema},
		func(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			v.mu.Lock()
			v.calls = append(v.calls, string(req.Params.Arguments))
			v.mu.Unlock()
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "Plan: water the garden."}}}, nil
		})
	server.AddTool(&mcp.Tool{Name: "note_delete", Description: "Deletes a note", InputSchema: schema},
		func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			t.Error("a tool the task did not name was called")
			return &mcp.CallToolResult{}, nil
		})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+testToken {
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

// newTask is a task that may call note_read, on the vault at url.
func newTask(url, prompt string) *numenv1.AskAgentRequest {
	return &numenv1.AskAgentRequest{
		Prompt:           prompt,
		Instructions:     "Answer from the vault.",
		Tools:            []string{"note_read"},
		ToolDeclarations: []*numenv1.ToolDeclaration{{Name: "note_read", Title: "Read a note", Kind: "read", About: "path"}},
		ToolsUrl:         url,
		ToolsToken:       testToken,
	}
}

func ask(t *testing.T, s *Server, task *numenv1.AskAgentRequest) []*numenv1.AskAgentResponse {
	t.Helper()
	var steps []*numenv1.AskAgentResponse
	if err := s.Ask(t.Context(), task, func(r *numenv1.AskAgentResponse) error {
		steps = append(steps, r)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return steps
}

// describeSteps writes the steps out one word each, a run of pieces of what was said as one.
func describeSteps(steps []*numenv1.AskAgentResponse) string {
	words := make([]string, 0, len(steps))
	for _, one := range steps {
		switch {
		case one.GetModelStarted() != nil:
			words = append(words, "model")
		case one.GetToolCall() != nil:
			words = append(words, "call")
		case one.GetToolFinished() != nil:
			words = append(words, "finished")
		case one.GetSaid() != "":
			if len(words) == 0 || words[len(words)-1] != "said" {
				words = append(words, "said")
			}
		default:
			words = append(words, "stopped")
		}
	}
	return strings.Join(words, " ")
}

// getSaid is every piece of what was said, in the order it was sent.
func getSaid(steps []*numenv1.AskAgentResponse) []string {
	var said []string
	for _, one := range steps {
		if text := one.GetSaid(); text != "" {
			said = append(said, text)
		}
	}
	return said
}

func newServer(chatURL string) *Server {
	s := &Server{}
	s.Configure(map[string]string{SettingBaseURL: chatURL, SettingAPIKey: "host-key", SettingModel: "m1"})
	return s
}

func TestAToolTheModelCallsRunsAndItsResultIsFedBack(t *testing.T) {
	model := &fakeChat{replies: []message{newCallReply("call-1", "note_read", `{"path":"Plan.md"}`), newTextReply("Water the garden.")}}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "What is the plan?"))

	if got := describeSteps(steps); got != "model call finished model said stopped" {
		t.Fatalf("steps are %q", got)
	}
	call := steps[1].GetToolCall()
	if call.GetTool() != "Read a note" || call.GetKind() != "read" || call.GetCall() != "call-1" || call.GetPath() != "Plan.md" {
		t.Errorf("the call is %+v", call)
	}
	if said, last := strings.Join(getSaid(steps), ""), steps[len(steps)-1]; said != "Water the garden." || last.GetStopped() != reasonDone {
		t.Errorf("the answer is %q, stopped %q", said, last.GetStopped())
	}
	if calls := vault.getCalls(); len(calls) != 1 || calls[0] != `{"path":"Plan.md"}` {
		t.Errorf("the vault was called with %v", calls)
	}

	requests := model.getRequests()
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
		t.Errorf("the model was told %+v", fedBack)
	}
}

func TestAModelThatKeepsCallingToolsStopsAtTheTurnLimit(t *testing.T) {
	model := &fakeChat{replies: []message{newCallReply("c", "note_read", `{"path":"A.md"}`)}}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)
	task := newTask(vault.serve(t), "Read forever.")
	task.MaxSteps = 2

	steps := ask(t, s, task)

	if got := describeSteps(steps); got != "model call finished model call finished stopped" {
		t.Fatalf("steps are %q", got)
	}
	if steps[len(steps)-1].GetStopped() != reasonTurns {
		t.Errorf("stopped %q", steps[len(steps)-1].GetStopped())
	}
	if len(model.getRequests()) != 2 {
		t.Errorf("the model was asked %d times", len(model.getRequests()))
	}
}

func TestAFailureOfTheChatAPIStopsTheRunWithItsReason(t *testing.T) {
	model := &fakeChat{status: http.StatusUnauthorized}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Hello?"))

	if got := describeSteps(steps); got != "model stopped" {
		t.Fatalf("steps are %q", got)
	}
	if reason := steps[1].GetStopped(); !strings.Contains(reason, "401") || !strings.Contains(reason, "invalid key") {
		t.Errorf("stopped %q", reason)
	}
}

func TestAVaultThatCannotBeReachedStopsTheRunBeforeTheModelIsAsked(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("hi")}}
	s := newServer(model.serve(t).URL)
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

func TestTheKeyIsTheOneConfigureHandsOverAndNoneIsSentWithoutOne(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	model := &fakeChat{replies: []message{newTextReply("hi")}}
	s := &Server{}
	s.Configure(map[string]string{SettingBaseURL: model.serve(t).URL})

	ask(t, s, newTask("", "Hello?"))
	s.Configure(map[string]string{SettingAPIKey: "host-key"})
	ask(t, s, newTask("", "Hello?"))
	s.Configure(map[string]string{SettingAPIKey: ""})
	ask(t, s, newTask("", "Hello?"))

	requests := model.getRequests()
	if requests[0].Auth != "" || requests[1].Auth != "Bearer host-key" || requests[2].Auth != "" {
		t.Errorf("authorization was %q, %q, %q", requests[0].Auth, requests[1].Auth, requests[2].Auth)
	}
	if requests[0].Model != defaultModel {
		t.Errorf("the model asked was %q", requests[0].Model)
	}
}

func TestAConversationGoesOnFromWhatWasSaidUntilItIsFinished(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("Noted.")}}
	s := newServer(model.serve(t).URL)
	task := newTask("", "Remember the garden.")
	task.Conversation = "talk-1"

	ask(t, s, task)
	task.Prompt = "What did I ask?"
	ask(t, s, task)
	if _, err := s.FinishConversation(t.Context(), connect.NewRequest(&numenv1.FinishConversationRequest{Conversation: "talk-1"})); err != nil {
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
	s := newServer(model.serve(t).URL)

	res, err := s.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	program := res.Msg.GetPrograms()[0]
	if program.GetName() != ProgramName || !program.GetIsInstalled() || !program.GetLimitsTools() || len(program.GetModels()) != 2 {
		t.Fatalf("the program is %+v", program)
	}
	if program.GetModels()[0].GetIsDefault() || !program.GetModels()[1].GetIsDefault() || program.GetModels()[1].GetTitle() != "M1" {
		t.Errorf("the models are %+v", program.GetModels())
	}
	state, err := s.GetAgentState(t.Context(), connect.NewRequest(&numenv1.GetAgentStateRequest{}))
	if err != nil || state.Msg.GetUnreachable() != "" {
		t.Errorf("state %v, %v", state, err)
	}
}

func TestTheProgramIsNotInstalledWhileTheChatAPIRefuses(t *testing.T) {
	model := &fakeChat{status: http.StatusUnauthorized}
	s := newServer(model.serve(t).URL)

	res, err := s.GetAgentPrograms(t.Context(), connect.NewRequest(&numenv1.GetAgentProgramsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetPrograms()[0].GetIsInstalled() {
		t.Error("a refusing chat API was installed")
	}
	state, err := s.GetAgentState(t.Context(), connect.NewRequest(&numenv1.GetAgentStateRequest{}))
	if err != nil || !strings.Contains(state.Msg.GetUnreachable(), "401") {
		t.Errorf("state %v, %v", state, err)
	}
}

func TestAskAgentStreamsTheStepsOverConnect(t *testing.T) {
	model := &fakeChat{replies: []message{newCallReply("call-1", "note_read", `{"path":"Plan.md"}`), newTextReply("Done.")}}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewAgentServiceHandler(s))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stream, err := numenv1connect.NewAgentServiceClient(srv.Client(), srv.URL).AskAgent(t.Context(), connect.NewRequest(newTask(vault.serve(t), "Plan?")))
	if err != nil {
		t.Fatal(err)
	}
	var steps []*numenv1.AskAgentResponse
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

func TestAnAnswerCutOffAtItsLengthLimitIsReportedSo(t *testing.T) {
	cut := newTextReply("The plan is to")
	model := &fakeChat{replies: []message{cut}, finish: "length"}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("", "Plan?"))

	if got := describeSteps(steps); got != "model said stopped" || steps[len(steps)-1].GetStopped() != reasonCut {
		t.Errorf("steps are %q, stopped %q", got, steps[len(steps)-1].GetStopped())
	}
}

func TestACallWithNoArgumentsIsToldBackWithAnEmptyObject(t *testing.T) {
	model := &fakeChat{replies: []message{newCallReply("c", "note_read", ""), newTextReply("ok")}}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)

	ask(t, s, newTask(vault.serve(t), "Read."))

	told := model.getRequests()[1].Messages
	if args := told[len(told)-2].ToolCalls[0].Function.Arguments; args != "{}" {
		t.Errorf("the call was told back with %q", args)
	}
	if calls := vault.getCalls(); len(calls) != 1 || calls[0] != "{}" {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestAKeyIsNotSentOverPlainHTTPToAnotherMachine(t *testing.T) {
	s := &Server{}
	s.Configure(map[string]string{SettingBaseURL: "http://192.0.2.1/v1", SettingAPIKey: "host-key"})

	steps := ask(t, s, newTask("", "Hello?"))

	if reason := steps[len(steps)-1].GetStopped(); reason != errPlainKey.Error() {
		t.Errorf("stopped %q", reason)
	}
}

func TestAKeyIsNotFollowedToPlainHTTP(t *testing.T) {
	err := refuseDowngrade(httptest.NewRequest(http.MethodGet, "http://192.0.2.1/v1/models", nil), nil)
	if err == nil {
		t.Error("a redirect to plain http was followed")
	}
	if err := refuseDowngrade(httptest.NewRequest(http.MethodGet, "https://example.com/v1/models", nil), nil); err != nil {
		t.Errorf("a redirect to https was refused: %v", err)
	}
}

func TestTheVaultTokenIsNotSentOverPlainHTTPToAnotherMachine(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("hi")}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask("http://192.0.2.1/mcp", "Hello?"))

	if len(steps) != 1 || !strings.Contains(steps[0].GetStopped(), errPlainToken.Error()) {
		t.Errorf("steps are %v", steps)
	}
}

func TestAFailedListingIsAskedAgain(t *testing.T) {
	model := &fakeChat{status: http.StatusServiceUnavailable, models: []string{"m1"}}
	s := newServer(model.serve(t).URL)
	if _, err := s.listModels(t.Context()); err == nil {
		t.Fatal("a refusing chat API listed models")
	}
	model.mu.Lock()
	model.status = 0
	model.mu.Unlock()

	if models, err := s.listModels(t.Context()); err != nil || len(models) != 1 {
		t.Errorf("the second listing is %v, %v", models, err)
	}
}

// A task names the model it is answered by, else the configured one answers.
func TestATaskNamesTheModelThatAnswersIt(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("the answer")}}
	s := newServer(model.serve(t).URL)

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

func TestACutOffReplyCallsNoTool(t *testing.T) {
	model := &fakeChat{replies: []message{newCallReply("c", "note_read", "")}, finish: "length"}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask(vault.serve(t), "Read."))

	if got := describeSteps(steps); got != "model stopped" || steps[1].GetStopped() != reasonCut {
		t.Errorf("steps are %q", got)
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("the vault was called with %v", calls)
	}
}

func TestTheVaultTokenIsNotFollowedToPlainHTTP(t *testing.T) {
	redirecting := httptest.NewServer(http.RedirectHandler("http://192.0.2.1/mcp", http.StatusTemporaryRedirect))
	defer redirecting.Close()
	model := &fakeChat{replies: []message{newTextReply("hi")}}
	s := newServer(model.serve(t).URL)

	steps := ask(t, s, newTask(redirecting.URL, "Hello?"))

	if len(steps) != 1 || !strings.Contains(steps[0].GetStopped(), errDowngrade.Error()) {
		t.Errorf("steps are %v", steps)
	}
}

func TestAStepThatCannotBeSentEndsTheRunWithItsError(t *testing.T) {
	model := &fakeChat{replies: []message{newCallReply("c", "note_read", `{"path":"A.md"}`), newTextReply("ok")}}
	vault := &fakeVault{}
	s := newServer(model.serve(t).URL)
	gone := errors.New("the window closed")
	sent := 0

	err := s.Ask(t.Context(), newTask(vault.serve(t), "Read."), func(*numenv1.AskAgentResponse) error {
		sent++
		if sent == 2 {
			return gone
		}
		return nil
	})

	if !errors.Is(err, gone) {
		t.Errorf("the run ended with %v", err)
	}
	if calls := vault.getCalls(); len(calls) != 0 {
		t.Errorf("the vault was called after the call could not be told: %v", calls)
	}
}

// A task that names no tool is one answer: the model is offered no tool, and the vault is not
// connected to, whatever endpoint the task holds.
func TestATaskNamingNoToolOffersTheModelNoTool(t *testing.T) {
	model := &fakeChat{replies: []message{newTextReply("the answer")}}
	var connected atomic.Bool
	vault := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { connected.Store(true) }))
	t.Cleanup(vault.Close)
	s := newServer(model.serve(t).URL)
	task := newTask(vault.URL, "why?")
	task.Tools, task.ToolDeclarations = nil, nil

	steps := ask(t, s, task)

	if got := strings.Join(getSaid(steps), ""); got != "the answer" {
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
