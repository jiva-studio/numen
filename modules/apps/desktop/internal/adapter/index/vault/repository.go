// Package vault stores the vaults the index knows about.
package vault

import (
	"context"
	"database/sql"
	"embed"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Repository is the collection of vaults. Rows elsewhere point at these, which
// is why a vault is saved before anything is stored for it.
type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Save(ctx context.Context, v domain.Vault) error {
	_, err := r.db.ExecContext(ctx, stmt.Get("save"), v.ID, v.Name, v.Path)
	return err
}
