// Package providerwire turns the answers of the backend into what the core reads from a provider plugin.
package providerwire

import (
	"errors"
	"math"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/vaultprovider"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
)

// NewError returns the failure the core reads for the error of a backend method.
func NewError(err error) error {
	failure, ok := errors.AsType[*server.APIError](err)
	if !ok {
		return err
	}
	return rpcerror.New(getRPCCode(failure.Code), errors.New(failure.Message))
}

func getRPCCode(code server.ErrorCode) rpcerror.Code {
	if code == server.CodeFailedPrecondition {
		return rpcerror.FailedPrecondition
	}
	if code == server.CodeUnavailable {
		return rpcerror.Unavailable
	}
	return rpcerror.Internal
}

// IsOffline reports whether the plugin cannot reach its cloud: no server, no identity, no sign-in or an unreachable server.
func IsOffline(err error) bool {
	failure, ok := errors.AsType[*server.APIError](err)
	return ok && (failure.Code == server.CodeUnavailable || failure.Code == server.CodeFailedPrecondition)
}

// JoinRemoteVaults lists the cloud vaults, each with the local vault that is connected to it.
func JoinRemoteVaults(remote []api.RemoteVault, local []api.Vault) []vaultprovider.RemoteVault {
	connected := make(map[string]domain.VaultID, len(local))
	for _, vault := range local {
		if vault.RemoteID != "" {
			connected[vault.RemoteID] = domain.VaultID(vault.ID)
		}
	}
	listed := make([]vaultprovider.RemoteVault, len(remote))
	for i, one := range remote {
		listed[i] = vaultprovider.RemoteVault{
			RemoteID:  one.RemoteID,
			Name:      one.Name,
			SizeBytes: int64(min(one.SizeBytes, math.MaxInt64)),
			VaultID:   connected[one.RemoteID],
		}
	}
	return listed
}
