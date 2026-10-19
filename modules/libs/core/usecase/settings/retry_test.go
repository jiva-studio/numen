package settings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/settings"
)

func TestARetryPluginMadeWithoutARetrierIsRefused(t *testing.T) {
	retry, err := settings.NewRetryPlugin(nil)

	if !errors.Is(err, settings.ErrNoRetrier) || retry != nil {
		t.Fatalf("NewRetryPlugin(nil) = %v, %v, want %v", retry, err, settings.ErrNoRetrier)
	}
}

func TestARetryPluginHandsTheIdToTheRetrier(t *testing.T) {
	var got string
	retry, err := settings.NewRetryPlugin(func(_ context.Context, id string) (*settings.PluginFailure, error) {
		got = id
		return &settings.PluginFailure{PluginID: id}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	failure, err := retry.Execute(t.Context(), "md.numen.sample")

	if err != nil || got != "md.numen.sample" || failure == nil || failure.PluginID != "md.numen.sample" {
		t.Fatalf("Execute() = %+v, %v with %q", failure, err, got)
	}
}
