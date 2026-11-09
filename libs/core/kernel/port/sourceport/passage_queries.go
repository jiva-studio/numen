package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// PassageQueries is the two indexes a search runs over. Both are built on the
// chunk, so a hit from either half has the same row type and the results merge.
// Every method takes the source kinds a question is about; there is no wildcard
// value, so a question that does not restrict the kind passes every kind.
// Neither half returns the text: a passage is a location in a file.
type PassageQueries interface {
	// FindLexical returns the chunks of one vault whose text matches the words of `q.Text`,
	// best first, at most `q.Limit` of them.
	FindLexical(ctx context.Context, vaultID domain.VaultID, q LexicalQuery) ([]domain.Passage, error)

	// GetNamedPassages is the sections of one vault whose names match the words
	// typed, best first. A section is returned as the chunk it opens, so each
	// result is located where the section begins.
	GetNamedPassages(ctx context.Context, vaultID domain.VaultID, q LexicalQuery) ([]domain.Passage, error)

	// FindNearest is the chunks of one vault nearest a query vector, nearest first,
	// at most `q.Limit` of them. `q.Vector` is the full precision the model answered
	// with, and a chunk whose similarity to it is under `q.Floor` is not an answer
	// and is not returned.
	//
	// `q.Recipe` is what a vector is kept under, which is everything about the
	// model that decides what a vector is. A name alone finds none of them.
	FindNearest(ctx context.Context, vaultID domain.VaultID, q NearestQuery) ([]domain.Passage, error)
}

// NearestQuery is a search by meaning: the vector the model answered with, the
// recipe it was made under, the kinds of source to look in (all where empty),
// how many chunks to return at most and the similarity under which a chunk is
// not an answer.
type NearestQuery struct {
	Recipe     string
	Vector     []float32
	Kinds      []domain.SourceKind
	PathPrefix string
	Limit      int
	Floor      float64
}

// LexicalQuery is a search by words: the words typed, the kinds of source to
// look in (all where empty) and how many chunks to return at most.
type LexicalQuery struct {
	Text       string
	Kinds      []domain.SourceKind
	PathPrefix string
	Limit      int
	IsGrowing  bool
}
