package domain

// An Artifact is text nobody typed into the file it belongs to: the words
// published with a video, the prose of a page, what a model heard in a
// recording. Producer is what made it, and is the word the index files it
// under.
//
// It is kept in the vault's own folder, beside the file rather than in it, so a
// file and what was made from it are never one thing.
type Artifact struct {
	Producer string
	Text     string
}

// IsZero reports whether anything has been made yet, which is the ordinary
// state of a file nothing has fetched for or listened to.
func (a Artifact) IsZero() bool { return a.Producer == "" }

// An IndexedNote is a note on its way into the index: the note as its file
// reads, and the artifact searched together with it.
//
// Only a link note has the second. It is cut with the note rather than beside
// it, because what a person wrote about an address and what is at that address
// answer one question.
type IndexedNote struct {
	Note     Note
	Artifact Artifact
}

// Indexed is notes on their way into the index with no artifact beside them,
// which is every note but a link.
func Indexed(notes ...Note) []IndexedNote {
	out := make([]IndexedNote, 0, len(notes))
	for _, one := range notes {
		out = append(out, IndexedNote{Note: one})
	}
	return out
}
