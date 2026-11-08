package domain

// NameMatch is one name in a vault that matches the typed text: a note, and the
// matching heading inside it when a heading matched, or none when the note's
// own title matched.
//
// It is a read model: a note is not reconstructed from it. A search over the
// text of a vault returns a Passage; NameMatch answers the narrower query
// issued while the user is still typing.
type NameMatch struct {
	Path  string
	Title string

	Type NoteType

	Heading string

	Line int

	Ranges []UnitRange
}
