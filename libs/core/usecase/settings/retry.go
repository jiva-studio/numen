package settings

import (
	"context"
	"errors"
)

// ErrNoRetrier is a RetryPlugin made without a retry.
var ErrNoRetrier = errors.New("retry plugin: no retrier")

// RetryPlugin starts a plugin the host suspends once more, and answers the failure where it fails again.
type RetryPlugin struct {
	retry func(ctx context.Context, pluginID string) (*PluginFailure, error)
}

// NewRetryPlugin makes the use case over the host's retry. It refuses a nil retry.
func NewRetryPlugin(retry func(ctx context.Context, pluginID string) (*PluginFailure, error)) (*RetryPlugin, error) {
	if retry == nil {
		return nil, ErrNoRetrier
	}
	return &RetryPlugin{retry: retry}, nil
}

// Execute starts the plugin once more and answers the failure where it fails again.
func (u *RetryPlugin) Execute(ctx context.Context, pluginID string) (*PluginFailure, error) {
	return u.retry(ctx, pluginID)
}
