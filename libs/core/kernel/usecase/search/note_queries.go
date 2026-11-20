package search

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NameMatches finds notes by a part of their name.
type NameMatches interface {
	GetNameMatches(ctx context.Context, vaultID domain.VaultID, query string, limit int) ([]domain.NameMatch, error)
}
