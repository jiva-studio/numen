package domain

// ChunkID names one chunk of one source. The index hands it out and answers for
// it; nothing here reads anything into how it is spelled.
type ChunkID string

// Passage is what a search returns: the text around a hit, and where it came
// from. It is a read model — a chunk is not reconstructed from it.
//
// `Start` and `Length` address the large chunk enclosing the hit, in the bytes
// of the file `Source` names, and `Text` is what stands there once the file has
// been read.
type Passage struct {
	// ChunkID is the chunk a ranking named. Two rankings are merged on it.
	ChunkID ChunkID

	// Source is the path of the file the text is read from, relative to the
	// vault folder.
	Source string

	// Kind is what the vault holds there, so a caller drawing a passage draws
	// the source it came out of as what it is.
	Kind SourceKind

	Start  int
	Length int

	// Location is what the source's own numbering calls the place, and is empty
	// when the format offered none.
	Location string

	// Producer is what made the text the words are read from. Empty where the
	// source's own bytes are the text.
	Producer string

	// SourceHash addresses the content of the source, and is what the files of a
	// reading of it are kept under. Reading a passage back composes the name
	// from this and Producer.
	SourceHash string

	// HitAt is where the chunk that matched begins inside Text, in bytes. A
	// passage whose hit is the chunk itself begins at its own beginning.
	HitAt int

	// Line is where the chunk that matched stands, counted from the first line
	// of the source's prose. A note's frontmatter is not prose and is not
	// counted.
	Line int

	// ChunkHash addresses the text this chunk held when the index cut it. It is
	// what a vector made from that text is kept under and found by.
	ChunkHash string

	Text string
}
