package note

import (
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// errTooManyPaths is a filter naming more paths than are answered at once.
var errTooManyPaths = errors.New("that is more paths than are answered at once")

const maxQueryLength = 200

func eachOnce(paths []string) ([]string, error) {
	observed := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if observed[path] {
			continue
		}
		observed[path] = true
		out = append(out, path)
	}
	if len(out) > maxQueryLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooManyPaths)
	}
	return out, nil
}

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

func instant(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

// Writes tells what a write of the window did: whether the index followed it, and the embedder that a note
// holds new prose.
type Writes interface {
	// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
	IsIndexOutdated(err error) bool
	// NoteWritten tells the embedder that a note holds new prose.
	NoteWritten()
}
