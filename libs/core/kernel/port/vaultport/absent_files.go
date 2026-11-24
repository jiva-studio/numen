package vaultport

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// AbsentFiles is the registry of the files a vault lists whose bytes are not on this device. It is
// derived data: a plugin sends it again after the index is rebuilt.
type AbsentFiles interface {
	// PutAbsentFiles replaces the entry of each path.
	PutAbsentFiles(ctx context.Context, v domain.Vault, files []vault.AbsentFile) error
	// ListAbsentFiles returns every entry of the vault, by path.
	ListAbsentFiles(ctx context.Context, v domain.Vault) ([]vault.AbsentFile, error)
	// GetAbsentFile returns the entry of the path, and false where there is none.
	GetAbsentFile(ctx context.Context, v domain.Vault, path string) (vault.AbsentFile, bool, error)
	// RemoveAbsentFiles deletes the entries of the paths.
	RemoveAbsentFiles(ctx context.Context, v domain.Vault, paths []string) error
}
