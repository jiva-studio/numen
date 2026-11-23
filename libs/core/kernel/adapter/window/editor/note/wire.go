package note

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func convertNoteTypeToWire(noteType domain.NoteType) string {
	if noteType == domain.TypeNote {
		return ""
	}
	return string(noteType)
}

func stamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func parseInstant(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

// Writes tells what a write of the window did: whether the index followed it, and the embedder that a note
// holds new prose.
type Writes struct {
	IsIndexOutdated func(error) bool
	indexing.WriteWake
}
