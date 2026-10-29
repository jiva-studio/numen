package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Every method takes the source kinds a question is about. The list has no
// wildcard value: a question that does not restrict the file kind passes every
// kind.
//
// PassageQueries is the two indexes a search runs over. Both are built on the
// chunk, so a hit from either half has the same row type and the two result
// sets can be merged into one list.
//
// Neither half returns the text: a passage is a location in a file, and the
// file holds the words.
type PassageQueries interface {
	// FindLexical is the chunks of one vault whose text matches the words typed,
	// best first, at most `limit` of them. `growing` marks the last word as possibly
	// still being typed, and it is then matched by prefix.
	FindLexical(ctx context.Context, vaultID domain.VaultID, query string, kinds []domain.SourceKind, limit int, growing bool) ([]domain.Passage, error)

	// GetNamedPassages is the sections of one vault whose names match the words
	// typed, best first. A section is returned as the chunk it opens, so each
	// result is located where the section begins.
	GetNamedPassages(ctx context.Context, vaultID domain.VaultID, query string, kinds []domain.SourceKind, limit int, growing bool) ([]domain.Passage, error)

	// FindNearest is the chunks of one vault nearest a query vector, nearest first,
	// at most `limit` of them. `query` is the full precision the model answered
	// with, and a chunk whose similarity to it is under `floor` is not an answer
	// and does not come back.
	//
	// `recipe` is what a vector is kept under, which is everything about the
	// model that decides what a vector is. A name alone finds none of them.
	FindNearest(ctx context.Context, vaultID domain.VaultID, recipe string, query []float32, kinds []domain.SourceKind, limit int, floor float64) ([]domain.Passage, error)
}
