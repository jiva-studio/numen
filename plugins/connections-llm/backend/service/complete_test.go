package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"

	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestCompleteSendsAChatRequestAndReturnsTheAnswer(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})
	temperature := 0.25
	request := oneUserMessage("fix this line")
	request.Messages = append([]*llm.Message{{Role: "system", Text: "You proofread."}}, request.GetMessages()...)
	request.Temperature = &temperature
	request.MaxTokens = 300

	result, err := newTestService(t).Complete(newContext(t), newCall("openrouter", address, testKey, "openai/gpt-4o-mini"), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.GetText() != "echo: fix this line" {
		t.Errorf("text = %q", result.GetText())
	}
	requests := fake.Requests()
	if len(requests) != 1 || requests[0].Method != http.MethodPost || requests[0].Path != "/chat/completions" {
		t.Fatalf("requests = %+v", requests)
	}
	if got := requests[0].Header.Get("Authorization"); got != "Bearer "+testKey {
		t.Errorf("Authorization = %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal(requests[0].Body, &body); err != nil {
		t.Fatal(err)
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("body = %s", requests[0].Body)
	}
	if body["model"] != "openai/gpt-4o-mini" || body["temperature"] != 0.25 || body["max_tokens"] != float64(300) {
		t.Errorf("body = %s", requests[0].Body)
	}
	first, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("first message = %v", messages[0])
	}
	if first["role"] != "system" || first["content"] != "You proofread." {
		t.Errorf("first message = %v", first)
	}
}

func TestCompleteSendsNoKeyHeaderWhenTheConnectionHasNoKey(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	if _, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", address, "", "qwen2.5:14b"), oneUserMessage("hi")); err != nil {
		t.Fatal(err)
	}
	if got := fake.Requests()[0].Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want none", got)
	}
}

func TestCompleteOmitsTheOptionsTheRequestDoesNotSet(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	if _, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", address, "", "m"), oneUserMessage("hi")); err != nil {
		t.Fatal(err)
	}
	body := string(fake.Requests()[0].Body)
	for _, key := range []string{"temperature", "max_tokens", "max_completion_tokens", "reasoning"} {
		if strings.Contains(body, key) {
			t.Errorf("body %s holds %s", body, key)
		}
	}
}

func TestCompleteNamesTheTokenLimitAsTheServiceExpectsIt(t *testing.T) {
	for typeID, want := range map[string]string{"openai": "max_completion_tokens", "openrouter": "max_tokens", "openai-compatible": "max_tokens"} {
		fake, address := startFakeServer(t, fakeopenai.Config{})
		request := oneUserMessage("hi")
		request.MaxTokens = 50
		if _, err := newTestService(t).Complete(newContext(t), newCall(typeID, address, "", "m"), request); err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(fake.Requests()[0].Body, &body); err != nil {
			t.Fatal(err)
		}
		if body[want] != float64(50) {
			t.Errorf("%s: body %s lacks %s", typeID, fake.Requests()[0].Body, want)
		}
	}
}

func TestCompleteNamesTheReasoningLevelAsTheServiceExpectsIt(t *testing.T) {
	cases := map[string]struct {
		model string
		check func(body map[string]any) bool
	}{
		"openrouter": {"openai/gpt-5", func(body map[string]any) bool {
			reasoning, ok := body["reasoning"].(map[string]any)
			return ok && reasoning["effort"] == "low"
		}},
		"openai": {"gpt-5-mini", func(body map[string]any) bool { return body["reasoning_effort"] == "low" }},
	}
	for typeID, one := range cases {
		fake, address := startFakeServer(t, fakeopenai.Config{})
		call := withReasoning(newCall(typeID, address, "", one.model), "low")
		if _, err := newTestService(t).Complete(newContext(t), call, oneUserMessage("hi")); err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(fake.Requests()[0].Body, &body); err != nil {
			t.Fatal(err)
		}
		if !one.check(body) {
			t.Errorf("%s: body %s", typeID, fake.Requests()[0].Body)
		}
	}
}

func TestCompleteStopsOnAReasoningLevelThatTheModelRejects(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	for typeID, model := range map[string]string{"openai-compatible": "qwen2.5:14b", "openai": "gpt-4o-mini"} {
		call := withReasoning(newCall(typeID, address, "", model), "low")
		_, err := newTestService(t).Complete(newContext(t), call, oneUserMessage("hi"))
		checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
	}
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests, want none", len(fake.Requests()))
	}
}

func TestCompleteStopsOnAnAnswerWithoutChoices(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`{"choices":[]}`)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(ts.Close)
	_, err := newTestService(t).Complete(newContext(t), newCall("openai-compatible", ts.URL, "", "m"), oneUserMessage("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonUnavailable})
}

func TestCompleteStopsOnAnUnknownType(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{})
	_, err := newTestService(t).Complete(newContext(t), newCall("nope", address, "", "m"), oneUserMessage("hi"))
	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
}
