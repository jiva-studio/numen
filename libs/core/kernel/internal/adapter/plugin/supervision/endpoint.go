package supervision

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// CreateNoEndpoint is the endpoint of a plugin that has none.
func CreateNoEndpoint() transport.Endpoint {
	return transport.Endpoint{Host: "", Port: 0, Token: ""}
}

// IsNoEndpoint reports whether the endpoint is the one of a plugin that has none.
func IsNoEndpoint(e transport.Endpoint) bool {
	return e.Host == "" && e.Port == 0
}

func createRPCShutdown(endpoint transport.Endpoint, client *http.Client, gracePeriod time.Duration) func(context.Context) error {
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
			connect.WithInterceptors(transport.CreateAuthInterceptor(endpoint.Token)),
		)
		req := connect.NewRequest(&v1.ShutdownRequest{})
		_, err := client.Shutdown(rpcCtx, req)
		return err
	}
}
