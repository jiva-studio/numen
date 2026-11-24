package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// FileCounts counts the files at a path for the checked move and remove of notes.
type FileCounts struct{ countFiles *file.CountFiles }

// NewFileCounts returns the counter that runs countFiles.
func NewFileCounts(countFiles *file.CountFiles) FileCounts { return FileCounts{countFiles: countFiles} }

// Count returns one for a file and the files under it for a folder.
func (c FileCounts) Count(ctx context.Context, v domain.Vault, path string) (int, error) {
	return c.countFiles.Execute(ctx, v, path)
}

// FileHashes reads the stat of a file with the hash of its content for the checked move and remove of notes.
type FileHashes struct{ hashFile *file.HashFile }

// NewFileHashes returns the hasher that runs hashFile.
func NewFileHashes(hashFile *file.HashFile) FileHashes { return FileHashes{hashFile: hashFile} }

// Hash returns the stat of the file at path with the hash of its content.
func (h FileHashes) Hash(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error) {
	return h.hashFile.Execute(ctx, v, path)
}
