package version

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Statements is every statement of the package by name, for the query plan test.
func Statements() map[string]string { return sqlfile.Load(files, "sql").All() }

// FindContentHash is the hash the sources table holds for the file at the path,
// with the stat it was taken at, and false where the table has no row of it.
func (r *Repository) FindContentHash(ctx context.Context, vaultID domain.VaultID, path string) (textport.ContentHash, bool, error) {
	var size, modifiedAt int64
	var fileID, hashTime sql.NullInt64
	record := textport.ContentHash{Path: path}
	err := r.read.QueryRowContext(ctx, getStatement("find_content_hash"), string(vaultID), path).
		Scan(&record.Hash, &size, &modifiedAt, &fileID, &hashTime)
	if errors.Is(err, sql.ErrNoRows) {
		return textport.ContentHash{}, false, nil
	}
	if err != nil {
		return textport.ContentHash{}, false, fmt.Errorf("find the content hash of %s: %w", path, err)
	}
	record.Size, record.ModTime = size, chunk.ConvertNanosToTime(modifiedAt)
	if fileID.Valid {
		record.FileID = domain.FileID(fileID.Int64)
	}
	if hashTime.Valid {
		record.HashTime = chunk.ConvertNanosToTime(hashTime.Int64)
	}
	return record, true, nil
}

// SaveContentHash records the hash of a file on its row of the sources table,
// where the row holds the record's size and modification time or already holds
// its hash. A file with no row is not recorded.
func (r *Repository) SaveContentHash(ctx context.Context, vaultID domain.VaultID, record textport.ContentHash) error {
	modifiedAt := chunk.ConvertTimeToNanos(record.ModTime)
	_, err := sqlwrite.Exec(ctx, r.write, getStatement("save_content_hash"),
		record.Hash, record.Size, modifiedAt, int64(record.FileID), chunk.ConvertTimeToNanos(record.HashTime),
		string(vaultID), record.Path, record.Size, modifiedAt, record.Hash)
	if err != nil {
		return fmt.Errorf("save the content hash of %s: %w", record.Path, err)
	}
	return nil
}
