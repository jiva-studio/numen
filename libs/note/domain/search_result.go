package domain

// ChunkID names one chunk of one source. The index assigns and resolves it; nothing
// here interprets its spelling.
type ChunkID string

// SearchResult is what a search returns: the text around a hit, and where it came from.
// It is a read model; a chunk is not reconstructed from it.
//
// `Start` and `Length` address the large chunk enclosing the hit, in bytes of the file
// `Path` names, and `Text` is the content there once the file has been read.
//
// `Similarity` is the cosine similarity of the query with the best small chunk of this place
// that the search by meaning returned, and `WordScore` the negated bm25 of the best hit of the
// words in this place, positive and comparable only within one query. Each is nil when its
// search did not reach the place; a similarity of zero or below is a value.
type SearchResult struct {
	ChunkID ChunkID

	Path string

	Kind SourceKind

	Start  int
	Length int

	PlaceLabel string

	VersionID string

	Selector Selector

	FileHash string

	MatchOffset int

	MatchLine int

	ChunkHash string

	Text string

	Similarity *float64

	WordScore *float64
}
