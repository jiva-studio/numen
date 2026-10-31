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

	// Type is what the note is. A heading carries the type of the note
	Type NoteType

	// Heading is the heading that matched, empty when the note's own title did.
	Heading string

	// Line is the line of that heading, counted from the first line of the
	// prose. Zero for a title, which has no line in the prose.
	Line int

	// Ranges are where in the name that matched the words typed stand, counted
	// the way a client counts text: in UTF-16 code units.
	Ranges []UnitRange
}
