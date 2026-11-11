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

// VersionItem is a text version of a file that a command with items offers to run over.
type VersionItem struct {
	Version    domain.TextVersionID
	Kind       string
	IsLive     bool
	PluginID   string
	CreateTime time.Time
}

// DeleteCommand is the application command that deletes a text version of a file. The core serves
// it with no plugin: its items are every version of the file.
const DeleteCommand = "app:version.delete"
