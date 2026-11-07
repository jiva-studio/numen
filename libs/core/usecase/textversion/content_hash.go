package textversion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// racyWindow is the coarsest modification time tick of a file system (FAT). A
// file changed this close to the time its hash was recorded can change again
// without its stat changing.
const racyWindow = 2 * time.Second

// ErrFileUnreadable wraps the failure to stat, open or read the file whose
// hash is requested. A failure of the vault, of the index or of the context is
// returned without it.
var ErrFileUnreadable = errors.New("the file cannot be read")

// GetContentHash returns domain.HashContent of the file at the path as it is now. A
// recorded hash whose stat equals the file's is returned without opening the
// file; otherwise the file is hashed from a stream and the hash is recorded
// with the stat. A file the vault ignores has no index row and is hashed
// without a record.
//
// The record is a cache: a hash that cannot be recorded is returned all the same.
func (u *Versions) GetContentHash(ctx context.Context, v domain.Vault, path string) (string, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	stat, err := reader.StatFile(ctx, path)
	if errors.Is(err, vaultport.ErrNotANote) {
		return hashFile(ctx, reader, path)
	}
	if err != nil {
		return "", markFileFailure(ctx, err)
	}
	repo, err := u.repos.Open(v)
	if err != nil {
		return "", err
	}
	record, found, err := repo.FindContentHash(ctx, path)
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
	err = repo.SaveContentHash(ctx, textport.ContentHash{
		Path: path, Hash: hash, Size: stat.Size, ModTime: stat.ModTime, FileID: stat.FileID, HashTime: u.clock.Now(),
	})
	if err != nil && ctx.Err() != nil {
		return "", ctx.Err()
	}
	return hash, nil
}

// markFileFailure wraps a failure of the file in ErrFileUnreadable, and
// returns the end of the context as it is.
func markFileFailure(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%w: %w", ErrFileUnreadable, err)
}

// isRecordCurrent reports whether the recorded hash is the hash of the file
// with the stat given.
func isRecordCurrent(record textport.ContentHash, stat domain.FileStat) bool {
	if record.Hash == "" || record.Size != stat.Size || !record.ModTime.Equal(stat.ModTime) || record.FileID != stat.FileID {
		return false
	}
	return !isRacy(record)
}

// isRacy reports whether the file may have changed after it was hashed with
// its modification time kept (git's racy check): its modification time is
// within one tick of the hash time. A modification time further ahead cannot be
// kept by a later write, which stamps the current time.
func isRacy(record textport.ContentHash) bool {
	if record.HashTime.IsZero() {
		return true
	}
	return !record.ModTime.Before(record.HashTime.Add(-racyWindow)) && record.ModTime.Before(record.HashTime.Add(racyWindow))
}

// hashFile returns domain.HashReader of the file at the path.
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
