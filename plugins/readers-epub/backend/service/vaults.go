package service

import (
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
)

// getVault returns the vault the request's Numen-Vault header names, and the shown vault where it
// names none.
func getVault(files host.Vault, header http.Header) host.Vault {
	if id := host.GetRequestVault(header); id != "" {
		return files.With(id)
	}
	return files
}

// readVaultError turns a host vault error into a Connect error.
func readVaultError(path string, err error) error {
	if errors.Is(err, host.ErrOutside) {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope: %w", vault.ErrOutsideVault))
	}
	if errors.Is(err, host.ErrNoVault) {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read book %s: %w", path, err))
	}
	return connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
}
