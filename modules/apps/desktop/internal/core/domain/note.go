package domain

// Note is a parsed markdown file. Frontmatter is kept as it was found: the
// application owns a closed set of keys and preserves everything else verbatim,
// so the parser is not allowed to normalise or drop what it does not recognise.
type Note struct {
	Ref         FileRef
	Title       string
	Frontmatter map[string]any
	Headings    []Heading
	Body        string

	// FrontmatterErr is set when the block between the delimiters is not valid
	// YAML. The note is still indexed — its body is readable text either way —
	// and the file is never repaired in place, because that means guessing at
	// what the user wrote.
	FrontmatterErr string
}

// Heading is one ATX heading of a note, in document order.
type Heading struct {
	Level int
	Text  string
	Pos   int
}
