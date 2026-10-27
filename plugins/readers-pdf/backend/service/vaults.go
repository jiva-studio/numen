package service

import (
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// getVault scopes files to the vault named by the request's Numen-Vault header when present.
func getVault(files host.Vault, header http.Header) host.Vault {
	if id := host.GetRequestVault(header); id != "" {
		return files.With(id)
	}
	return files
}

// readVaultError converts a host vault error into a Connect error.
func readVaultError(err error) error {
	if errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeNotFound, err)
}

// getStatus returns the HTTP status code for a host vault error.
func getStatus(err error) int {
	if errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return http.StatusForbidden
	}
	return http.StatusNotFound
}
