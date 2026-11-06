package hostendpoint

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

var errNotImplemented = errors.New("not implemented")

func (e *vaultEndpoint) AddLinks(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceAddLinksRequest],
) (*connect.Response[v1.HostVaultServiceAddLinksResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotImplemented)
}

func (e *vaultEndpoint) UpdateLinks(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceUpdateLinksRequest],
) (*connect.Response[v1.HostVaultServiceUpdateLinksResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotImplemented)
}

func (e *vaultEndpoint) RemoveLinks(
	ctx context.Context, req *connect.Request[v1.HostVaultServiceRemoveLinksRequest],
) (*connect.Response[v1.HostVaultServiceRemoveLinksResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errNotImplemented)
}
