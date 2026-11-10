package settings

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// ErrNoRetrier is a RetryPlugin made without a retry.
var ErrNoRetrier = errors.New("retry plugin: no retrier")

// RetryPlugin starts a plugin whose last start failed once more, and answers the failure where it fails again.
type RetryPlugin struct {
	retry func(ctx context.Context, pluginID string) (*plugin.Failure, error)
}

// NewRetryPlugin creates the use case over the host's retry. It rejects a nil retry.
func NewRetryPlugin(retry func(ctx context.Context, pluginID string) (*plugin.Failure, error)) (*RetryPlugin, error) {
	if retry == nil {
		return nil, ErrNoRetrier
	}
	return &RetryPlugin{retry: retry}, nil
}

// Execute starts the plugin once more and answers the failure where it fails again.
func (u *RetryPlugin) Execute(ctx context.Context, pluginID string) (*plugin.Failure, error) {
	return u.retry(ctx, pluginID)
}
