package domain

import "time"

// ModTime is when a file last changed, in nanoseconds since the Unix epoch. It
// is a stamp to compare and to order by, not an instant the domain reasons
// about: seconds put here match nothing a walk took, so every file reads as
// changed on every scan.
type ModTime int64

// ModTimeOf is the stamp for an instant, which is how an adapter holding a
// file's modification time makes one.
func ModTimeOf(t time.Time) ModTime { return ModTime(t.UnixNano()) }

// Time is the instant the stamp names, for a caller that has to say it in a
// protocol of its own.
func (m ModTime) Time() time.Time { return time.Unix(0, int64(m)) }

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
)

// Fingerprint is what a walk of a vault reports before anything is read: enough to
// decide whether the file has to be read at all, and what would read it.
type Fingerprint struct {
	// Path is relative to the vault root, always with forward slashes, so that
	// an index built on one platform describes the same source on another.
	Path string
	// Kind is what the file is. A walk and a stat both say it; a fingerprint the
	// index hands back leaves it empty, because the kind is what was asked for.
	Kind    SourceKind
	Size    int64
	ModTime ModTime
}

// Unchanged reports whether the file can be skipped. Size and modification time
// are the invalidation key; content is not hashed during a walk, because that
// would mean reading every file to discover that nothing changed.
func (f Fingerprint) Unchanged(other Fingerprint) bool {
	return f.Size == other.Size && f.ModTime == other.ModTime
}
