package service

import (
	"bytes"
	"context"
	"errors"

	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/acp/acptest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools/toolstest"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/conversation"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/internal/adapter/hosttools"
	"github.com/jiva-studio/numen/plugins/agents-numen/backend/ports"
)

const hostToken = "host-token"

// turn is what the fake model answers to one request: the text it says and the calls it asks for.
type turn struct {
	text  string
	calls []conversation.ToolCall
}

func newTextTurn(text string) turn {
	return turn{text: text}
}

func newCallTurn(id, tool, path string) turn {
	call := conversation.ToolCall{ID: id, Name: tool, Arguments: `{"path":"` + path + `"}`, About: path, Path: path}
	return turn{text: "Let me look. ", calls: []conversation.ToolCall{call}}
}

// fakeModel answers each request with the next turn it holds, and the last turn again after that. It
// keeps the requests it received.
type fakeModel struct {
	mu       sync.Mutex
	turns    []turn
	requests []conversation.Turn
}

func (f *fakeModel) Stream(_ context.Context, request conversation.Turn, emit func(string) error) (conversation.Reply, error) {
	f.mu.Lock()
	f.requests = append(f.requests, request)
	one := f.turns[min(len(f.requests), len(f.turns))-1]
	f.mu.Unlock()
	if err := emit(one.text); err != nil {
		return conversation.Reply{}, err
	}
	return conversation.Reply{
		Message: conversation.Message{Role: "assistant", Text: one.text, ToolCalls: append([]conversation.ToolCall(nil), one.calls...)},
	}, nil
}

func (f *fakeModel) getRequests() []conversation.Turn {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]conversation.Turn(nil), f.requests...)
}

// fakeHost serves the tool service of the host behind its token, with one tool that reads a note, and
// keeps every call.
type fakeHost struct {
	mu    sync.Mutex
	calls []tools.Call
}

func (h *fakeHost) ListTools(context.Context, http.Header) (tools.Listing, error) {
	return tools.Listing{Tools: []*tools.ToolInfo{{
		Name:         "note_read",
		Title:        "Read a note",
		Description:  "Reads a note",
		InputSchema:  []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
		Effect:       tools.ReadOnly,
		Instructions: []string{"Cite the note you read."},
	}}}, nil
}

func (h *fakeHost) InvokeTool(_ context.Context, _ http.Header, call tools.Call) (*tools.CallResponse, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, call)
	return &tools.CallResponse{Result: &tools.CallResult{Text: "Plan: water the garden."}}, nil
}

func (h *fakeHost) getCalls() []tools.Call {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]tools.Call(nil), h.calls...)
}

func (h *fakeHost) serve(t *testing.T) string {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(toolstest.NewServiceHandler(h))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+hostToken {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// chatConnections are two connections that serve llm.chat: Meadow and Harbor, each with its models.
type chatConnections struct{}

func (chatConnections) ListConnections(context.Context) ([]conversation.ChatConnection, error) {
	return []conversation.ChatConnection{
		{ID: "c-meadow", Name: "Meadow", ModelField: "model"},
		{ID: "c-harbor", Name: "Harbor", ModelField: "model"},
	}, nil
}

func (chatConnections) ListModels(_ context.Context, chat conversation.ChatConnection) ([]conversation.ModelOption, error) {
	if chat.ID == "c-meadow" {
		return []conversation.ModelOption{{ID: "wren-2", Title: "Wren 2"}}, nil
	}
	return []conversation.ModelOption{{ID: "gull-7b", Title: "Gull 7B"}}, nil
}

func newTestAgent(t *testing.T, model ports.Chat, hostURL string) *agent {
	t.Helper()
	s, err := newAgent(host{
		chat: model, models: chatConnections{},
		tools: hosttools.New(tools.NewClient(hostcall.Endpoint{BaseURL: hostURL, Token: hostToken})),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// gull is the row of a task that chose the model Gull 7B of Harbor.
var gull = connection.Call{Type: "numen", Options: map[string]string{"model": "c-harbor|gull-7b"}}

func runTask(ctx context.Context, t *testing.T, s *agent, task *acp.TaskRequest) ([]*acp.TaskFrame, acp.StopReason, error) {
	t.Helper()
	var frames []*acp.TaskFrame
	done, err := acptest.RunTask(ctx, s.provider, gull, task, func(frame *acp.TaskFrame) error {
		frames = append(frames, frame)
		return nil
	})
	if err != nil {
		return frames, acp.StopReason_STOP_REASON_UNSPECIFIED, err
	}
	return frames, done.GetStopReason(), nil
}

func describeFrames(frames []*acp.TaskFrame) string {
	words := make([]string, 0, len(frames))
	for _, one := range frames {
		switch {
		case one.GetAgentMessageChunk() != nil:
			words = append(words, "said "+one.GetAgentMessageChunk().GetText())
		case one.GetToolCall() != nil:
			words = append(words, "call")
		case one.GetToolCallUpdate() != nil:
			words = append(words, "update "+one.GetToolCallUpdate().GetStatus().String())
		}
	}
	return strings.Join(words, " | ")
}

func TestATurnWithAToolCallSendsTheTextTheCallAndItsEnd(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("call-1", "note_read", "Plan.md"), newTextTurn("Water the garden.")}}
	host := &fakeHost{}
	s := newTestAgent(t, model, host.serve(t))

	frames, reason, err := runTask(t.Context(), t, s, &acp.TaskRequest{
		Prompt: "What is the plan?", ToolContext: map[string]string{"md.example.cards/card": "k7"}, VaultId: "v-garden",
	})

	if err != nil || reason != acp.StopReason_STOP_REASON_END_TURN {
		t.Fatalf("the task ended with %v, %v", reason, err)
	}
	if got := describeFrames(frames); got != "said Let me look.  | call | update TOOL_CALL_STATUS_COMPLETED | said Water the garden." {
		t.Fatalf("the frames are %q", got)
	}
	call := frames[1].GetToolCall()
	if call.GetToolCallId() != "call-1" || call.GetTitle() != "Read a note" || call.GetKind() != acp.ToolKind_TOOL_KIND_READ ||
		call.GetLocations()[0].GetPath() != "Plan.md" || call.GetRawInputJson() != `{"path":"Plan.md"}` {
		t.Errorf("the call is %v", call)
	}
	calls := host.getCalls()
	if len(calls) != 1 || calls[0].Name != "note_read" || string(calls[0].Arguments) != `{"path":"Plan.md"}` ||
		calls[0].VaultID != "v-garden" || calls[0].Context["md.example.cards/card"] != "k7" {
		t.Errorf("the host was called with %+v", calls)
	}
	requests := model.getRequests()
	if len(requests) != 2 {
		t.Fatalf("the model was asked %d times", len(requests))
	}
	first := requests[0]
	if len(first.Tools) != 1 || first.Tools[0].Name != "note_read" || !strings.Contains(string(first.Tools[0].Parameters), `"path"`) {
		t.Errorf("the model was offered %+v", first.Tools)
	}
	if first.Messages[0].Role != "system" || first.Messages[0].Text != "Cite the note you read." || first.Messages[1].Text != "What is the plan?" {
		t.Errorf("the first request is %+v", first.Messages)
	}
	if fedBack := requests[1].Messages[len(requests[1].Messages)-1]; fedBack.Role != "tool" || fedBack.ToolCallID != "call-1" || fedBack.Text != "Plan: water the garden." {
		t.Errorf("the model received %+v", fedBack)
	}
}

func TestTheTurnsOfAConversationContinueItsHistory(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("Noted.")}}
	host := &fakeHost{}
	s := newTestAgent(t, model, host.serve(t))
	task := &acp.TaskRequest{Prompt: "Remember the garden.", Conversation: "talk-1"}

	if _, _, err := runTask(t.Context(), t, s, task); err != nil {
		t.Fatal(err)
	}
	task.Prompt = "What did I ask?"
	if _, _, err := runTask(t.Context(), t, s, task); err != nil {
		t.Fatal(err)
	}

	second := model.getRequests()[1].Messages
	if len(second) != 4 || second[1].Text != "Remember the garden." || second[2].Text != "Noted." || second[3].Text != "What did I ask?" {
		t.Errorf("the second request was %+v", second)
	}
}

func TestAModelThatKeepsCallingToolsStopsAtTheStepLimit(t *testing.T) {
	model := &fakeModel{turns: []turn{newCallTurn("c", "note_read", "A.md")}}
	host := &fakeHost{}
	s := newTestAgent(t, model, host.serve(t))
	s.settings.Configure(map[string]string{"maxSteps": "3"})

	_, reason, err := runTask(t.Context(), t, s, &acp.TaskRequest{Prompt: "Read forever."})

	if err != nil || reason != acp.StopReason_STOP_REASON_MAX_TURN_REQUESTS {
		t.Errorf("the task ended with %v, %v", reason, err)
	}
	if got := len(model.getRequests()); got != 3 {
		t.Errorf("the model was asked %d times", got)
	}
}

func TestTheOptionsOfTheSettingsGoWithEveryRequest(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	host := &fakeHost{}
	s := newTestAgent(t, model, host.serve(t))
	s.settings.Configure(map[string]string{"temperature": "0.5", "maxTokens": "2048"})

	if _, _, err := runTask(t.Context(), t, s, &acp.TaskRequest{Prompt: "Hello?"}); err != nil {
		t.Fatal(err)
	}

	request := model.getRequests()[0]
	if request.Temperature == nil || *request.Temperature != 0.5 || request.MaxTokens != 2048 {
		t.Errorf("temperature = %v, max tokens = %d", request.Temperature, request.MaxTokens)
	}
}

func TestAHostThatCannotListItsToolsFailsTheTaskBeforeTheModelIsAsked(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("hi")}}
	nowhere := httptest.NewServer(http.NotFoundHandler())
	defer nowhere.Close()
	s := newTestAgent(t, model, nowhere.URL)

	_, _, err := runTask(t.Context(), t, s, &acp.TaskRequest{Prompt: "Hello?"})

	if err == nil || len(model.getRequests()) != 0 {
		t.Errorf("the task ended with %v after %d requests", err, len(model.getRequests()))
	}
}

// blockingModel says one word and then waits for the context of its request.
type blockingModel struct {
	ended chan struct{}
}

func (m *blockingModel) Stream(ctx context.Context, _ conversation.Turn, emit func(string) error) (conversation.Reply, error) {
	if err := emit("partial"); err != nil {
		return conversation.Reply{}, err
	}
	<-ctx.Done()
	close(m.ended)
	return conversation.Reply{}, ctx.Err()
}

func startBlockedTask(ctx context.Context, t *testing.T) (*agent, *blockingModel, chan error) {
	t.Helper()
	model := &blockingModel{ended: make(chan struct{})}
	host := &fakeHost{}
	s := newTestAgent(t, model, host.serve(t))
	finish := make(chan error, 1)
	firstText := make(chan struct{}, 1)
	go func() {
		_, err := acptest.RunTask(ctx, s.provider, gull, &acp.TaskRequest{Prompt: "Hello?"}, func(*acp.TaskFrame) error {
			firstText <- struct{}{}
			return nil
		})
		finish <- err
	}()
	<-firstText
	return s, model, finish
}

func TestACanceledTaskEndsTheTurnOfTheModel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	_, model, finish := startBlockedTask(ctx, t)

	cancel()

	if err := <-finish; !errors.Is(err, context.Canceled) {
		t.Errorf("a canceled task returned %v", err)
	}
	<-model.ended
}

func TestClosingThePluginEndsARunningTask(t *testing.T) {
	s, model, finish := startBlockedTask(t.Context(), t)

	if err := s.answerer.Close(); err != nil {
		t.Fatal(err)
	}

	if err := <-finish; !errors.Is(err, context.Canceled) {
		t.Errorf("a task of a closed plugin returned %v", err)
	}
	<-model.ended
}

func TestATaskChatsWithTheModelItsRowChose(t *testing.T) {
	model := &fakeModel{turns: []turn{newTextTurn("Noted.")}}
	host := &fakeHost{}
	s := newTestAgent(t, model, host.serve(t))

	if _, _, err := runTask(t.Context(), t, s, &acp.TaskRequest{Prompt: "Hello?"}); err != nil {
		t.Fatal(err)
	}

	if got := model.getRequests()[0].Model; got != (conversation.Model{ConnectionID: "c-harbor", Field: "model", ID: "gull-7b"}) {
		t.Errorf("the model asked is %+v", got)
	}
}

func TestTheModelsOfTheAgentAreTheModelsOfEveryChatConnection(t *testing.T) {
	s := newTestAgent(t, &fakeModel{turns: []turn{newTextTurn("hi")}}, (&fakeHost{}).serve(t))

	done, err := s.provider[llm.OperationModels](t.Context(), connection.Call{Type: "numen"}, nil, func([]byte) error { return nil })

	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"c-meadow|wren-2", "Meadow / Wren 2", "c-harbor|gull-7b", "Harbor / Gull 7B"} {
		if !bytes.Contains(done, []byte(want)) {
			t.Errorf("the listing %q holds no %q", done, want)
		}
	}
	if bytes.Index(done, []byte("Meadow / Wren 2")) > bytes.Index(done, []byte("Harbor / Gull 7B")) {
		t.Errorf("the listing %q is not in the order of the connections", done)
	}
}
