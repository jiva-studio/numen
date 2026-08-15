// Package domain holds what the system is, with no knowledge of where any of it
// comes from. Nothing here may open a file, reach a database, or read the clock
// .
package domain

// Vault is a folder the user has added to the application. Its identity is a
// ULID carried inside the vault itself, not derived from its path, because
// folders move outside the application.
type Vault struct {
	ID   string
	Name string
	Path string
}

// FileRef is what a walk of the vault reports before anything is read: enough to
// decide whether the file has to be parsed at all.
type FileRef struct {
	// Path is relative to the vault root, always with forward slashes, so that
	// an index built on one platform describes the same note on another.
	Path  string
	Size  int64
	MTime int64
}

// Unchanged reports whether the file can be skipped. Size and modification time
// are the invalidation key; content is not hashed during a walk because that
// would mean reading every file to discover that nothing changed.
func (f FileRef) Unchanged(other FileRef) bool {
	return f.Size == other.Size && f.MTime == other.MTime
}

// Heading is one ATX heading, in document order.
type Heading struct {
	Level int
	Text  string
	Pos   int
}

// Note is a parsed markdown file. Frontmatter is kept as it was found: the
// application owns a closed set of keys and preserves everything else verbatim
// , so the parser is not allowed to normalise or drop what it does not
// recognise.
type Note struct {
	Ref         FileRef
	Title       string
	Frontmatter map[string]any
	Headings    []Heading
	Tags        []string
	Body        string

	// FrontmatterErr is set when the block between the delimiters is not valid
	// YAML. The note is still indexed — its body is readable text either way —
	// and the file is never repaired in place, because that means guessing at
	// what the user wrote.
	FrontmatterErr string
}

// Hit is one search result. Origin is recorded so that results from notes and
// from source documents can later be merged into one ranking; today
// only notes exist.
type Hit struct {
	Path    string
	Title   string
	Snippet string
}
