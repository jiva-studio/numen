package domain

// NoteMatch is one note that matched a search: where it is and enough of it to
// read. It is a read model — a note is not reconstructed from it.
type NoteMatch struct {
	Path    string
	Title   string
	Snippet string
}
