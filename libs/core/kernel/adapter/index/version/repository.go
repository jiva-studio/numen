// Package version stores the text versions of vault files in the index database.
package version

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

// Repository is the text versions of every vault. Reads go through read and
// writes through write, the pools the index database opens.
type Repository struct {
	read  *sql.DB
	write *sql.DB
}

// NewRepository is the versions over the index database's two pools.
func NewRepository(read, write *sql.DB) *Repository {
	return &Repository{read: read, write: write}
}

// GetVersions is the versions of one file, in the order they were first recorded.
func (r *Repository) GetVersions(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.TextVersion, error) {
	return r.query(ctx, getPathSQL, string(vaultID), path)
}

// GetVersionsUnder is the versions of a file, or of everything under a folder, in the order they were first recorded.
func (r *Repository) GetVersionsUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.TextVersion, error) {
	first, past := getRangeUnder(path)
	return r.query(ctx, listUnderSQL, string(vaultID), path, first, past)
}

// ListVersions is the versions of every file of a vault, in the order they were first recorded.
func (r *Repository) ListVersions(ctx context.Context, vaultID domain.VaultID) ([]domain.TextVersion, error) {
	return r.query(ctx, listSQL, string(vaultID))
}

// SaveVersion records a version, replacing the record of the same path, plugin and key in its place.
func (r *Repository) SaveVersion(ctx context.Context, vaultID domain.VaultID, v domain.TextVersion) error {
	_, err := sqlwrite.Exec(ctx, r.write, getStatement("save"),
		string(vaultID), v.Path, v.PluginID, v.Key, string(v.Source), v.SourceProperty.Key, v.SourceProperty.Value, string(v.Kind), v.File,
		v.InputHash, v.TextHash, v.CreateTime.UnixNano())
	if err != nil {
		return fmt.Errorf("save version %s of %s: %w", v.GetID(), v.Path, err)
	}
	return nil
}

// RemoveVersion removes the record of a version. A version that is not there is removed.
func (r *Repository) RemoveVersion(ctx context.Context, vaultID domain.VaultID, path, pluginID, key string) error {
	if _, err := sqlwrite.Exec(ctx, r.write, getStatement("remove"), string(vaultID), path, pluginID, key); err != nil {
		return fmt.Errorf("remove version %s/%s of %s: %w", pluginID, key, path, err)
	}
	return nil
}

// RemovePluginVersions removes every version a plugin made in a vault, and
// returns the records it removed.
func (r *Repository) RemovePluginVersions(ctx context.Context, vaultID domain.VaultID, pluginID string) ([]domain.TextVersion, error) {
	var removed []domain.TextVersion
	err := sqlwrite.InTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		listed, err := scan(tx.QueryContext(ctx, getStatement("list_plugin"), string(vaultID), pluginID))
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, getStatement("remove_plugin"), string(vaultID), pluginID); err != nil {
			return fmt.Errorf("remove the versions of %s: %w", pluginID, err)
		}
		removed = listed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return removed, nil
}

// MoveVersions re-files the versions of what was at one path, and of everything
// under it, at the path it now has. The versions of whatever was at the
// destination are removed.
func (r *Repository) MoveVersions(ctx context.Context, vaultID domain.VaultID, from, to string) error {
	if from == to {
		return nil
	}
	return sqlwrite.InTransaction(ctx, r.write, func(tx *sqlwrite.Transaction) error {
		toFirst, toPast := getRangeUnder(to)
		fromFirst, fromPast := getRangeUnder(from)
		if _, err := tx.ExecContext(ctx, getStatement("clear_target"),
			string(vaultID), to, toFirst, toPast, from, fromFirst, fromPast); err != nil {
			return fmt.Errorf("clear the versions at %s: %w", to, err)
		}
		// substr counts characters from one.
		fromLen := utf8.RuneCountInString(from) + 1
		if _, err := tx.ExecContext(ctx, getStatement("move"),
			to, fromLen, string(vaultID), from, fromFirst, fromPast); err != nil {
			return fmt.Errorf("move the versions of %s to %s: %w", from, to, err)
		}
		return nil
	})
}

func getRangeUnder(folder string) (first, past string) {
	return folder + "/", folder + "0"
}

func (r *Repository) query(ctx context.Context, query string, args ...any) ([]domain.TextVersion, error) {
	return scan(r.read.QueryContext(ctx, query, args...))
}

func scan(rows *sql.Rows, queryErr error) (out []domain.TextVersion, err error) {
	if queryErr != nil {
		return nil, fmt.Errorf("read versions: %w", queryErr)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	for rows.Next() {
		var v domain.TextVersion
		var source, kind string
		var created int64
		if err := rows.Scan(&v.Path, &v.PluginID, &v.Key, &source, &v.SourceProperty.Key, &v.SourceProperty.Value, &kind, &v.File, &v.InputHash, &v.TextHash, &created); err != nil {
			return nil, fmt.Errorf("read versions: %w", err)
		}
		v.Source, v.Kind, v.CreateTime = domain.TextVersionID(source), domain.TextKind(kind), time.Unix(0, created).UTC()
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read versions: %w", err)
	}
	return out, nil
}
