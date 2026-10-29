package service

import (
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// getVault scopes files to the vault named by the request's Numen-Vault header when present.
func getVault(files host.Vault, header http.Header) host.Vault {
	if id := host.GetRequestVault(header); id != "" {
		return files.With(id)
	}
	return files
}

// readVaultError converts a host vault error into an API error.
func readVaultError(err error) error {
	if errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return server.NewAPIError(server.CodeNotFound, err.Error())
}

// getStatus returns the HTTP status code for a host vault error.
func getStatus(err error) int {
	if errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return http.StatusForbidden
	}
	return http.StatusNotFound
}
