package settings

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// RetryPlugin starts a plugin whose last start failed once more, and answers the failure where it fails again.
type RetryPlugin struct {
	retry func(ctx context.Context, pluginID string) (*plugin.Failure, error)
}

// NewRetryPlugin creates the use case over the host's retry.
func NewRetryPlugin(retry func(ctx context.Context, pluginID string) (*plugin.Failure, error)) *RetryPlugin {
	return &RetryPlugin{retry: retry}
}

// Execute starts the plugin once more and answers the failure where it fails again.
func (u *RetryPlugin) Execute(ctx context.Context, pluginID string) (*plugin.Failure, error) {
	return u.retry(ctx, pluginID)
}
