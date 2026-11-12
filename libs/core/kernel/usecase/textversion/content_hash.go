package textversion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// racyWindow is the coarsest modification time tick of a file system (FAT). A
// file changed this close to the time its hash was recorded can change again
// without its stat changing.
const racyWindow = 2 * time.Second

// GetContentHash returns domain.HashContent of the file at the path as it is now.
// A recorded hash whose stat equals the file's is returned without opening the
// file; otherwise the file is hashed from a stream and recorded with the stat.
// A file the vault ignores has no index row and is hashed without a record.
// The record is a cache: a hash that cannot be recorded is returned all the same.
func (u *Texts) GetContentHash(ctx context.Context, v domain.Vault, path string) (string, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	stat, err := reader.StatFile(ctx, path)
	if errors.Is(err, edit.ErrNotANote) {
		return hashFile(ctx, reader, path)
	}
	if err != nil {
		return "", markFileFailure(ctx, err)
	}
	hashes, err := u.stores.OpenContentHashes(v)
	if err != nil {
		return "", err
	}
	record, found, err := hashes.FindContentHash(ctx, path)
	if err != nil {
		return "", err
	}
	if found && isRecordCurrent(record, stat) {
		return record.Hash, nil
	}
	hash, err := hashFile(ctx, reader, path)
	if err != nil {
		return "", err
	}
	err = hashes.SaveContentHash(ctx, text.ContentHash{
		Path: path, Hash: hash, Size: stat.Size, ModTime: stat.ModTime, FileID: stat.FileID, HashTime: u.clock.Now(),
	})
	if err != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}
	return hash, nil
}

func markFileFailure(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %w", text.ErrFileUnreadable, err)
}

func isRecordCurrent(record text.ContentHash, stat domain.FileStat) bool {
	if record.Hash == "" || record.Size != stat.Size || !record.ModTime.Equal(stat.ModTime) || record.FileID != stat.FileID {
		return false
	}
	return !isRacy(record)
}

func isRacy(record text.ContentHash) bool {
	if record.HashTime.IsZero() {
		return true
	}
	return !record.ModTime.Before(record.HashTime.Add(-racyWindow)) && record.ModTime.Before(record.HashTime.Add(racyWindow))
}

func hashFile(ctx context.Context, reader vaultport.VaultReader, path string) (hash string, err error) {
	file, err := reader.Open(ctx, path)
	if err != nil {
		return "", markFileFailure(ctx, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			hash, err = "", errors.Join(err, markFileFailure(ctx, closeErr))
		}
	}()
	hash, err = domain.HashReader(file)
	if err != nil {
		return "", markFileFailure(ctx, err)
	}
	return hash, nil
}
