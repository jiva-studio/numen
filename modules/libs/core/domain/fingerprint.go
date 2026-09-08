package domain

import "time"

// SourceKind says what sort of source a file is. It is the word the index files
// the row under, and it is decided from the file's name alone.
type SourceKind string

const (
	// KindNote is a markdown file in a vault.
	KindNote SourceKind = "note"
	// KindBook is a source with text that nobody typed here: an EPUB, a PDF.
	// What reads one is decided from its name and recorded in its recipe.
	KindBook SourceKind = "book"
	// KindRecording is speech: an MP3, a WAV. It carries no text of its own,
	// and what it says is there once a model has listened to it.
	KindRecording SourceKind = "recording"
	// KindURL is a web address and nothing else: a `.url` file. Its text is
	// what was fetched from that address, and it holds no body to write in.
	KindURL SourceKind = "url"
)

// Fingerprint is what a walk of a vault reports before anything is read: enough to
// decide whether the file has to be read at all, and what would read it.
type Fingerprint struct {
	// Path is relative to the vault root, always with forward slashes, so that
	// an index built on one platform describes the same source on another.
	Path string
	// Kind is what the file is. A walk and a stat both say it; a fingerprint the
	// index hands back leaves it empty, because the kind is what was asked for.
	Kind SourceKind
	Size int64
	// ModTime is when the file last changed. Each adapter converts at its own
	// edge: the index and the wire carry nanoseconds since the epoch, and a
	// stamp in seconds put here matches nothing a walk took.
	ModTime time.Time
}

// Unchanged reports whether the file can be skipped. Size and modification time
// are the invalidation key; content is not hashed during a walk, because that
// would mean reading every file to discover that nothing changed.
//
// The time is compared with Equal, because a time.Time also carries a zone and
// a monotonic reading, and neither of those says which instant it is.
func (f Fingerprint) Unchanged(other Fingerprint) bool {
	return f.Size == other.Size && f.ModTime.Equal(other.ModTime)
}

// IsZero reports whether this is no fingerprint at all, which is what a caller
// presents when it says nothing about the file it expected to write over.
func (f Fingerprint) IsZero() bool {
	return f.Path == "" && f.Kind == "" && f.Size == 0 && f.ModTime.IsZero()
}
