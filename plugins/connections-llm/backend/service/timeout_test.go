package service

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

func TestTheLimitOfARequestDefaultsToTheOperation(t *testing.T) {
	svc := New()
	call := newCall("openrouter", "https://openrouter.ai/api/v1", testKey, "m")
	if got := svc.getLimit(call, operationComplete); got != 120*time.Second {
		t.Errorf("complete limit = %v, want 120 s", got)
	}
	if got := svc.getLimit(call, operationEmbed); got != 60*time.Second {
		t.Errorf("embed limit = %v, want 60 s", got)
	}
}

func TestTheSettingRequestTimeoutSetsTheLimitOfBothOperations(t *testing.T) {
	svc := New()
	svc.Configure(map[string]string{"requestTimeout": "45"})
	call := newCall("openai", "https://api.openai.com/v1", testKey, "m")
	for _, operation := range []string{operationComplete, operationEmbed} {
		if got := svc.getLimit(call, operation); got != 45*time.Second {
			t.Errorf("%s limit = %v, want 45 s", operation, got)
		}
	}
}

func TestASettingThatIsNotAPositiveNumberLeavesTheDefault(t *testing.T) {
	call := newCall("openai", "https://api.openai.com/v1", testKey, "m")
	for _, value := range []string{"", "soon", "0", "-3", "NaN", "Inf"} {
		svc := New()
		svc.Configure(map[string]string{"requestTimeout": value})
		if got := svc.getLimit(call, operationComplete); got != 120*time.Second {
			t.Errorf("value %q: limit = %v, want 120 s", value, got)
		}
	}
}

func TestTheFirstRequestOfAnOpenAICompatibleModelHasALongLimit(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.Delay("/chat/completions", 300*time.Millisecond)
	svc := New()
	svc.Configure(map[string]string{"requestTimeout": "0.1"})
	call := newCall("openai-compatible", address, "", "qwen2.5:14b")
	if got := svc.getLimit(call, operationComplete); got != firstRequestLimit {
		t.Fatalf("limit before the first request = %v, want %v", got, firstRequestLimit)
	}

	if _, err := svc.Complete(newContext(t), call, oneUserMessage("hi")); err != nil {
		t.Fatalf("the first request: %v", err)
	}
	if got := svc.getLimit(call, operationComplete); got != 100*time.Millisecond {
		t.Errorf("limit after the first request = %v, want 100 ms", got)
	}
	_, err := svc.Complete(newContext(t), call, oneUserMessage("hi"))
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonTimeout})
}

func TestTheLongLimitBelongsToTheAddressAndTheModel(t *testing.T) {
	_, address := startFake(t, fakeopenai.Config{})
	svc := New()
	call := newCall("openai-compatible", address, "", "qwen2.5:14b")
	if _, err := svc.Complete(newContext(t), call, oneUserMessage("hi")); err != nil {
		t.Fatal(err)
	}
	other := newCall("openai-compatible", address, "", "llama3")
	if got := svc.getLimit(other, operationComplete); got != firstRequestLimit {
		t.Errorf("another model: limit = %v, want %v", got, firstRequestLimit)
	}
	if got := svc.getLimit(newCall("openrouter", address, testKey, "m"), operationComplete); got != 120*time.Second {
		t.Errorf("a type with a known service: limit = %v, want 120 s", got)
	}
}

func TestAFailedFirstRequestLeavesTheLongLimit(t *testing.T) {
	fake, address := startFake(t, fakeopenai.Config{})
	fake.Fail("/chat/completions", 503, "")
	svc := New()
	call := newCall("openai-compatible", address, "", "qwen2.5:14b")
	if _, err := svc.Complete(newContext(t), call, oneUserMessage("hi")); err == nil {
		t.Fatal("want a failure")
	}
	if got := svc.getLimit(call, operationComplete); got != firstRequestLimit {
		t.Errorf("limit = %v, want %v", got, firstRequestLimit)
	}
}
