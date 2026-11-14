package limits

import (
	"testing"
	"time"
)

const address = "https://openrouter.ai/api/v1"

func TestTheLimitOfARequestDefaultsToTheOperation(t *testing.T) {
	var subject Limits
	if got := subject.Get("openrouter", address, "m", "complete"); got != 120*time.Second {
		t.Errorf("complete limit = %v, want 120 s", got)
	}
	if got := subject.Get("openrouter", address, "m", "embed"); got != 60*time.Second {
		t.Errorf("embed limit = %v, want 60 s", got)
	}
}

func TestTheSettingRequestTimeoutSetsTheLimitOfEveryOperation(t *testing.T) {
	var subject Limits
	subject.Configure(map[string]string{"requestTimeout": "45"})
	for _, operation := range []string{"complete", "embed", "chat", "list"} {
		if got := subject.Get("openai", address, "m", operation); got != 45*time.Second {
			t.Errorf("%s limit = %v, want 45 s", operation, got)
		}
	}
}

func TestASettingThatIsNotAPositiveNumberKeepsTheDefault(t *testing.T) {
	for _, value := range []string{"", "soon", "0", "-3", "NaN", "Inf"} {
		var subject Limits
		subject.Configure(map[string]string{"requestTimeout": value})
		if got := subject.Get("openai", address, "m", "complete"); got != 120*time.Second {
			t.Errorf("value %q: limit = %v, want 120 s", value, got)
		}
	}
}

func TestTheFirstRequestOfAnOpenAICompatibleModelHasALongLimit(t *testing.T) {
	var subject Limits
	subject.Configure(map[string]string{"requestTimeout": "0.1"})
	if got := subject.Get("openai-compatible", address, "qwen2.5:14b", "complete"); got != FirstRequestLimit {
		t.Fatalf("limit before the first request = %v, want %v", got, FirstRequestLimit)
	}

	subject.MarkServed(address, "qwen2.5:14b")

	if got := subject.Get("openai-compatible", address, "qwen2.5:14b", "complete"); got != 100*time.Millisecond {
		t.Errorf("limit after the first request = %v, want 100 ms", got)
	}
}

func TestTheLongLimitBelongsToTheAddressAndTheModel(t *testing.T) {
	var subject Limits
	subject.MarkServed(address, "qwen2.5:14b")

	if got := subject.Get("openai-compatible", address, "llama3", "complete"); got != FirstRequestLimit {
		t.Errorf("another model: limit = %v, want %v", got, FirstRequestLimit)
	}
	if got := subject.Get("openai-compatible", "http://localhost:1/v1", "qwen2.5:14b", "complete"); got != FirstRequestLimit {
		t.Errorf("another address: limit = %v, want %v", got, FirstRequestLimit)
	}
	if got := subject.Get("openrouter", address, "m", "complete"); got != 120*time.Second {
		t.Errorf("a type with a known service: limit = %v, want 120 s", got)
	}
}
