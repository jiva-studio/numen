package vault

import (
	"context"

	"connectrpc.com/connect"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

func getCallerOption(auth Authority) connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if pluginID, err := auth.Authenticate(req.Header().Get("Authorization")); err == nil {
				ctx = vaultdomain.WithCaller(ctx, pluginID)
			}
			return next(ctx, req)
		}
	}))
}
