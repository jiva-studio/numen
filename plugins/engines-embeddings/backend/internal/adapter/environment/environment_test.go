package environment

import (
	"context"
	"errors"
	"testing"
)

func TestParsePositiveIntKeepsTheBoundsOfAnInt32(t *testing.T) {
	for value, want := range map[string]int{
		"": 0, "0": 0, "-1": 0, "1": 1, "2147483647": 2147483647, "2147483648": 0, "many": 0,
	} {
		if got := parsePositiveInt(value); got != want {
			t.Errorf("parsePositiveInt(%q) = %d, want %d", value, got, want)
		}
	}
}

func TestTheVariablesNameTheModelAndTheRuntime(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	t.Setenv("NUMEN_EMBEDDING_DIMENSIONS", "1")
	t.Setenv("NUMEN_EMBEDDING_MAX_TOKENS", "4294967296")
	t.Setenv("NUMEN_ONNX_THREADS", "6")

	got := (Variables{}).GetOverrides()

	if got.Model != "owner/model" || got.Dimensions != 1 || got.MaxTokens != 0 || got.Threads != 6 {
		t.Fatalf("overrides = %+v", got)
	}
}

func TestAThreadCountThatIsNoPositiveNumberNamesNothing(t *testing.T) {
	for _, value := range []string{"", "invalid", "0", "-2"} {
		t.Setenv("NUMEN_ONNX_THREADS", value)
		if got := (Variables{}).GetOverrides().Threads; got != 0 {
			t.Errorf("threads %q = %d, want 0", value, got)
		}
	}
}

func TestTheMockOpenHoldWaitsAndStopsWithItsContext(t *testing.T) {
	t.Setenv(envMockOpenHold, "0")
	if err := (Variables{}).WaitMockOpenHold(t.Context()); err != nil {
		t.Fatalf("0ms hold failed: %v", err)
	}

	t.Setenv(envMockOpenHold, "bad")
	if err := (Variables{}).WaitMockOpenHold(t.Context()); err != nil {
		t.Fatalf("invalid hold failed: %v", err)
	}

	t.Setenv(envMockOpenHold, "1")
	if err := (Variables{}).WaitMockOpenHold(t.Context()); err != nil {
		t.Fatalf("1ms hold failed: %v", err)
	}

	t.Setenv(envMockOpenHold, "200")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := (Variables{}).WaitMockOpenHold(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("hold with a canceled context = %v, want Canceled", err)
	}
}
