package vaultchecks

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type CountFunc func(ctx context.Context, v domain.Vault, path string) (int, error)

type HashFunc func(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error)

// FileCounts counts the files at a path with the function it is given.
type FileCounts struct{ count CountFunc }

func NewFileCounts(count CountFunc) FileCounts { return FileCounts{count: count} }

// Count returns one for a file and the files under it for a folder.
func (c FileCounts) Count(ctx context.Context, v domain.Vault, path string) (int, error) {
	return c.count(ctx, v, path)
}

// FileHashes reads the stat of a file with the hash of its content with the function it is given.
type FileHashes struct{ hash HashFunc }

func NewFileHashes(hash HashFunc) FileHashes { return FileHashes{hash: hash} }

// Hash returns the stat of the file at path with the hash of its content.
func (h FileHashes) Hash(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error) {
	return h.hash(ctx, v, path)
}
