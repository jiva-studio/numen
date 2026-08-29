package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Every half takes the kinds of source a question is about. None of them is
// every kind, which is what a question that says nothing about the sort of file
// it wants asks for.
//
// PassageQueries is the two indexes a search runs over. Both are built on the
// chunk, so a hit from either half names the same kind of row and the two
// answers can be placed in one list.
//
// Neither half returns the text: a passage is a place in a file, and the file is
// what holds the words.
type PassageQueries interface {
	// Lexical is the chunks of one vault whose text matches the words typed,
	// best first, at most `limit` of them. `growing` says the last word may
	// still be being typed, and is then matched by its opening.
	Lexical(ctx context.Context, vaultID, query string, of []domain.SourceKind, limit int, growing bool) ([]domain.Passage, error)

	// Named is the sections of one vault whose names match the words typed,
	// best first. A section answers with the chunk it opens, so what comes back
	// stands where the section begins.
	Named(ctx context.Context, vaultID, query string, of []domain.SourceKind, limit int, growing bool) ([]domain.Passage, error)

	// Nearest is the chunks of one vault nearest a query vector, nearest first,
	// at most `limit` of them. `query` is the full precision the model answered
	// with, and a chunk whose similarity to it is under `floor` is not an answer
	// and does not come back.
	//
	// `recipe` is what a vector is kept under, which is everything about the
	// model that decides what a vector is. A name alone finds none of them.
	Nearest(ctx context.Context, vaultID, recipe string, query []float32, of []domain.SourceKind, limit int, floor float64) ([]domain.Passage, error)
}
