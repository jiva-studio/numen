package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// IndexFollow reports whether the index followed a write.
type IndexFollow interface {
	// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
	IsIndexOutdated(err error) bool
}

// MovedFiles answers what a move touches: the sources the index holds at a path and beneath it, and the entries of
// absent files, which it moves and counts.
type MovedFiles interface {
	SourcesUnder
	MoveAbsent(ctx context.Context, v domain.Vault, from, to string) (int, error)
}

// SourcesUnder answers which sources the index holds at a path and beneath it.
type SourcesUnder interface {
	GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error)
}

// FileMovers opens a vault for moving and removing its files.
type FileMovers interface {
	OpenFileMover(v domain.Vault) (vaultport.FileMover, error)
}
