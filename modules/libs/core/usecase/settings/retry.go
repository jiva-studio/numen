package settings

import "context"

// RetryPlugin starts a plugin the host holds off once more, and answers the failure where it fails again.
type RetryPlugin struct {
	retry func(ctx context.Context, pluginID string) (*PluginFailure, error)
}

func NewRetryPlugin(retry func(ctx context.Context, pluginID string) (*PluginFailure, error)) *RetryPlugin {
	return &RetryPlugin{retry: retry}
}

func (u *RetryPlugin) Execute(ctx context.Context, pluginID string) (*PluginFailure, error) {
	return u.retry(ctx, pluginID)
}
