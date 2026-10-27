package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
)

// vaultRoots finds the folder of the vault a request works.
type vaultRoots struct {
	files host.Vault
	// defaultRoot is the folder of the vault the plugin was started for, which a request naming no
	// vault works.
	defaultRoot func() string
}

// getRoot returns the folder of the vault the request's Numen-Vault header names, and the folder of
// the vault the plugin was started for where it names none.
func (r vaultRoots) getRoot(ctx context.Context, header http.Header) (string, error) {
	id := host.GetRequestVault(header)
	if id == "" {
		return r.defaultRoot(), nil
	}
	found, err := r.files.With(id).GetVault(ctx)
	if err != nil {
		return "", fmt.Errorf("find the vault %s: %w", id, err)
	}
	return found.Path, nil
}

// getFilePath returns the path on disk of a file of the vault the request works, and a Connect
// error where the vault is not found or the path leaves it.
func (r vaultRoots) getFilePath(ctx context.Context, header http.Header, path string) (string, error) {
	root, err := r.getRoot(ctx, header)
	if err != nil {
		return "", connect.NewError(connect.CodeNotFound, err)
	}
	cleanPath, err := vault.ValidatePathIn(root, path)
	if err != nil {
		return "", connect.NewError(connect.CodePermissionDenied, fmt.Errorf("path traversal outside allowed scope: %w", err))
	}
	return cleanPath, nil
}

// getStatus is the HTTP status of an error getFilePath returned.
func getStatus(err error) int {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodePermissionDenied {
		return http.StatusForbidden
	}
	return http.StatusNotFound
}
