package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// SourcesUnder answers which sources the index holds at a path and beneath it.
type SourcesUnder interface {
	GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error)
}

// RemovalLookup answers what the index holds about the sources at and under a
// path that left the vault: where each sat, and the text it was read with.
type RemovalLookup interface {
	GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) ([]domain.FileStat, error)
	GetSourceText(ctx context.Context, vaultID domain.VaultID, path string) (source.Text, bool, error)
}

// SourceBacklog answers which sources of one kind still need their text cut.
type SourceBacklog interface {
	GetUnchunkedSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, limit int) ([]string, error)
	GetSourcesWithOtherRecipe(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, recipes []string, limit int) ([]string, error)
}
