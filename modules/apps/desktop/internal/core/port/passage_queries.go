package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// PassageQueries is the two indexes a search runs over. Both are built on the
// chunk, so a hit from either half names the same kind of row and the two
// answers can be placed in one list.
//
// Neither half returns the text: a passage is a place in a file, and the file is
// what holds the words.
type PassageQueries interface {
	// Lexical is the chunks of one vault whose text matches the words typed,
	// best first, at most `limit` of them.
	Lexical(ctx context.Context, vaultID, query string, limit int) ([]domain.Passage, error)

	// Nearest is the chunks of one vault nearest a query vector, nearest first.
	// `coarse` is one bit per dimension, and `k` is how many candidates the pass
	// keeps.
	Nearest(ctx context.Context, vaultID string, coarse []byte, k int) ([]domain.Passage, error)
}
