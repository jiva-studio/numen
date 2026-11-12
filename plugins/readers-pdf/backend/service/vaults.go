package service

import (
	"errors"
	"net/http"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

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
