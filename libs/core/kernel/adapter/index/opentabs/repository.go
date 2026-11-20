// Package opentabs stores the editor workspace of each vault.
package opentabs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const workspaceKey = "editor.workspace"

// Repository reads and writes the editor workspace as one vault state entry.
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// NewRepository is the workspace store over the index database's two pools.
func NewRepository(read, write *sql.DB) *Repository {
	return &Repository{read: read, write: write}
}

// ReadOpenTabs returns the last workspace written for the vault.
func (r *Repository) ReadOpenTabs(ctx context.Context, v domain.Vault) (windowdomain.OpenTabs, error) {
	var value []byte
	err := r.read.QueryRowContext(ctx, listSQL, string(v.ID), workspaceKey).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return windowdomain.OpenTabs{}, nil
	}
	if err != nil {
		return windowdomain.OpenTabs{}, fmt.Errorf("read editor workspace of %s: %w", v.ID, err)
	}
	var saved workspace
	if err := json.Unmarshal(value, &saved); err != nil {
		return windowdomain.OpenTabs{}, fmt.Errorf("decode editor workspace of %s: %w", v.ID, err)
	}
	return saved.getOpenTabs(), nil
}

// WriteOpenTabs stores the tabs, selection and pane layout together.
func (r *Repository) WriteOpenTabs(ctx context.Context, v domain.Vault, open windowdomain.OpenTabs) error {
	value, err := json.Marshal(newWorkspace(open))
	if err != nil {
		return fmt.Errorf("encode editor workspace of %s: %w", v.ID, err)
	}
	if _, err := sqlwrite.Exec(ctx, r.write, saveSQL, string(v.ID), workspaceKey, string(value)); err != nil {
		return fmt.Errorf("write editor workspace of %s: %w", v.ID, err)
	}
	return nil
}
