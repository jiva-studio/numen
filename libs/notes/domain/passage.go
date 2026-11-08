package domain

// ChunkID names one chunk of one source. The index assigns and resolves it; nothing
// here interprets its spelling.
type ChunkID string

// Passage is what a search returns: the text around a hit, and where it came from.
// It is a read model; a chunk is not reconstructed from it.
//
// `Start` and `Length` address the large chunk enclosing the hit, in bytes of the file
// `Source` names, and `Text` is the content there once the file has been read.
type Passage struct {
	ChunkID ChunkID

	Source string

	Kind SourceKind

	Start  int
	Length int

	Location string

	Producer string

	Selector Selector

	ContentHash string

	HitAt int

	Line int

	ChunkHash string

	Text string
}
