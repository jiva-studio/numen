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
	t.Setenv("NUMEN_ONNX_THREADS", "1")
	if got := (Variables{}).GetOverrides().Threads; got != 1 {
		t.Errorf("threads \"1\" = %d, want 1", got)
	}
	for _, value := range []string{"", "invalid", "0", "-2"} {
		t.Setenv("NUMEN_ONNX_THREADS", value)
		if got := (Variables{}).GetOverrides().Threads; got != 0 {
			t.Errorf("threads %q = %d, want 0", value, got)
		}
	}
}

func TestTheMockOpenHoldWaitsAndStopsWithItsContext(t *testing.T) {
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	for _, value := range []string{"0", "-1", "bad"} {
		t.Setenv(envMockOpenHold, value)
		if err := (Variables{}).WaitMockOpenHold(canceled); err != nil {
			t.Fatalf("%q hold with canceled context = %v, want nil", value, err)
		}
	}

	t.Setenv(envMockOpenHold, "1")
	if err := (Variables{}).WaitMockOpenHold(t.Context()); err != nil {
		t.Fatalf("1ms hold failed: %v", err)
	}
	if err := (Variables{}).WaitMockOpenHold(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("1ms hold with a canceled context = %v, want Canceled", err)
	}
}
