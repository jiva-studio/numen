package supervision

import (
	"context"
	"crypto/subtle"
	"net/http"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// NoEndpoint is the endpoint of a plugin that has none.
func NoEndpoint() plugindomain.Endpoint {
	return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}
}

// IsNoEndpoint reports whether the endpoint is the one of a plugin that has none.
func IsNoEndpoint(e plugindomain.Endpoint) bool {
	return e.Host == "" && e.Port == 0
}

func tokensEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func createRPCShutdown(endpoint plugindomain.Endpoint, client *http.Client, gracePeriod time.Duration) func(context.Context) error {
	if IsNoEndpoint(endpoint) {
		return nil
	}
	return func(ctx context.Context) error {
		timeout := gracePeriod
		if timeout <= 0 {
			timeout = rpcShutdownGrace
		}
		rpcCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		client := numenv1connect.NewPluginLifecycleServiceClient(
			client,
			transport.GetBaseURL(endpoint),
			connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token)),
		)
		req := connect.NewRequest(&v1.ShutdownRequest{
			TimeoutSeconds: int32(gracePeriod.Seconds()),
		})
		_, err := client.Shutdown(rpcCtx, req)
		return err
	}
}
