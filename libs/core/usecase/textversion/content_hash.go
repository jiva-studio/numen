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

// ErrNoVaultReader wraps the failure to open the reader of a vault. The failure
// to stat or read one file is returned without it.
var ErrNoVaultReader = errors.New("the files of the vault cannot be read")

// GetContentHash is domain.HashContent of the file at the path as it is now. A
// recorded hash whose stat equals the file's is returned without opening the
// file; otherwise the file is hashed from a stream and the hash is recorded
// with the stat.
func (u *Versions) GetContentHash(ctx context.Context, v domain.Vault, path string) (string, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrNoVaultReader, err)
	}
	stat, err := reader.StatFile(ctx, path)
	if err != nil {
		return "", err
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
	if err != nil {
		return "", err
	}
	return hash, nil
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
// its modification time kept (git's racy check).
func isRacy(record textport.ContentHash) bool {
	return record.HashTime.IsZero() || !record.ModTime.Before(record.HashTime.Add(-racyWindow))
}

// hashFile is domain.HashReader of the file at the path.
func hashFile(ctx context.Context, reader vaultport.VaultReader, path string) (hash string, err error) {
	file, err := reader.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}
