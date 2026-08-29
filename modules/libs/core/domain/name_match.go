package domain

// NameMatch is one name in a vault that matched what was typed: a note, and
// the heading inside it when a heading is what matched rather than the note's
// own title.
//
// It is a read model — a note is not reconstructed from it. A search over the
// text a vault holds returns a Passage; this is the answer to the narrower
// question, asked while a person is still typing.
type NameMatch struct {
	Path  string
	Title string

	// Heading is the heading that matched, empty when the note's own title did.
	Heading string

	// Line is where that heading stands, counted from the first line of the
	// prose. Zero for a title, which stands on no line of the prose at all.
	Line int

	// At is where in the name that matched the words typed stand.
	At []Span
}

// Span is a run of text, by where it begins and where it ends, counted the way
// a client counts text: in UTF-16 code units.
type Span struct {
	From int
	To   int
}
