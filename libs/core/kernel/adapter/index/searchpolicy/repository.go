// Package searchpolicy stores the effective similarity floor of each vault under
// each embedding recipe in the index database, and reads the samples a plugin
// measures the floor with.
package searchpolicy

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

// Repository is the floors of every vault. Reads go through read and writes
// through write, the pools the index database opens.
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// NewRepository is the floors over the index database's two pools.
func NewRepository(read, write *sql.DB) *Repository {
	return &Repository{read: read, write: write}
}

// ReadFloor returns the floor kept for the vault under the recipe, and false
// where none is kept.
func (r *Repository) ReadFloor(ctx context.Context, v domain.Vault, recipe string) (float64, bool, error) {
	var floor float64
	err := r.read.QueryRowContext(ctx, readSQL, string(v.ID), recipe).Scan(&floor)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read the floor of %s: %w", v.ID, err)
	}
	return floor, true, nil
}

// WriteFloor keeps the floor, replacing the one kept.
func (r *Repository) WriteFloor(ctx context.Context, v domain.Vault, recipe string, floor float64) error {
	if _, err := sqlwrite.Exec(ctx, r.write, getStatement("save"), string(v.ID), recipe, floor); err != nil {
		return fmt.Errorf("write the floor of %s: %w", v.ID, err)
	}
	return nil
}

func (r *Repository) DeleteFloor(ctx context.Context, v domain.Vault, recipe string) error {
	if _, err := sqlwrite.Exec(ctx, r.write, getStatement("delete"), string(v.ID), recipe); err != nil {
		return fmt.Errorf("delete the floor of %s: %w", v.ID, err)
	}
	return nil
}
