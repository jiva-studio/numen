package vault

import (
	"context"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// getCallerOption names the plugin of the token on the context of every unary call, so that a write the call makes is
// the write of that plugin.
func getCallerOption(auth Authority) connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if pluginID, err := auth.Authenticate(req.Header().Get("Authorization")); err == nil {
				ctx = vaultport.WithCaller(ctx, pluginID)
			}
			return next(ctx, req)
		}
	}))
}
