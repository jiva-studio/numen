package filetext

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// SourceRecords answers what the index recorded about each file when it last read
// it, in shapes that are not sources.
type SourceRecords interface {
	GetFileStats(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.FileStat, error)
	GetRecognizedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]source.Text, error)
}
