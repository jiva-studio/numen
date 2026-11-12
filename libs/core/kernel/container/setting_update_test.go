package container_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func updateSetting(ctx context.Context, host *container.PluginRoles, pluginID, key string, value any) error {
	return container.AdaptPluginHost(host).ApplySetting(ctx, pluginID, key, value)
}

func setEnabled(ctx context.Context, host *container.PluginRoles, pluginID string, enabled bool) error {
	return updateSetting(ctx, host, pluginID, "enabled", enabled)
}
