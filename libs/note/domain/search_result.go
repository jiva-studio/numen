package domain

// ChunkID names one chunk of one source. The index assigns and resolves it; nothing
// here interprets its spelling.
type ChunkID string

// SearchResult is what a search returns: the text around a hit, and where it came from.
// It is a read model; a chunk is not reconstructed from it.
//
// `Start` and `Length` address the large chunk enclosing the hit, in bytes of the file
// `Path` names, and `Text` is the content there once the file has been read.
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
}
