package domain

// NoteMatch is one note whose text matched: what it is called and where it is.
// It is a read model — a note is not reconstructed from it.
//
// A search over everything the vault holds returns a Passage. This is the answer
// to the narrower question, asked when what the caller wants is the note.
type NoteMatch struct {
	Path  string
	Title string
}
