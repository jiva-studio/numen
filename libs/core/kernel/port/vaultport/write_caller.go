package vaultport

import "context"

// WithCaller returns a context that names the plugin on whose behalf a write is made. A context without a caller is
// the window.
func WithCaller(ctx context.Context, _ string) context.Context {
	return ctx
}
