package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/settings"
)

func TestAStopFailedPluginMadeWithoutAStopperIsRefused(t *testing.T) {
	stop, err := settings.NewStopFailedPlugin(nil)

	if !errors.Is(err, settings.ErrNoStopper) || stop != nil {
		t.Fatalf("NewStopFailedPlugin(nil) = %v, %v, want %v", stop, err, settings.ErrNoStopper)
	}
}

func TestAStopFailedPluginPassesTheIdAndTheReasonToTheStopper(t *testing.T) {
	var gotID, gotReason string
	stop, err := settings.NewStopFailedPlugin(func(_ context.Context, id, reason string) (*settings.PluginFailure, error) {
		gotID, gotReason = id, reason
		return &settings.PluginFailure{PluginID: id, Detail: reason}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	failure, err := stop.Execute(t.Context(), "md.numen.sample", "the plugin did not start within 30 s")

	if err != nil || gotID != "md.numen.sample" || gotReason != "the plugin did not start within 30 s" || failure == nil {
		t.Fatalf("Execute() = %+v, %v with %q, %q", failure, err, gotID, gotReason)
	}
}
