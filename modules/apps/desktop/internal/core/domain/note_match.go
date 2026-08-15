package domain

// NoteMatch is one note that matched a search: what it is called and where it
// is. It is a read model — a note is not reconstructed from it.
type NoteMatch struct {
	Path  string
	Title string
}
