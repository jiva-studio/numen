package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/vault"
)

type requestVaultKey struct{}

func withRequestVault(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, requestVaultKey{}, host.GetRequestVault(header))
}

func getRequestVault(ctx context.Context) domain.VaultID {
	if id, ok := ctx.Value(requestVaultKey{}).(domain.VaultID); ok {
		return id
	}
	return ""
}

// vaultFiles is what the text server reads of a vault: its files and its name.
type vaultFiles interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
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
