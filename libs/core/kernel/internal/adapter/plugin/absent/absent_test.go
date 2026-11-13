package absent

import (
	"errors"
	"testing"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestRuntimeAnswersUnavailableForItsReason(t *testing.T) {
	cause := errors.New("safe mode")
	runtime := Runtime{Reason: plugindomain.Disabled, Cause: cause}

	_, err := runtime.ResolveCapability("search.source")
	var unavailable *plugindomain.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Reason != plugindomain.Disabled || unavailable.Capability != "search.source" || !errors.Is(err, cause) {
		t.Fatalf("ResolveCapability = %v, want the embedding plugin disabled by safe mode", err)
	}
	_, err = runtime.RetryPlugin(t.Context(), "ocr")
	if !errors.As(err, &unavailable) || unavailable.PluginID != "ocr" {
		t.Fatalf("RetryPlugin = %v, want the ocr plugin unavailable", err)
	}
}

func TestRuntimeHasNoPlugin(t *testing.T) {
	runtime := Runtime{Reason: plugindomain.NotInstalled}

	if len(runtime.GetPluginInfos()) != 0 || runtime.IsPluginEnabled("ocr") || runtime.IsEnabled("ocr") {
		t.Error("an absent runtime lists or enables a plugin")
	}
	if _, found := runtime.GetPlugin("ocr"); found {
		t.Error("an absent runtime found a plugin")
	}
	runtime.WatchPluginConfig("ocr", func() {})()
	if err := (Settings{}).TeardownAll(t.Context(), 0); err != nil {
		t.Errorf("TeardownAll = %v", err)
	}
}
