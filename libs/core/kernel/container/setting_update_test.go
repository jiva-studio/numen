package container_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"
)

func updateSetting(ctx context.Context, host *pluginhost.Host, pluginID, key string, value any) error {
	return container.AdaptPluginHost(host).ApplySetting(ctx, pluginID, key, value)
}

func setEnabled(ctx context.Context, host *pluginhost.Host, pluginID string, enabled bool) error {
	return updateSetting(ctx, host, pluginID, "enabled", enabled)
}
