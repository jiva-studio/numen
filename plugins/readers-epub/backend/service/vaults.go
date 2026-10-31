package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
)

type requestVaultKey struct{}

// withRequestVault returns ctx carrying the vault the request's Numen-Vault header names.
func withRequestVault(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, requestVaultKey{}, host.GetRequestVault(header))
}

// getRequestVault returns the vault ctx carries, and empty where it names none.
func getRequestVault(ctx context.Context) host.VaultID {
	if id, ok := ctx.Value(requestVaultKey{}).(host.VaultID); ok {
		return id
	}
	return ""
}

// getVault returns the vault ctx names, and the shown vault where it names none.
func getVault(ctx context.Context, files host.Vault) host.Vault {
	if id := getRequestVault(ctx); id != "" {
		return files.With(id)
	}
	return files
}

// readVaultError turns a host vault error into an API error.
func readVaultError(path string, err error) error {
	if errors.Is(err, host.ErrOutside) {
		return server.NewAPIError(server.CodePermissionDenied, fmt.Sprintf("path traversal outside allowed scope: %v", vault.ErrOutsideVault))
	}
	if errors.Is(err, host.ErrNoVault) {
		return server.NewAPIError(server.CodePermissionDenied, fmt.Sprintf("read book %s: %v", path, err))
	}
	return server.NewAPIError(server.CodeNotFound, fmt.Sprintf("read book %s: %v", path, err))
}
