package domain

// Source is one file with text that the index holds: what the file was when it
// was read, and what read it.
//
// Hash addresses the content, and Recipe names what took the text out and the
// sizes that produced the source's chunks. Both are empty on a source nothing
// has taken text out of yet.
type Source struct {
	Fingerprint Fingerprint
	Hash        string
	Recipe      string

	// Producer is what made the text this source's chunks are places in. Empty
	// where the source's own bytes are the text, which is the ordinary case.
	//
	// It is cleared by a write that records a fingerprint alone, because a file
	// that changed is a file whose reading was of other bytes.
	Producer string
}

// Chunk is one cut of a source's text, as the index holds it. Location is where
// it sits in the terms the source's own numbering uses, and is empty where the
// format named none.
//
// Text is indexed for the words it holds and is not kept: it is the text at
// Start for Length in the source.
//
// Small are the chunks inside this one. A Chunk with none of its own is a large
// chunk all the same: what makes it large is that nothing encloses it.
type Chunk struct {
	Start    int
	Length   int
	Location string
	Text     string
	// Opens are the parts of the source that begin exactly where this chunk
	// does: what a section starting here is called. Empty for a chunk that
	// opens none, which is most of them.
	Opens []string
	Small []Chunk
}

// SourceChunks is one source as reading it left it: the file, the recipe that
// read it, and the chunks its text was cut into.
type SourceChunks struct {
	Source Source
	Chunks []Chunk
}
