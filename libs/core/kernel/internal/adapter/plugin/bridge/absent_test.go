package bridge

import (
	"errors"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestAbsentRuntimeAnswersUnavailableForItsReason(t *testing.T) {
	cause := errors.New("safe mode")
	runtime := AbsentRuntime{Reason: plugindomain.Disabled, Cause: cause}

	_, err := runtime.ResolveCapability("embedding")
	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugindomain.Disabled || unavailable.Capability != "embedding" || !errors.Is(err, cause) {
		t.Fatalf("ResolveCapability = %v, want the embedding plugin disabled by safe mode", err)
	}
	_, err = runtime.RetryPlugin(t.Context(), "ocr")
	if !errors.As(err, &unavailable) || unavailable.PluginID != "ocr" {
		t.Fatalf("RetryPlugin = %v, want the ocr plugin unavailable", err)
	}
}

func TestAbsentRuntimeHoldsNoPlugin(t *testing.T) {
	runtime := AbsentRuntime{Reason: plugindomain.NotInstalled}

	if len(runtime.GetPluginInfos()) != 0 || runtime.IsPluginEnabled("ocr") || runtime.IsEnabled("ocr") {
		t.Error("an absent runtime lists or enables a plugin")
	}
	if _, found := runtime.GetPlugin("ocr"); found {
		t.Error("an absent runtime found a plugin")
	}
	runtime.WatchPluginConfig("ocr", func() {})()
	if err := (AbsentSettings{}).TeardownAll(t.Context(), 0); err != nil {
		t.Errorf("TeardownAll = %v", err)
	}
}
