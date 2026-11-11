package transport

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const pingTimeout = 3 * time.Second

// CheckPluginProtocol pings a started plugin and returns port.ErrUnsupportedProtocol when the protocol
// it reports differs from the one its manifest declares or from manifest.ProtocolVersion.
func CheckPluginProtocol(ctx context.Context, httpClient *http.Client, m manifest.Manifest, endpoint Endpoint) error {
	client := numenv1connect.NewPluginLifecycleServiceClient(
		httpClient,
		GetBaseURL(endpoint),
		connect.WithInterceptors(AuthInterceptor(endpoint.Token)),
	)
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	resp, err := client.Ping(pingCtx, connect.NewRequest(&v1.PingRequest{}))
	if err != nil {
		return fmt.Errorf("ping plugin %s: %w", m.ID, err)
	}
	reported := int(resp.Msg.GetProtocol())
	if declared := m.GetProtocol(); reported != declared {
		return fmt.Errorf("%w: plugin %s uses protocol %d, its manifest declares %d",
			manifest.ErrUnsupportedProtocol, m.ID, reported, declared)
	}
	if reported != manifest.ProtocolVersion {
		return fmt.Errorf("%w: plugin %s uses protocol %d, this host uses %d",
			manifest.ErrUnsupportedProtocol, m.ID, reported, manifest.ProtocolVersion)
	}
	return nil
}
