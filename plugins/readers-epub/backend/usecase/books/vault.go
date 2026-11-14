package books

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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
		return newError(KindPermissionDenied, "path traversal outside allowed scope: path outside allowed vault root")
	}
	if errors.Is(err, host.ErrNoVault) {
		return newError(KindPermissionDenied, fmt.Sprintf("read book %s: %v", path, err))
	}
	return newError(KindNotFound, fmt.Sprintf("read book %s: %v", path, err))
}
