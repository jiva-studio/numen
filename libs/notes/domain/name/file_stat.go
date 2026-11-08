package name

import "time"

// SourceKind classifies a file as a note or another file. The index stores it
// with each row, and it is derived from the file's name alone.
type SourceKind string

const (
	// KindNote is a markdown file in a vault.
	KindNote SourceKind = "note"
	// KindFile is every other regular file. Its text, if it has any, is a text
	// version a plugin made.
	KindFile SourceKind = "file"
)

// FileStat is what a walk of a vault reports before anything is read: enough to
// decide whether the file has to be read at all, and what would read it.
type FileStat struct {
	Path    string
	Kind    SourceKind
	Size    int64
	ModTime time.Time
	FileID  FileID
}

// FileID is the number a file system assigns to a file, which a rename keeps and a
// replacement of the file does not.
type FileID uint64

// IsUnchanged reports whether the file can be skipped. Size and modification
// time are the invalidation key; content is not hashed during a walk, since
// that requires reading every file to detect that nothing changed.
//
// The time is compared with Equal, since a time.Time also carries a zone and a
// monotonic reading, neither of which identifies the instant.
func (f FileStat) IsUnchanged(other FileStat) bool {
	return f.Size == other.Size && f.ModTime.Equal(other.ModTime)
}

// IsZero reports whether this is the zero file stat, which a caller passes when
// it has no expectation about the file it is about to overwrite.
func (f FileStat) IsZero() bool {
	return f.Path == "" && f.Kind == "" && f.Size == 0 && f.ModTime.IsZero()
}
