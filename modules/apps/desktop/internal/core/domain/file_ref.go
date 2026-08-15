package domain

// FileRef is what a walk of a vault reports before anything is read: enough to
// decide whether the file has to be parsed at all.
type FileRef struct {
	// Path is relative to the vault root, always with forward slashes, so that
	// an index built on one platform describes the same note on another.
	Path  string
	Size  int64
	MTime int64
}

// Unchanged reports whether the file can be skipped. Size and modification time
// are the invalidation key; content is not hashed during a walk, because that
// would mean reading every file to discover that nothing changed.
func (f FileRef) Unchanged(other FileRef) bool {
	return f.Size == other.Size && f.MTime == other.MTime
}
