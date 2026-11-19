package container_test

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

func updateSetting(ctx context.Context, host *pluginhost.Host, pluginID, key string, value any) error {
	return container.AdaptTestPlugins(host.Management, host.Supervision).Preferences.ApplySetting(ctx, pluginID, key, value)
}

func setEnabled(ctx context.Context, host *pluginhost.Host, pluginID string, enabled bool) error {
	return updateSetting(ctx, host, pluginID, "enabled", enabled)
}
