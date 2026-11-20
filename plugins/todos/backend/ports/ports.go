// Package ports holds the small interfaces the use cases need from the host.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// FileReader reads the notes of the vault.
type FileReader interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	// Read returns the file with the file stat it had when it was read.
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
}

// NoteLister is the one call of host.NoteLookup that the index makes.
type NoteLister interface {
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}
