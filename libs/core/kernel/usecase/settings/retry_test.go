package settings_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func TestARetryPluginPassesTheIdToTheRetrier(t *testing.T) {
	var got string
	retry := settings.NewRetryPlugin(func(_ context.Context, id string) (*plugin.Failure, error) {
		got = id
		return &plugin.Failure{PluginID: id}, nil
	})

	failure, err := retry.Execute(t.Context(), "md.numen.sample")

	if err != nil || got != "md.numen.sample" || failure == nil || failure.PluginID != "md.numen.sample" {
		t.Fatalf("Execute() = %+v, %v with %q", failure, err, got)
	}
}
