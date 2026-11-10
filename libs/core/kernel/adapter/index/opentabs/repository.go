// Package opentabs stores the tabs the window has open on each vault in the index database.
package opentabs

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

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
func (r *Repository) ReadOpenTabs(ctx context.Context, v domain.Vault) (_ windowdomain.OpenTabs, err error) {
	rows, err := r.read.QueryContext(ctx, listSQL, string(v.ID))
	if err != nil {
		return windowdomain.OpenTabs{}, fmt.Errorf("read the open tabs of %s: %w", v.ID, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var open windowdomain.OpenTabs
	for rows.Next() {
		var tab windowdomain.Tab
		var isFront bool
		if err := rows.Scan(&tab.ID, &tab.Kind, &tab.Path, &tab.Title, &tab.State, &isFront); err != nil {
			return windowdomain.OpenTabs{}, fmt.Errorf("read the open tabs of %s: %w", v.ID, err)
		}
		if isFront {
			open.ActiveID = tab.ID
		}
		open.Tabs = append(open.Tabs, tab)
	}
	if err := rows.Err(); err != nil {
		return windowdomain.OpenTabs{}, fmt.Errorf("read the open tabs of %s: %w", v.ID, err)
	}
	return open, nil
}

// WriteOpenTabs replaces the tabs kept for the vault in one transaction.
func (r *Repository) WriteOpenTabs(ctx context.Context, v domain.Vault, open windowdomain.OpenTabs) error {
	return sqlwrite.InTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		if _, err := tx.ExecContext(ctx, getStatement("clear"), string(v.ID)); err != nil {
			return fmt.Errorf("clear the open tabs of %s: %w", v.ID, err)
		}
		for position, tab := range open.Tabs {
			isFront := tab.ID != "" && tab.ID == open.ActiveID
			if _, err := tx.ExecContext(ctx, getStatement("save"),
				string(v.ID), position, tab.ID, tab.Kind, tab.Path, tab.Title, tab.State, isFront); err != nil {
				return fmt.Errorf("save the open tab %s of %s: %w", tab.ID, v.ID, err)
			}
		}
		return nil
	})
}
