// Package absent stores the registry of the files a vault lists and does not hold.
package absent

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

var errNotImplemented = errors.New("not implemented")

// Repository is the registry over the two pools of the index database.
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// NewRepository is the registry over the index database's two pools.
func NewRepository(read, write *sql.DB) *Repository {
	return &Repository{read: read, write: write}
}

// PutAbsentFiles replaces the entry of each path.
func (r *Repository) PutAbsentFiles(context.Context, domain.Vault, []vault.AbsentFile) error {
	return errNotImplemented
}

// ListAbsentFiles returns every entry of the vault, by path.
func (r *Repository) ListAbsentFiles(context.Context, domain.Vault) ([]vault.AbsentFile, error) {
	return nil, errNotImplemented
}

// GetAbsentFile returns the entry of the path, and false where there is none.
func (r *Repository) GetAbsentFile(context.Context, domain.Vault, string) (vault.AbsentFile, bool, error) {
	return vault.AbsentFile{}, false, errNotImplemented
}

// RemoveAbsentFiles deletes the entries of the paths.
func (r *Repository) RemoveAbsentFiles(context.Context, domain.Vault, []string) error {
	return errNotImplemented
}
