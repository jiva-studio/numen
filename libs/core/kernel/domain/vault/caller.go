package vault

import "context"

type callerKey struct{}

// WithCaller returns a context that names the plugin on whose behalf a write is made. A context without a caller is
// the window.
func WithCaller(ctx context.Context, pluginID string) context.Context {
	return context.WithValue(ctx, callerKey{}, pluginID)
}

// GetCaller returns the id of the plugin that makes the write, and an empty string for the window.
func GetCaller(ctx context.Context) string {
	pluginID, ok := ctx.Value(callerKey{}).(string)
	if !ok {
		return ""
	}
	return pluginID
}
