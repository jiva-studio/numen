package plugin_test

import (
	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// hostVaultClient calls the host vault services of one endpoint as a plugin does.
type hostVaultClient struct {
	numenv1connect.HostVaultServiceClient
	numenv1connect.HostFileServiceClient
	numenv1connect.HostNoteServiceClient
	numenv1connect.HostLinkServiceClient
	numenv1connect.HostCacheServiceClient
	numenv1connect.VaultsServiceClient
}

func newHostVaultClient(httpClient connect.HTTPClient, url string, options ...connect.ClientOption) hostVaultClient {
	return hostVaultClient{
		HostVaultServiceClient: numenv1connect.NewHostVaultServiceClient(httpClient, url, options...),
		HostFileServiceClient:  numenv1connect.NewHostFileServiceClient(httpClient, url, options...),
		HostNoteServiceClient:  numenv1connect.NewHostNoteServiceClient(httpClient, url, options...),
		HostLinkServiceClient:  numenv1connect.NewHostLinkServiceClient(httpClient, url, options...),
		HostCacheServiceClient: numenv1connect.NewHostCacheServiceClient(httpClient, url, options...),
		VaultsServiceClient:    numenv1connect.NewVaultsServiceClient(httpClient, url, options...),
	}
}
