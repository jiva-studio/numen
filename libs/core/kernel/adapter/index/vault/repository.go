// Package vault stores the vaults that the index tracks.
package vault

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

// Repository is the collection of vaults. Rows elsewhere point at these, which
// is why a vault is registered before anything is stored for it.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// Register creates a row for the vault that other rows reference. A vault the
// index already tracks keeps its existing row.
func (r *Repository) Register(ctx context.Context, vaultID domain.VaultID) error {
	_, err := sqlwrite.Exec(ctx, r.db, getStatement("register"), string(vaultID))
	return err
}

func getForgettingStatements() []string {
	return []string{
		"clear_vec", "clear_fts", "clear_sections", "clear_title_names", "clear_heading_names", "delete",
	}
}

// Unregister takes everything the index holds for one vault, and the vault's
// own row with it. A vault the index does not hold is already unregistered.
//
// The vectors stay. One is addressed by the text it was made from, so chunks of
// several vaults hold the same vector.
func (r *Repository) Unregister(ctx context.Context, vaultID domain.VaultID) error {
	return sqlwrite.InTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		var row int64
		err := tx.QueryRowContext(ctx, getStatement("vault_row"), string(vaultID)).Scan(&row)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("which row this vault is filed under: %w", err)
		}
		for _, name := range getForgettingStatements() {
			if _, err := tx.ExecContext(ctx, getStatement(name), row); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		return nil
	})
}
