package version

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/note/domain"
)


// FindContentHash returns the stored hash and stat for the file at the path, and
// false when no row exists.
func (r *Repository) FindContentHash(ctx context.Context, vaultID domain.VaultID, path string) (text.ContentHash, bool, error) {
	var size, modifiedAt int64
	var fileID, hashTime sql.NullInt64
	record := text.ContentHash{Path: path}
	err := r.read.QueryRowContext(ctx, getStatement("find_content_hash"), string(vaultID), path).
		Scan(&record.Hash, &size, &modifiedAt, &fileID, &hashTime)
	if errors.Is(err, sql.ErrNoRows) {
		return text.ContentHash{}, false, nil
	}
	if err != nil {
		return text.ContentHash{}, false, fmt.Errorf("find the content hash of %s: %w", path, err)
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
// where the row has the record's size and modification time or already has its
// hash. A file with no row is not recorded.
func (r *Repository) SaveContentHash(ctx context.Context, vaultID domain.VaultID, record text.ContentHash) error {
	modifiedAt := chunk.ConvertTimeToNanos(record.ModTime)
	_, err := sqlwrite.Exec(ctx, r.write, getStatement("save_content_hash"),
		record.Hash, record.Size, modifiedAt, int64(record.FileID), chunk.ConvertTimeToNanos(record.HashTime),
		string(vaultID), record.Path, record.Size, modifiedAt, record.Hash)
	if err != nil {
		return fmt.Errorf("save the content hash of %s: %w", record.Path, err)
	}
	return nil
}
