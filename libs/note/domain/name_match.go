package domain

// NameMatch is one name in a vault that matches the typed text: a note, and the
// matching heading inside it, or none when the note's own title matched.
// It is a read model for the query issued while the user is typing, not a SearchResult.
type NameMatch struct {
	Path  string
	Title string

	Type NoteType

	Heading string

	Line int

	Ranges []UnitRange
}
