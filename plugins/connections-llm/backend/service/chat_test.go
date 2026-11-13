package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func chatUser(text string) *llm.ChatRequest {
	return &llm.ChatRequest{Messages: []*llm.ChatMessage{{Role: "user", Text: text}}}
}

// toolCall is a tool call of a frame, comparable as a value.
type toolCall struct{ ID, Name, Arguments string }

// chatEvent is one frame of a turn, or its finish, which is the last event of a turn that ended.
type chatEvent struct {
	TextDelta string
	ToolCall  *toolCall
	Finish    *llm.ChatFinish
}

func runChat(t *testing.T, svc *testService, call connection.Call, request *llm.ChatRequest) ([]chatEvent, error) {
	t.Helper()
	var events []chatEvent
	finish, err := svc.Chat(newContext(t), call, request, func(frame *llm.ChatFrame) error {
		event := chatEvent{TextDelta: frame.GetTextDelta()}
		if one := frame.GetToolCall(); one != nil {
			event.ToolCall = &toolCall{ID: one.GetId(), Name: one.GetName(), Arguments: one.GetArgumentsJson()}
		}
		events = append(events, event)
		return nil
	})
	if err == nil {
		events = append(events, chatEvent{Finish: finish})
	}
	return events, err
}

func ignoreFrame(*llm.ChatFrame) error { return nil }

func joinText(events []chatEvent) string {
	var text strings.Builder
	for _, one := range events {
		text.WriteString(one.TextDelta)
	}
	return text.String()
}

func lastChatBody(t *testing.T, fake *fakeopenai.Server) map[string]any {
	t.Helper()
	requests := fake.Requests()
	var body map[string]any
	if err := json.Unmarshal(requests[len(requests)-1].Body, &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestChatStreamsTheTextAsDeltasAndEndsWithAFinish(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})

	events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "openai/gpt-4o-mini"), chatUser("hello there"))

	if err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if joinText(events) != "echo: hello there" || len(events) < 3 || last.Finish == nil || last.Finish.GetReason() != llm.FinishReason_FINISH_REASON_STOP {
		t.Errorf("events = %+v", events)
	}
	for _, one := range events[:len(events)-1] {
		if one.TextDelta == "" {
			t.Errorf("an event before the finish is not a text delta: %+v", one)
		}
	}
	requests := fake.Requests()
	if len(requests) != 1 || requests[0].Path != "/chat/completions" || requests[0].Header.Get("Authorization") != "Bearer "+testKey {
		t.Errorf("requests = %+v", requests)
	}
	if body := lastChatBody(t, fake); body["stream"] != true || body["model"] != "openai/gpt-4o-mini" {
		t.Errorf("body = %v", body)
	}
}

func TestChatSendsTheHistoryTheToolsAndTheOptions(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	temperature := 0.3
	request := &llm.ChatRequest{
		Messages: []*llm.ChatMessage{
			{Role: "system", Text: "You are an agent."},
			{Role: "user", Text: "list the notes"},
			{Role: "assistant", ToolCalls: []*llm.ToolCall{{Id: "call-1", Name: "list_notes", ArgumentsJson: `{"limit":3}`}}},
			{Role: "tool", Text: "a, b, c", ToolCallId: "call-1"},
		},
		Tools:       []*llm.ToolDefinition{{Name: "list_notes", Description: "Lists the notes.", ParametersJson: `{"type":"object","properties":{"limit":{"type":"number"}}}`}},
		Temperature: &temperature,
		MaxTokens:   200,
	}

	if _, err := runChat(t, newTestService(t), newCall("openai", address, "", "gpt-4o"), request); err != nil {
		t.Fatal(err)
	}

	body := lastChatBody(t, fake)
	messages := list(t, body["messages"])
	if len(messages) != 4 {
		t.Fatalf("body = %v", body)
	}
	assistant := object(t, messages[2])
	calls := list(t, assistant["tool_calls"])
	call := object(t, calls[0])
	function := object(t, call["function"])
	if _, hasContent := assistant["content"]; hasContent || assistant["role"] != "assistant" || call["id"] != "call-1" || call["type"] != "function" ||
		function["name"] != "list_notes" || function["arguments"] != `{"limit":3}` {
		t.Errorf("assistant message = %v", assistant)
	}
	tool := object(t, messages[3])
	if tool["role"] != "tool" || tool["tool_call_id"] != "call-1" || tool["content"] != "a, b, c" {
		t.Errorf("tool message = %v", tool)
	}
	tools := list(t, body["tools"])
	definition := object(t, tools[0])
	described := object(t, definition["function"])
	parameters := object(t, described["parameters"])
	if definition["type"] != "function" || described["name"] != "list_notes" || described["description"] != "Lists the notes." || parameters["type"] != "object" {
		t.Errorf("tools = %v", tools)
	}
	if body["temperature"] != 0.3 || body["max_completion_tokens"] != float64(200) {
		t.Errorf("body = %v", body)
	}
}

func TestChatJoinsTheStreamedPiecesOfAToolCallIntoOneEvent(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"delta":{"content":"Let me look."}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"list_notes","arguments":""}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"lim"}}]}}]}`,
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"it\":3}"}},{"index":1,"id":"call-2","function":{"name":"read_note","arguments":"{}"}}]}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
		"[DONE]",
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || len(events) != 4 {
		t.Fatalf("events = %+v, err = %v", events, err)
	}
	if events[0].TextDelta != "Let me look." ||
		*events[1].ToolCall != (toolCall{ID: "call-1", Name: "list_notes", Arguments: `{"limit":3}`}) ||
		*events[2].ToolCall != (toolCall{ID: "call-2", Name: "read_note", Arguments: `{}`}) ||
		events[3].Finish == nil || events[3].Finish.GetReason() != llm.FinishReason_FINISH_REASON_TOOL_CALLS {
		t.Errorf("events = %+v", events)
	}
}

func TestAToolCallWithEmptyArgumentsIsSentAsAnEmptyObject(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c","function":{"name":"ping"}}]},"finish_reason":"tool_calls"}]}`,
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || events[0].ToolCall == nil || events[0].ToolCall.Arguments != "{}" {
		t.Errorf("events = %+v, err = %v", events, err)
	}
}

func TestAnAnswerTruncatedAtTheLengthLimitFinishesWithLength(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"delta":{"content":"cut"}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"length"}]}`,
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || events[len(events)-1].Finish == nil || events[len(events)-1].Finish.GetReason() != llm.FinishReason_FINISH_REASON_LENGTH {
		t.Errorf("events = %+v, err = %v", events, err)
	}
}

func TestAServerThatAnswersAWholeMessageToAStreamRequestIsRead(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"message":{"role":"assistant","content":"whole","tool_calls":[{"id":"c","type":"function","function":{"name":"ping","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || len(events) != 3 || events[0].TextDelta != "whole" || events[1].ToolCall.Name != "ping" || events[2].Finish.GetReason() != llm.FinishReason_FINISH_REASON_TOOL_CALLS {
		t.Errorf("events = %+v, err = %v", events, err)
	}
}

func TestChatNamesTheReasoningLevelAsTheServiceExpectsIt(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	call := withReasoning(newCall("openrouter", address, "", "openai/gpt-5"), "low")

	if _, err := runChat(t, newTestService(t), call, chatUser("hi")); err != nil {
		t.Fatal(err)
	}

	reasoning := object(t, lastChatBody(t, fake)["reasoning"])
	if reasoning["effort"] != "low" {
		t.Errorf("body = %v", lastChatBody(t, fake))
	}
}

func TestChatStopsOnAReasoningLevelThatTheModelRejects(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	call := withReasoning(newCall("openai-compatible", address, "", "qwen2.5:14b"), "low")

	_, err := runChat(t, newTestService(t), call, chatUser("hi"))

	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests, want none", len(fake.Requests()))
	}
}

func TestTheStatusOfAChatAnswerDecidesTheKindOfTheFailure(t *testing.T) {
	cases := []struct {
		status     int
		retryAfter string
		want       failureWant
	}{
		{http.StatusUnauthorized, "", failureWant{isTryNext: true, reason: connection.ReasonRejectedKey}},
		{http.StatusPaymentRequired, "", failureWant{isTryNext: true, reason: connection.ReasonPaymentRequired}},
		{http.StatusTooManyRequests, "30", failureWant{isTryNext: true, reason: connection.ReasonRateLimited, retryAfter: 30 * time.Second}},
		{http.StatusBadGateway, "", failureWant{isTryNext: true, reason: connection.ReasonUnavailable}},
		{http.StatusServiceUnavailable, "", failureWant{isTryNext: true, reason: connection.ReasonUnavailable}},
		{http.StatusGatewayTimeout, "", failureWant{isTryNext: true, reason: connection.ReasonUnavailable}},
		{http.StatusBadRequest, "", failureWant{reason: connection.ReasonInvalidRequest}},
		{http.StatusInternalServerError, "", failureWant{reason: connection.ReasonUnavailable}},
	}
	for _, one := range cases {
		fake, address := startFakeServer(t, fakeopenai.Config{})
		fake.Fail("/chat/completions", one.status, one.retryAfter)

		events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"))

		checkFailure(t, err, one.want)
		if len(events) != 0 {
			t.Errorf("status %d: events = %+v, want none", one.status, events)
		}
	}
}

func TestAStreamThatBreaksAfterTheFirstTextKeepsTheTextAndFailsAsUnavailable(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{IsBroken: true, Chunks: []string{`{"choices":[{"delta":{"content":"partial"}}]}`}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("hi"))

	if len(events) != 1 || events[0].TextDelta != "partial" {
		t.Errorf("events = %+v", events)
	}
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestAStreamThatEndsWithoutAFinishFailsAsUnavailable(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{`{"choices":[{"delta":{"content":"cut"}}]}`, "[DONE]"}})

	_, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("hi"))

	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestAnErrorEventOfTheStreamIsAFailureOfItsKind(t *testing.T) {
	for code, want := range map[string]failureWant{
		"429": {isTryNext: true, reason: connection.ReasonRateLimited},
		"402": {isTryNext: true, reason: connection.ReasonPaymentRequired},
		"401": {isTryNext: true, reason: connection.ReasonRejectedKey},
		"503": {isTryNext: true, reason: connection.ReasonUnavailable},
		"400": {reason: connection.ReasonInvalidRequest},
	} {
		fake, address := startFakeServer(t, fakeopenai.Config{})
		fake.QueueStream(fakeopenai.Stream{Chunks: []string{
			`{"choices":[{"delta":{"content":"partial"}}]}`,
			`{"error":{"code":` + code + `,"message":"upstream said Bearer ` + testKey + `"}}`,
		}})

		events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"))

		checkFailure(t, err, want)
		if len(events) != 1 || strings.Contains(err.Error(), testKey) {
			t.Errorf("code %s: events = %+v, err = %v", code, events, err)
		}
	}
}

func TestAChatThatStallsAfterAnEventIsUnavailableWhenNothingMoreArrives(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{`{"choices":[{"delta":{"content":"partial"}}]}`}})
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.2"})

	events, err := runChat(t, svc, newCall("openrouter", address, testKey, "m"), chatUser("hi"))

	if len(events) != 1 {
		t.Errorf("events = %+v", events)
	}
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}

func TestTheTimeOfAChatRestartsWithEachEvent(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	var chunks []string
	for range 6 {
		chunks = append(chunks, `{"choices":[{"delta":{"content":"x"}}]}`)
	}
	chunks = append(chunks, `{"choices":[{"delta":{},"finish_reason":"stop"}]}`, "[DONE]")
	fake.QueueStream(fakeopenai.Stream{Chunks: chunks, Pause: 100 * time.Millisecond})
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.3"})

	events, err := runChat(t, svc, newCall("openrouter", address, testKey, "m"), chatUser("hi"))

	if err != nil || joinText(events) != "xxxxxx" {
		t.Errorf("events = %+v, err = %v: a stream longer than the timeout must go on while events arrive", events, err)
	}
}

func TestAChatCanceledWhileTheServiceIsSilentEndsTheRequest(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 30*time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	finish := make(chan error, 1)
	go func() {
		_, err := newTestService(t).Chat(ctx, newCall("openrouter", address, testKey, "m"), chatUser("hi"), ignoreFrame)
		finish <- err
	}()
	time.Sleep(100 * time.Millisecond)

	cancel()

	if err := <-finish; !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the cancellation", err)
	}
	select {
	case <-fake.Aborted():
	case <-time.After(5 * time.Second):
		t.Error("the request to the service was not abandoned")
	}
}

func TestAChatStopsWhenTheReceiverOfTheEventsFails(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{})
	consumerFailure := errors.New("the stream is closed")

	_, err := newTestService(t).Chat(newContext(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"), func(*llm.ChatFrame) error { return consumerFailure })

	if !errors.Is(err, consumerFailure) {
		t.Errorf("err = %v, want the error of the receiver", err)
	}
}

func TestNoEventAndNoFailureOfAChatContainsTheKey(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{APIKey: "another-key"})

	events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"))

	if err == nil || strings.Contains(joinText(events)+err.Error(), testKey) {
		t.Errorf("events = %+v, err = %v, want a failure without the key", events, err)
	}
}

func TestChatStopsOnAnUnknownTypeAndALoneKeyOverPlainHTTPToARemoteHost(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{})
	_, err := runChat(t, newTestService(t), newCall("nope", address, "", "m"), chatUser("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
	_, err = runChat(t, newTestService(t), newCall("openai-compatible", "http://203.0.113.9/v1", testKey, "m"), chatUser("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
}

func object(t *testing.T, value any) map[string]any {
	t.Helper()
	one, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%v is no JSON object", value)
	}
	return one
}

func list(t *testing.T, value any) []any {
	t.Helper()
	one, ok := value.([]any)
	if !ok {
		t.Fatalf("%v is no JSON list", value)
	}
	return one
}

func TestAChatWhoseServiceSendsHeadersAndThenNothingIsUnavailableAfterTheLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.2"})
	finish := make(chan error, 1)

	go func() {
		_, err := runChat(t, svc, newCall("openrouter", server.URL+"/v1", testKey, "m"), chatUser("hi"))
		finish <- err
	}()

	select {
	case err := <-finish:
		checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
	case <-time.After(5 * time.Second):
		t.Fatal("the chat waited past its limit for the first byte of the answer")
	}
}

func TestAWholeReplyLargerThanTheBoundStopsTheChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"`)
		piece := strings.Repeat("x", 1<<20)
		for range 65 {
			if _, err := io.WriteString(w, piece); err != nil {
				return
			}
		}
		_, _ = io.WriteString(w, `"},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(server.Close)

	_, err := newTestService(t).Chat(newContext(t), newCall("openai-compatible", server.URL+"/v1", "", "m"), chatUser("hi"), ignoreFrame)

	checkFailure(t, err, failureWant{isTryNext: false, reason: connection.ReasonUnavailable})
}
