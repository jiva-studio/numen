// Package vaultprovider serves a vault provider over VaultProviderService: the plugin that keeps vaults
// remotely. The core calls it to list the remote vaults and to connect one to a folder of this device,
// and names no plugin to do so. A plugin that serves it declares the capability vault.provider.
package vaultprovider

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// RemoteVault is a vault the plugin keeps remotely. VaultID is the vault of this device it is connected
// to, and empty when it is connected to none.
//
//numen:api
type RemoteVault struct {
	// RemoteID is the identity the plugin gives the remote vault.
	RemoteID string
	// Name is the name the person sees.
	Name string
	// SizeBytes is how many bytes the remote vault holds.
	SizeBytes int64
	// VaultID is the vault of this device the remote vault is connected to.
	VaultID domain.VaultID
}

// Handler is what a plugin implements to provide remote vaults. A failure is answered to the client with
// its code and message.
type Handler interface {
	// ListRemoteVaults lists the remote vaults this device may connect.
	ListRemoteVaults(ctx context.Context) ([]RemoteVault, error)
	// ConnectRemoteVault connects the remote vault to the vault of this device whose folder exists and is
	// registered, and returns once the connection is made.
	ConnectRemoteVault(ctx context.Context, remoteID string, vaultID domain.VaultID) error
}

type adapter struct {
	numenv1connect.UnimplementedVaultProviderServiceHandler
	h Handler
}

func (a *adapter) ListRemoteVaults(
	ctx context.Context, _ *connect.Request[v1.ListRemoteVaultsRequest],
) (*connect.Response[v1.ListRemoteVaultsResponse], error) {
	listed, err := a.h.ListRemoteVaults(ctx)
	if err != nil {
		return nil, err
	}
	out := &v1.ListRemoteVaultsResponse{Vaults: make([]*v1.RemoteVault, 0, len(listed))}
	for _, one := range listed {
		out.Vaults = append(out.Vaults, &v1.RemoteVault{
			RemoteId: one.RemoteID, Name: one.Name, SizeBytes: one.SizeBytes, VaultId: string(one.VaultID),
		})
	}
	return connect.NewResponse(out), nil
}

func (a *adapter) ConnectRemoteVault(
	ctx context.Context, req *connect.Request[v1.ConnectRemoteVaultRequest],
) (*connect.Response[v1.ConnectRemoteVaultResponse], error) {
	if err := a.h.ConnectRemoteVault(ctx, req.Msg.GetRemoteId(), domain.VaultID(req.Msg.GetVaultId())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.ConnectRemoteVaultResponse{}), nil
}

// NewHandler returns the path and the handler of the vault provider service over h.
func NewHandler(h Handler) (string, http.Handler) {
	return numenv1connect.NewVaultProviderServiceHandler(&adapter{h: h})
}

// Mount registers the vault provider service of h on s.
func Mount(s *server.Server, h Handler) {
	s.RegisterAuthenticated(NewHandler(h))
}
