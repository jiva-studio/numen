package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func chatUser(text string) connection.ProviderChatRequest {
	return connection.ProviderChatRequest{Messages: []connection.ChatMessage{{Role: "user", Text: text}}}
}

func runChat(t *testing.T, svc *testService, call connection.Call, request connection.ProviderChatRequest) ([]connection.ChatEvent, error) {
	t.Helper()
	var events []connection.ChatEvent
	err := svc.Chat(newContext(t), call, request, func(event connection.ChatEvent) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

func joinText(events []connection.ChatEvent) string {
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
	fake, address := startFake(t, fakeopenai.Config{APIKey: testKey})

	events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "openai/gpt-4o-mini"), chatUser("hello there"))

	if err != nil {
		t.Fatal(err)
	}
	last := events[len(events)-1]
	if joinText(events) != "echo: hello there" || len(events) < 3 || last.Finish == nil || last.Finish.Reason != connection.FinishStop {
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
	fake, address := startFake(t, fakeopenai.Config{})
	temperature := 0.3
	request := connection.ProviderChatRequest{
		Messages: []connection.ChatMessage{
			{Role: "system", Text: "You are an agent."},
			{Role: "user", Text: "list the notes"},
			{Role: "assistant", ToolCalls: []connection.ToolCall{{ID: "call-1", Name: "list_notes", Arguments: `{"limit":3}`}}},
			{Role: "tool", Text: "a, b, c", ToolCallID: "call-1"},
		},
		Tools:       []connection.ToolDefinition{{Name: "list_notes", Description: "Lists the notes.", Parameters: json.RawMessage(`{"type":"object","properties":{"limit":{"type":"number"}}}`)}},
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
	fake, address := startFake(t, fakeopenai.Config{})
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
		*events[1].ToolCall != (connection.ToolCall{ID: "call-1", Name: "list_notes", Arguments: `{"limit":3}`}) ||
		*events[2].ToolCall != (connection.ToolCall{ID: "call-2", Name: "read_note", Arguments: `{}`}) ||
		events[3].Finish == nil || events[3].Finish.Reason != connection.FinishToolCalls {
		t.Errorf("events = %+v", events)
	}
}

func TestAToolCallWithEmptyArgumentsIsSentAsAnEmptyObject(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c","function":{"name":"ping"}}]},"finish_reason":"tool_calls"}]}`,
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || events[0].ToolCall == nil || events[0].ToolCall.Arguments != "{}" {
		t.Errorf("events = %+v, err = %v", events, err)
	}
}

func TestAnAnswerTruncatedAtTheLengthLimitFinishesWithLength(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"delta":{"content":"cut"}}]}`,
		`{"choices":[{"delta":{},"finish_reason":"length"}]}`,
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || events[len(events)-1].Finish == nil || events[len(events)-1].Finish.Reason != connection.FinishLength {
		t.Errorf("events = %+v, err = %v", events, err)
	}
}

func TestAServerThatAnswersAWholeMessageToAStreamRequestIsRead(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{
		`{"choices":[{"message":{"role":"assistant","content":"whole","tool_calls":[{"id":"c","type":"function","function":{"name":"ping","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
	}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("go"))

	if err != nil || len(events) != 3 || events[0].TextDelta != "whole" || events[1].ToolCall.Name != "ping" || events[2].Finish.Reason != connection.FinishToolCalls {
		t.Errorf("events = %+v, err = %v", events, err)
	}
}

func TestChatNamesTheReasoningLevelAsTheServiceExpectsIt(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	call := newCall("openrouter", address, "", "openai/gpt-5")
	call.Reasoning = "low"

	if _, err := runChat(t, newTestService(t), call, chatUser("hi")); err != nil {
		t.Fatal(err)
	}

	reasoning := object(t, lastChatBody(t, fake)["reasoning"])
	if reasoning["effort"] != "low" {
		t.Errorf("body = %v", lastChatBody(t, fake))
	}
}

func TestChatStopsOnAReasoningLevelThatTheModelRejects(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	call := newCall("openai-compatible", address, "", "qwen2.5:14b")
	call.Reasoning = "low"

	_, err := runChat(t, newTestService(t), call, chatUser("hi"))

	checkFailure(t, err, failureWant{reason: connection.ReasonBadRequest})
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
		{http.StatusUnauthorized, "", failureWant{isTryNext: true, reason: connection.ReasonKeyRejected}},
		{http.StatusPaymentRequired, "", failureWant{isTryNext: true, reason: connection.ReasonPaymentRequired}},
		{http.StatusTooManyRequests, "30", failureWant{isTryNext: true, reason: connection.ReasonRateLimited, retryAfter: 30 * time.Second}},
		{http.StatusBadGateway, "", failureWant{isTryNext: true, reason: connection.ReasonUnreachable}},
		{http.StatusServiceUnavailable, "", failureWant{isTryNext: true, reason: connection.ReasonUnreachable}},
		{http.StatusGatewayTimeout, "", failureWant{isTryNext: true, reason: connection.ReasonUnreachable}},
		{http.StatusBadRequest, "", failureWant{reason: connection.ReasonBadRequest}},
		{http.StatusInternalServerError, "", failureWant{reason: connection.ReasonUnknown}},
	}
	for _, one := range cases {
		fake, address := startFake(t, fakeopenai.Config{})
		fake.Fail("/chat/completions", one.status, one.retryAfter)

		events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"))

		checkFailure(t, err, one.want)
		if len(events) != 0 {
			t.Errorf("status %d: events = %+v, want none", one.status, events)
		}
	}
}

func TestAStreamThatBreaksAfterTheFirstTextKeepsTheTextAndFailsAsUnreachable(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{IsBroken: true, Chunks: []string{`{"choices":[{"delta":{"content":"partial"}}]}`}})

	events, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("hi"))

	if len(events) != 1 || events[0].TextDelta != "partial" {
		t.Errorf("events = %+v", events)
	}
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnreachable})
}

func TestAStreamThatEndsWithoutAFinishFailsAsUnreachable(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{`{"choices":[{"delta":{"content":"cut"}}]}`, "[DONE]"}})

	_, err := runChat(t, newTestService(t), newCall("openai-compatible", address, "", "m"), chatUser("hi"))

	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnreachable})
}

func TestAnErrorEventOfTheStreamIsAFailureOfItsKind(t *testing.T) {
	for code, want := range map[string]failureWant{
		"429": {isTryNext: true, reason: connection.ReasonRateLimited},
		"402": {isTryNext: true, reason: connection.ReasonPaymentRequired},
		"401": {isTryNext: true, reason: connection.ReasonKeyRejected},
		"503": {isTryNext: true, reason: connection.ReasonUnreachable},
		"400": {reason: connection.ReasonBadRequest},
	} {
		fake, address := startFake(t, fakeopenai.Config{})
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

func TestAChatThatStallsAfterAnEventIsATimeoutWhenNothingMoreArrives(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.QueueStream(fakeopenai.Stream{Chunks: []string{`{"choices":[{"delta":{"content":"partial"}}]}`}})
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.2"})

	events, err := runChat(t, svc, newCall("openrouter", address, testKey, "m"), chatUser("hi"))

	if len(events) != 1 {
		t.Errorf("events = %+v", events)
	}
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonTimeout})
}

func TestTheTimeOfAChatRestartsWithEachEvent(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
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
	fake, address := startFake(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 30*time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	finish := make(chan error, 1)
	go func() {
		finish <- newTestService(t).Chat(ctx, newCall("openrouter", address, testKey, "m"), chatUser("hi"), func(connection.ChatEvent) error { return nil })
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
	_, address := startFake(t, fakeopenai.Config{})
	consumerFailure := errors.New("the stream is closed")

	err := newTestService(t).Chat(newContext(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"), func(connection.ChatEvent) error { return consumerFailure })

	if !errors.Is(err, consumerFailure) {
		t.Errorf("err = %v, want the error of the receiver", err)
	}
}

func TestNoEventAndNoFailureOfAChatContainsTheKey(t *testing.T) {
	_, address := startFake(t, fakeopenai.Config{APIKey: "another-key"})

	events, err := runChat(t, newTestService(t), newCall("openrouter", address, testKey, "m"), chatUser("hi"))

	if err == nil || strings.Contains(joinText(events)+err.Error(), testKey) {
		t.Errorf("events = %+v, err = %v, want a failure without the key", events, err)
	}
}

func TestChatStopsOnAnUnknownTypeAndALoneKeyOverPlainHTTPToARemoteHost(t *testing.T) {
	_, address := startFake(t, fakeopenai.Config{})
	_, err := runChat(t, newTestService(t), newCall("nope", address, "", "m"), chatUser("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonBadRequest})
	_, err = runChat(t, newTestService(t), newCall("openai-compatible", "http://203.0.113.9/v1", testKey, "m"), chatUser("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonBadRequest})
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
