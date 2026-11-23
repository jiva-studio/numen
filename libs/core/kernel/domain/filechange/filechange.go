// Package filechange holds the values of a read of the file change feed of a vault: the latest event
// of each kind of each file, numbered in the order the index stored them.
package filechange

import (
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ErrInvalidQuery is a query over the limits of a read, or one naming an event the feed does not keep.
var ErrInvalidQuery = errors.New("invalid file change query")

// GetEvents returns the events the feed keeps.
func GetEvents() []string {
	return []string{
		manifest.EventFileAdded, manifest.EventFileChanged, manifest.EventFileMoved,
		manifest.EventFileRemoved, manifest.EventNoteFrontmatterChanged,
	}
}

// IsEvent reports whether the feed keeps the event.
func IsEvent(event string) bool { return slices.Contains(GetEvents(), event) }

// Cursor is a place in the feed of a vault: the feed and the number of the last change read. The
// zero cursor is before the start of any feed.
type Cursor struct {
	FeedID string
	Seq    int64
}

// Head is the state of the feed of a vault: its cursor after the last change, and the number up to
// which removals were dropped from it.
type Head struct {
	Cursor     Cursor
	TrimmedSeq int64
}

// IsReset reports whether a reader at since has to read the vault again from the start: since is the
// zero cursor, is of another feed, or is after the start of the feed and before changes the feed no
// longer has. A reader at the start of the feed, number 0, reads the latest events of every file the
// vault holds.
func (h Head) IsReset(since Cursor) bool {
	return since.FeedID == "" || since.FeedID != h.Cursor.FeedID || (since.Seq > 0 && since.Seq < h.TrimmedSeq)
}

// SourceID is the number the index keeps a file under. It stays the same when the file moves, and a
// file removed and added again gets another.
type SourceID int64

// Change is the latest event of one kind of one file: the source the index keeps the file as, which
// stays the same when the file moves, its path now, or where it was for a removal, and the number of the
// change. Hash, Size and ModifiedAt are those of the source now, empty and 0 for a removal and for a
// source without a hash.
type Change struct {
	NoteTypes  []domain.NoteType
	SourceID   SourceID
	Path       string
	Event      string
	Hash       string
	Seq        int64
	Size       int64
	ModifiedAt int64
}

// File identifies a changed file by its path and its indexed note type at the time of the event.
type File struct {
	Path     string
	NoteType domain.NoteType
}

// Query names the changes a read returns: those after Since of the Events, at most Limit.
type Query struct {
	Since  Cursor
	Events []string
	Limit  int
}

// Page is one page of a read: its changes, the cursor the next read starts from, whether the reader
// has to read the vault again from the start, and whether more changes follow.
type Page struct {
	Changes []Change
	Next    Cursor
	IsReset bool
	HasMore bool
}
