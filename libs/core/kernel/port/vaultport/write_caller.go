package vaultport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type callerKey struct{}

// WithCaller returns a context that names the plugin on whose behalf a write is made. A context without a caller is
// the window.
func WithCaller(ctx context.Context, pluginID string) context.Context {
	return context.WithValue(ctx, callerKey{}, pluginID)
}

// GetCaller returns the id of the plugin that makes the write, and an empty string for the window.
func GetCaller(ctx context.Context) string {
	pluginID, _ := ctx.Value(callerKey{}).(string)
	return pluginID
}

// CheckWrite refuses a write to a vault that another plugin manages. The caller is the one WithCaller named.
func CheckWrite(ctx context.Context, v domain.Vault) error {
	return vault.CheckWrite(v.ManagedBy, GetCaller(ctx))
}
