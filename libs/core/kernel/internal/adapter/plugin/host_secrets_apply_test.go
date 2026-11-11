package plugin_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

func applyChange(host *pluginhost.Host, ctx context.Context, change func() ([]string, error)) error {
	readers, err := change()
	if err != nil {
		return err
	}
	return host.Config.Refresh(ctx, readers...)
}
