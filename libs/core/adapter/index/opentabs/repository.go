// Package opentabs stores the tabs the window has open on each vault in the index database.
package opentabs

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// Repository is the open tabs of every vault. Reads go through read and writes
// through write, the pools the index database opens.
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// NewRepository is the open tabs over the index database's two pools.
func NewRepository(read, write *sql.DB) *Repository {
	return &Repository{read: read, write: write}
}

// ReadOpenTabs returns the tabs last written for the vault, and none where nothing was written.
func (r *Repository) ReadOpenTabs(ctx context.Context, v domain.Vault) (domain.OpenTabs, error) {
	rows, err := r.read.QueryContext(ctx, stmt.Get("list"), string(v.ID))
	if err != nil {
		return domain.OpenTabs{}, fmt.Errorf("read the open tabs of %s: %w", v.ID, err)
	}
	defer rows.Close()
	var open domain.OpenTabs
	for rows.Next() {
		var tab domain.Tab
		var isFront bool
		if err := rows.Scan(&tab.ID, &tab.Kind, &tab.Path, &tab.Title, &tab.State, &isFront); err != nil {
			return domain.OpenTabs{}, fmt.Errorf("read the open tabs of %s: %w", v.ID, err)
		}
		if isFront {
			open.ActiveID = tab.ID
		}
		open.Tabs = append(open.Tabs, tab)
	}
	if err := rows.Err(); err != nil {
		return domain.OpenTabs{}, fmt.Errorf("read the open tabs of %s: %w", v.ID, err)
	}
	return open, nil
}

// WriteOpenTabs replaces the tabs kept for the vault in one transaction.
func (r *Repository) WriteOpenTabs(ctx context.Context, v domain.Vault, open domain.OpenTabs) error {
	tx, err := sqlwrite.Begin(ctx, r.write)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, stmt.Get("clear"), string(v.ID)); err != nil {
		return fmt.Errorf("clear the open tabs of %s: %w", v.ID, err)
	}
	for position, tab := range open.Tabs {
		isFront := tab.ID != "" && tab.ID == open.ActiveID
		if _, err := tx.ExecContext(ctx, stmt.Get("save"),
			string(v.ID), position, tab.ID, tab.Kind, tab.Path, tab.Title, tab.State, isFront); err != nil {
			return fmt.Errorf("save the open tab %s of %s: %w", tab.ID, v.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
