package container_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
)

func updateSetting(ctx context.Context, host *plugin.Host, pluginID, key string, value any) error {
	return container.AdaptPluginHost(host).ApplySetting(ctx, pluginID, key, value)
}

func setEnabled(ctx context.Context, host *plugin.Host, pluginID string, enabled bool) error {
	return updateSetting(ctx, host, pluginID, "enabled", enabled)
}
