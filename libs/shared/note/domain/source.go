package domain

// Source is one file with text that the index holds.
// ContentHash is the hash of the content, and Recipe identifies the extractor and
// the sizes that produced the chunks.
// Both are empty on a source with no extracted text yet.
type Source struct {
	FileStat    FileStat
	ContentHash string
	Recipe      string

	Producer string
}

// Chunk is one cut of a source's text, as the index holds it.
// Text is indexed but not kept: it is the text at Start for Length in the source.
// Small are the chunks inside this one; a Chunk with none is still a large chunk,
// because nothing encloses it.
type Chunk struct {
	Start    int
	Length   int
	Location string
	Text     string
	Opens    []string
	Small    []Chunk
}

// NoteChunks is one note as the index keeps it: the headings it is found by,
// and the chunks its body was cut into.
type NoteChunks struct {
	Note     Note
	Headings []Heading
	Chunks   []Chunk
	Recipe   string
}

// SourceChunks is one source as reading it left it: the file, the recipe that
// read it, and the chunks its text was cut into.
type SourceChunks struct {
	Source Source
	Chunks []Chunk
}

// A PartStart is somewhere in a source's text that carries a name. Parts bound
// the divisions a text is cut inside, and need not arrive in order.
type PartStart struct {
	Title  string
	Offset int
}
