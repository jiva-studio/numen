package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type sourceTextQueries interface {
	GetSourceText(ctx context.Context, vaultID domain.VaultID, path string) (source.Text, bool, error)
	GetRecognisedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]source.Text, error)
}

// SourceQueries answers what the index holds about sources, in shapes that are
// not sources: what it believes each file was, and which of them need their text.
type SourceQueries interface {
	sourceTextQueries
	GetFileStats(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) (map[string]domain.FileStat, error)
	GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error)
	GetUnchunkedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error)
	GetSourcesWithOtherRecipe(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error)
}
