package domain

// Passage is what a search returns: the text around a hit, and where it came
// from. It is a read model — a chunk is not reconstructed from it.
//
// `Start` and `Length` address the large window enclosing the hit, in the bytes
// of the file `Source` names, and `Text` is what stands there once the file has
// been read.
type Passage struct {
	// Chunk is the row a ranking named. Two rankings are merged on it.
	Chunk int64

	// Source is the path of the file the text is read from, relative to the
	// vault folder.
	Source string

	Start  int
	Length int

	// Location is what the source's own numbering calls the place, and is empty
	// when the format offered none.
	Location string

	// HitAt is where the chunk that matched begins inside Text, in bytes. A
	// passage whose hit is the window itself begins at its own beginning.
	HitAt int

	Text string
}
