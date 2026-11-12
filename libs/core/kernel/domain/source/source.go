package source

import "github.com/jiva-studio/numen/libs/note/domain"

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

// CoarseQuery asks for the chunks of one vault whose one-bit vectors are nearest a query vector:
// the recipe the vectors are kept under, the vector, the path prefix the sources must start
// with (any where empty) and how many chunks to return at most.
type CoarseQuery struct {
	Recipe     string
	Vector     []float32
	PathPrefix string
	Limit      int
}

// StoredVectorQuery asks for the full-precision vectors of some chunks under a recipe, limited to
// sources of some kinds (all where empty).
type StoredVectorQuery struct {
	Recipe string
	Kinds  []domain.SourceKind
	Chunks []domain.ChunkID
}

// StoredVector is the full-precision vector kept for one chunk, one signed byte per dimension.
type StoredVector struct {
	Chunk  domain.ChunkID
	Vector []byte
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

// Text is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that recognition are kept under.
type Text struct {
	domain.FileStat
	Producer    string
	ContentHash string
}
