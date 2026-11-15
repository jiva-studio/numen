package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// LexicalQueries is the word and section-name indexes a search runs over. Both are built on
// the chunk, so a hit from either has the same row type and the results merge.
// Every method takes the source kinds a question is about; there is no wildcard
// value, so a question that does not restrict the kind passes every kind.
// Neither returns the text: a search result is a location in a file.
type LexicalQueries interface {
	// FindLexical returns the chunks of one vault whose text matches the words of `q.Text`,
	// best first, at most `q.Limit` of them.
	FindLexical(ctx context.Context, vaultID domain.VaultID, q source.LexicalQuery) ([]domain.SearchResult, error)

	// GetNamedSections is the sections of one vault whose names match the words
	// typed, best first. A section is returned as the chunk it opens, so each
	// result is located where the section begins.
	GetNamedSections(ctx context.Context, vaultID domain.VaultID, q source.LexicalQuery) ([]domain.SearchResult, error)
}
