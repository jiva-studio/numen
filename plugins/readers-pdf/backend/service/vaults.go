package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// vaultFiles is what the text server reads of a vault: its files and its name.
type vaultFiles interface {
	host.FileReader
	GetVault(ctx context.Context) (domain.Vault, error)
}

func readVaultError(err error) error {
	if errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return server.NewAPIError(server.CodePermissionDenied, err.Error())
	}
	return server.NewAPIError(server.CodeNotFound, err.Error())
}

func getStatus(err error) int {
	if errors.Is(err, host.ErrOutside) || errors.Is(err, host.ErrNoVault) {
		return http.StatusForbidden
	}
	return http.StatusNotFound
}
