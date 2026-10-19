package plugin

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

const pingTimeout = 3 * time.Second

// checkPluginProtocol pings a started plugin and returns port.ErrUnsupportedProtocol when the protocol
// it reports differs from the one its manifest declares or from port.PluginProtocolVersion.
func checkPluginProtocol(ctx context.Context, m Manifest, endpoint Endpoint) error {
	client := numenv1connect.NewPluginLifecycleServiceClient(
		EndpointClient(endpoint),
		endpoint.BaseURL(),
		connect.WithInterceptors(authInterceptor(endpoint.Token)),
	)
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	resp, err := client.Ping(pingCtx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		return fmt.Errorf("ping plugin %s: %w", m.ID, err)
	}
	reported := int(resp.Msg.GetProtocol())
	if declared := m.GetProtocol(); reported != declared {
		return fmt.Errorf("%w: plugin %s speaks protocol %d, its manifest declares %d",
			port.ErrUnsupportedProtocol, m.ID, reported, declared)
	}
	if reported != port.PluginProtocolVersion {
		return fmt.Errorf("%w: plugin %s speaks protocol %d, this host speaks %d",
			port.ErrUnsupportedProtocol, m.ID, reported, port.PluginProtocolVersion)
	}
	return nil
}
