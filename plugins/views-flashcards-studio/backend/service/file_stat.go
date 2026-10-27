package service

import (
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// stamp is when a file last changed, as the schema carries it. Zero is a file nothing was said
// about, not the epoch.
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
func writeFileStat(ref domain.FileStat) *host.WireFileStat {
	if ref.IsZero() {
		return nil
	}
	return &host.WireFileStat{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

// readFileStat is what a client says a file was when it read it. Nothing said is nothing
// compared.
func readFileStat(pos *host.WireFileStat) domain.FileStat {
	if pos == nil {
		return domain.FileStat{}
	}
	return domain.FileStat{Path: pos.GetPath(), Size: pos.GetSize(), ModTime: instant(pos.GetMtime())}
}
