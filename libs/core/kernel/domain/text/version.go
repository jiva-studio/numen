package text

import (
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ContentHash is the hash of a vault file's bytes, recorded with the file's stat at
// the time it was hashed. HashTime is when the record was written.
type ContentHash struct {
	Path     string
	Hash     string
	Size     int64
	ModTime  time.Time
	FileID   domain.FileID
	HashTime time.Time
}

// Edit is a user's edit of a version of a file: the edited segments, derived from Source in the
// view of PluginID.
type Edit struct {
	PluginID string
	Path     string
	Source   domain.TextVersionID
	Segments []domain.TextSegment
}

// File is the segments of a version file and the hash of its bytes.
type File struct {
	Segments []domain.TextSegment
	TextHash string
}
