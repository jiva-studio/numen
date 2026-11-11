package books

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/vault"
)

type vaultKey struct{}

// WithVault returns a context whose calls read the vault id names; an empty id is the vault the
// host shows.
func WithVault(ctx context.Context, id domain.VaultID) context.Context {
	return context.WithValue(ctx, vaultKey{}, id)
}

func getVault(ctx context.Context) domain.VaultID {
	if id, ok := ctx.Value(vaultKey{}).(domain.VaultID); ok {
		return id
	}
	return ""
}

func readVaultError(path string, err error) error {
	if errors.Is(err, host.ErrOutside) {
		return server.NewAPIError(server.CodePermissionDenied, fmt.Sprintf("path traversal outside allowed scope: %v", vault.ErrOutsideVault))
	}
	if errors.Is(err, host.ErrNoVault) {
		return server.NewAPIError(server.CodePermissionDenied, fmt.Sprintf("read book %s: %v", path, err))
	}
	return server.NewAPIError(server.CodeNotFound, fmt.Sprintf("read book %s: %v", path, err))
}
