// Package vault stores the vaults the index knows about.
package vault

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Repository is the collection of vaults. Rows elsewhere point at these, which
// is why a vault is registered before anything is stored for it.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// Register gives the vault a row for other rows to point at. A vault the index
// already knows keeps the row it has.
func (r *Repository) Register(ctx context.Context, vaultID domain.VaultID) error {
	_, err := r.db.ExecContext(ctx, stmt.Get("register"), string(vaultID))
	return err
}

// forgetting is the order a vault is taken out in. The five virtual tables go
// first: nothing cascades into one, and the numbers four of them are addressed
// by are read from the tables the last statement takes away.
var forgetting = []string{
	"clear_vec", "clear_fts", "clear_parts", "clear_title_names", "clear_heading_names", "delete",
}

// Forget takes everything the index holds for one vault, and the vault's own
// row with it. A vault the index does not hold is already forgotten.
//
// The vectors stay. One is addressed by the text it was made from, so chunks of
// several vaults hold the same vector.
func (r *Repository) Forget(ctx context.Context, vaultID domain.VaultID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	var row int64
	err = tx.QueryRowContext(ctx, stmt.Get("vault_row"), string(vaultID)).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("which row this vault is filed under: %w", err)
	}
	for _, name := range forgetting {
		if _, err := tx.ExecContext(ctx, stmt.Get(name), row); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
