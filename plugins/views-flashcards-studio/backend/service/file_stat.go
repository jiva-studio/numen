package service

import (
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
)

// stamp returns when a file last changed, as the schema carries it. Zero is a file
// without a recorded time, and not the epoch.
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

// writeFileStat is what a file was, as the schema carries it. A file nothing is known of is
// carried as nothing.
func writeFileStat(ref domain.FileStat) *api.FileStat {
	if ref.IsZero() {
		return nil
	}
	return &api.FileStat{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

// readFileStat returns the file stat a client reports for a file when it read it.
// A missing stat is not compared.
func readFileStat(pos *api.FileStat) domain.FileStat {
	if pos == nil {
		return domain.FileStat{}
	}
	return domain.FileStat{Path: pos.Path, Size: pos.Size, ModTime: instant(pos.Mtime)}
}
