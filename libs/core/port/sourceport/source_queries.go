package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

type sourceTextQueries interface {
	GetSourceText(ctx context.Context, vaultID domain.VaultID, path string) (SourceText, bool, error)
	GetRecognisedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind) ([]SourceText, error)
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

// SourceText is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that recognition are kept under.
type SourceText struct {
	domain.FileStat
	Producer string
	Hash     string
}
