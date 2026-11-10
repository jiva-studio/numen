package source

import "github.com/jiva-studio/numen/libs/note/domain"

// ImportedFile is one item the user supplied to this application, by the name
// the source machine gives it.
type ImportedFile struct {
	Name     string
	Handle   string
	IsFolder bool
	IsFile   bool
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

// Text is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that recognition are kept under.
type Text struct {
	domain.FileStat
	Producer    string
	ContentHash string
}
