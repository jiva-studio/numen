// Package absent stores the registry of the files a vault lists and does not hold.
package absent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

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
func (r *Repository) PutAbsentFiles(ctx context.Context, v domain.Vault, files []vault.AbsentFile) error {
	err := sqlwrite.RunInTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		for _, f := range files {
			if _, err := tx.ExecContext(ctx, putSQL, string(v.ID), f.Path, f.Size, f.Mtime.UnixNano(), f.Hash, f.Provider); err != nil {
				return fmt.Errorf("put %s: %w", f.Path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("put absent files of %s: %w", v.ID, err)
	}
	return nil
}

// ListAbsentFiles returns every entry of the vault, by path.
func (r *Repository) ListAbsentFiles(ctx context.Context, v domain.Vault) (files []vault.AbsentFile, err error) {
	rows, err := r.read.QueryContext(ctx, listSQL, string(v.ID))
	if err != nil {
		return nil, fmt.Errorf("list absent files of %s: %w", v.ID, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	for rows.Next() {
		f, scanErr := scan(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("list absent files of %s: %w", v.ID, scanErr)
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list absent files of %s: %w", v.ID, err)
	}
	return files, nil
}

// GetAbsentFile returns the entry of the path, and false where there is none.
func (r *Repository) GetAbsentFile(ctx context.Context, v domain.Vault, path string) (vault.AbsentFile, bool, error) {
	f, err := scan(r.read.QueryRowContext(ctx, getSQL, string(v.ID), path))
	if errors.Is(err, sql.ErrNoRows) {
		return vault.AbsentFile{}, false, nil
	}
	if err != nil {
		return vault.AbsentFile{}, false, fmt.Errorf("get absent file %s of %s: %w", path, v.ID, err)
	}
	return f, true, nil
}

func (r *Repository) RemoveAbsentFiles(ctx context.Context, v domain.Vault, paths []string) error {
	err := sqlwrite.RunInTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		for _, p := range paths {
			if _, err := tx.ExecContext(ctx, removeSQL, string(v.ID), p); err != nil {
				return fmt.Errorf("remove %s: %w", p, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("remove absent files of %s: %w", v.ID, err)
	}
	return nil
}

type scanner interface{ Scan(dest ...any) error }

func scan(s scanner) (vault.AbsentFile, error) {
	var f vault.AbsentFile
	var mtime int64
	if err := s.Scan(&f.Path, &f.Size, &mtime, &f.Hash, &f.Provider); err != nil {
		return vault.AbsentFile{}, err
	}
	f.Mtime = time.Unix(0, mtime).UTC()
	return f, nil
}
