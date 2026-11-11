package plugin_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func applyChange(host *container.PluginRoles, ctx context.Context, change func() ([]string, error)) error {
	readers, err := change()
	if err != nil {
		return err
	}
	return host.Config.Refresh(ctx, readers...)
}
