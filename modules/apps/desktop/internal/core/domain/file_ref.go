package domain

// SourceKind says what sort of source a file is. It is the word the index files
// the row under, and it is decided from the file's name alone.
type SourceKind string

const (
	// KindNote is a markdown file in a vault.
	KindNote SourceKind = "note"
	// KindBook is an EPUB in a vault.
	KindBook SourceKind = "book"
)

// FileRef is what a walk of a vault reports before anything is read: enough to
// decide whether the file has to be read at all, and what would read it.
type FileRef struct {
	// Path is relative to the vault root, always with forward slashes, so that
	// an index built on one platform describes the same source on another.
	Path string
	// Kind is what the file is. A walk and a stat both say it; a fingerprint the
	// index hands back leaves it empty, because the kind is what was asked for.
	Kind  SourceKind
	Size  int64
	MTime int64
}

// Unchanged reports whether the file can be skipped. Size and modification time
// are the invalidation key; content is not hashed during a walk, because that
// would mean reading every file to discover that nothing changed.
func (f FileRef) Unchanged(other FileRef) bool {
	return f.Size == other.Size && f.MTime == other.MTime
}
