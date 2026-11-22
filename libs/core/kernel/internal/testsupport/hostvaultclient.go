package testsupport

import (
	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// HostVaultClient calls the host vault services of one endpoint as a plugin does.
type HostVaultClient struct {
	numenv1connect.HostVaultServiceClient
	numenv1connect.HostFileServiceClient
	numenv1connect.HostNoteServiceClient
	numenv1connect.HostLinkServiceClient
	numenv1connect.HostCacheServiceClient
}

// NewHostVaultClient is the client of the host vault services served at url.
func NewHostVaultClient(httpClient connect.HTTPClient, url string, options ...connect.ClientOption) HostVaultClient {
	return HostVaultClient{
		HostVaultServiceClient: numenv1connect.NewHostVaultServiceClient(httpClient, url, options...),
		HostFileServiceClient:  numenv1connect.NewHostFileServiceClient(httpClient, url, options...),
		HostNoteServiceClient:  numenv1connect.NewHostNoteServiceClient(httpClient, url, options...),
		HostLinkServiceClient:  numenv1connect.NewHostLinkServiceClient(httpClient, url, options...),
		HostCacheServiceClient: numenv1connect.NewHostCacheServiceClient(httpClient, url, options...),
	}
}
